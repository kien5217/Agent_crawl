package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"Agent_Crawl/internal/classify"
	"Agent_Crawl/internal/config"
	"Agent_Crawl/internal/db"
	"Agent_Crawl/internal/discovery"
	"Agent_Crawl/internal/learning"
	modelbundle "Agent_Crawl/internal/machine_learning/model_bundle"
	"Agent_Crawl/internal/worker"

	"strconv"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type clsProb struct {
	Class string
	Prob  float64
}

func topK(classes []string, p []float64, k int) []clsProb {
	if k <= 0 {
		k = 3
	}
	out := make([]clsProb, 0, len(p))
	for i := range p {
		c := "?"
		if i < len(classes) {
			c = classes[i]
		}
		out = append(out, clsProb{Class: c, Prob: p[i]})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Prob > out[j].Prob })
	if k > len(out) {
		k = len(out)
	}
	return out[:k]
}

func main() {
	zerolog.TimeFieldFormat = time.RFC3339

	if len(os.Args) < 2 {
		fmt.Println("usage: crawler <migrate|schedule|worker|list|show|serve> --config ./config/config.yaml")
		os.Exit(2)
	}

	cmd := os.Args[1]
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	configPath := fs.String("config", "./config/config.yaml", "path to config.yaml")
	concurrency := fs.Int("concurrency", 20, "worker concurrency (only for worker cmd)")
	limit := fs.Int("limit", 5000, "limit for learning/predict commands")
	minWeakConf := fs.Float64("min-weak-conf", 0.85, "min weak label confidence to include in training")
	modelName := fs.String("model-name", "tfidf_lr", "model name (stored in db.models)")
	modelVersion := fs.Int("model-version", 0, "model version to save (0 = auto increment)")
	batchSize := fs.Int("batch", 50, "active learning batch size (select)")
	classesCSV := fs.String("classes", "ai,security,cloud,programming,blockchain,cve", "comma-separated class list for multi-class training")
	pTopic := fs.String("topic", "all", "filter by documents.topic_id (or all)")
	pK := fs.Int("k", 3, "top-k classes to show")
	pWrite := fs.Bool("write", true, "write prediction back to documents.ml_*")
	_ = fs.Parse(os.Args[2:])

	ctx := context.Background()

	appCfg, err := config.LoadAll(*configPath, "./config/topics.yaml", "./config/sources.yaml")
	if err != nil {
		log.Fatal().Err(err).Msg("load config failed")
	}

	databaseURL := os.ExpandEnv(appCfg.Config.DatabaseURL)
	conn, err := db.Open(ctx, databaseURL)
	if err != nil {
		log.Fatal().Err(err).Msg("db open failed")
	}
	defer conn.Close(ctx)

	switch cmd {
	case "migrate":
		if err := db.Migrate(ctx, conn, "./migrations"); err != nil {
			log.Fatal().Err(err).Msg("migrate failed")
		}
		// upsert topics & sources for reference
		if err := db.UpsertTopics(ctx, conn, appCfg.Topics); err != nil {
			log.Fatal().Err(err).Msg("upsert topics failed")
		}
		if err := db.UpsertSources(ctx, conn, appCfg.Sources); err != nil {
			log.Fatal().Err(err).Msg("upsert sources failed")
		}
		log.Info().Msg("migrate done")
	case "schedule":
		disc := discovery.NewRSSDiscovery(appCfg.Config, appCfg.Sources)
		n1, err := disc.Enqueue(ctx, conn)
		if err != nil {
			log.Fatal().Err(err).Msg("rss schedule failed")
		}
		sm := discovery.NewSitemapDiscovery(appCfg.Config, appCfg.Sources)
		n2, err := sm.Enqueue(ctx, conn)
		if err != nil {
			log.Fatal().Err(err).Msg("sitemap schedule failed")
		}
		log.Info().Int("rss_enqueued", n1).Int("sitemap_enqueued", n2).Msg("schedule done")
	case "worker":
		clf := classify.NewKeywordClassifier(appCfg.Topics, appCfg.Config.Classify.MinScoreToAccept)
		w := worker.New(appCfg.Config, clf)
		if err := w.Run(ctx, conn, *concurrency); err != nil {
			log.Fatal().Err(err).Msg("worker failed")
		}
	case "list":
		// flags riêng (nhẹ): đọc từ args còn lại
		topic := "all"
		limit := 20
		if fs.NArg() >= 1 {
			topic = fs.Arg(0) // vd: cve
		}
		if fs.NArg() >= 2 {
			if v, err := strconv.Atoi(fs.Arg(1)); err == nil {
				limit = v
			}
		}
		docs, err := db.ListDocuments(ctx, conn, topic, limit)
		if err != nil {
			log.Fatal().Err(err).Msg("list failed")
		}
		for _, d := range docs {
			pub := ""
			if d.PublishedAt != nil {
				pub = d.PublishedAt.Format(time.RFC3339)
			}
			fmt.Printf("[%d] (%s) %s\n  %s\n", d.ID, d.TopicID, d.Title, d.URL)
			if pub != "" {
				fmt.Printf("  published_at: %s\n", pub)
			}
		}

	case "show":
		if fs.NArg() < 1 {
			log.Fatal().Msg("usage: crawler show <doc_id>")
		}
		id, err := strconv.ParseInt(fs.Arg(0), 10, 64)
		if err != nil {
			log.Fatal().Err(err).Msg("invalid id")
		}
		d, err := db.GetDocumentByID(ctx, conn, id)
		if err != nil {
			log.Fatal().Err(err).Msg("show failed")
		}
		pub := ""
		if d.PublishedAt != nil {
			pub = d.PublishedAt.Format(time.RFC3339)
		}
		fmt.Printf("ID: %d\nTopic: %s\nTitle: %s\nURL: %s\nPublished: %s\n\n%s\n",
			d.ID, d.TopicID, d.Title, d.URL, pub, d.ContentText)

	case "weak_label":
		wl := learning.NewWeakLabeler()
		docs, err := db.ListDocsForWeakLabel(ctx, conn, *limit)
		if err != nil {
			log.Fatal().Err(err).Msg("weak-label: list docs failed")
		}
		applied := learning.ApplyWeakLabels(docs, wl)

		n := 0
		for _, a := range applied {
			if err := db.UpsertWeakLabel(ctx, conn, a.DocID, a.TopicID, a.Confidence, a.RuleID); err != nil {
				log.Warn().Err(err).Int64("doc_id", a.DocID).Msg("weak-label: upsert failed")
				continue
			}
			n++
		}
		log.Info().Int("docs_scanned", len(docs)).Int("weak_labels_written", n).Msg("weak-label done")

	case "train":
		classes := splitCSV(*classesCSV)
		if len(classes) == 0 {
			log.Fatal().Msg("train: classes is empty")
		}
		trainDocs, err := db.ListTrainingSet(ctx, conn, float32(*minWeakConf), 50000)
		if err != nil {
			log.Fatal().Err(err).Msg("train: list training set failed")
		}

		bundle, stats := learning.TrainFromDocs(trainDocs, classes, 3)

		blob, err := bundle.Marshal()
		if err != nil {
			log.Fatal().Err(err).Msg("train: marshal model failed")
		}
		classesJSON, _ := json.Marshal(classes)

		ver := *modelVersion
		if ver == 0 {
			// auto increment: get max(version)+1
			var maxVer int
			_ = conn.QueryRow(ctx, `
				SELECT COALESCE(MAX(version), 0) FROM models WHERE name=$1
			`, *modelName).Scan(&maxVer)
			ver = maxVer + 1
		}

		if err := db.SaveModel(ctx, conn, *modelName, ver, classesJSON, blob); err != nil {
			log.Fatal().Err(err).Msg("train: save model failed")
		}
		log.Info().
			Str("model", *modelName).
			Int("version", ver).
			Int("samples", stats.NumSamples).
			Int("classes", stats.NumClasses).
			Int("vocab", stats.VocabSize).
			Msg("train done")

	case "select":
		_, blob, err := db.LoadLatestModel(ctx, conn, *modelName)
		if err != nil {
			log.Fatal().Err(err).Msg("select: load latest model failed")
		}
		bundle, err := modelbundle.Unmarshal(blob)
		if err != nil {
			log.Fatal().Err(err).Msg("select: unmarshal model failed")
		}

		ids, titles, contents, err := db.ListUnlabeledDocs(ctx, conn, *limit)
		if err != nil {
			log.Fatal().Err(err).Msg("select: list unlabeled docs failed")
		}
		if len(ids) == 0 {
			log.Info().Msg("select: no unlabeled docs found")
			return
		}

		// Pick batch using margin + diversity
		pickedIDs := learning.SelectBatchForLabeling(conn, bundle, ids, titles, contents, *batchSize)

		// Compute margins for saving (cheap enough for current limit; can optimize later)
		picks := learning.ComputeMargins(bundle, ids, titles, contents)
		marginByID := map[int64]float32{}
		for _, p := range picks {
			marginByID[p.DocID] = float32(p.Margin)
		}

		written := 0
		for _, id := range pickedIDs {
			margin := marginByID[id]
			if err := db.EnqueueLabelQueue(ctx, conn, id, "active:margin+diversity", margin); err != nil {
				log.Warn().Err(err).Int64("doc_id", id).Msg("select: enqueue label queue failed")
				continue
			}
			written++
		}

		log.Info().
			Int("unlabeled_scanned", len(ids)).
			Int("picked", len(pickedIDs)).
			Int("label_queue_written", written).
			Msg("select done")

	case "predict":
		// load latest model
		ver, blob, err := db.LoadLatestModel(ctx, conn, *modelName)
		if err != nil {
			log.Fatal().Err(err).Msg("predict: load model failed")
		}
		bundle, err := modelbundle.Unmarshal(blob)
		if err != nil {
			log.Fatal().Err(err).Msg("predict: unmarshal model failed")
		}

		docs, err := db.ListDocuments(ctx, conn, *pTopic, *limit)
		if err != nil {
			log.Fatal().Err(err).Msg("predict: list documents failed")
		}
		if len(docs) == 0 {
			log.Info().Msg("predict: no documents found")
			return
		}

		type clsProb struct {
			Class string
			Prob  float64
		}
		topK := func(classes []string, p []float64, k int) []clsProb {
			out := make([]clsProb, 0, len(p))
			for i := range p {
				c := "?"
				if i < len(classes) {
					c = classes[i]
				}
				out = append(out, clsProb{Class: c, Prob: p[i]})
			}
			sort.Slice(out, func(i, j int) bool { return out[i].Prob > out[j].Prob })
			if k <= 0 {
				k = 3
			}
			if k > len(out) {
				k = len(out)
			}
			return out[:k]
		}

		now := time.Now()

		for _, d := range docs {
			// predict
			x := bundle.Vectorizer.Transform(d.Title + "\n" + d.ContentText)
			p := bundle.Model.PredictProba(x)

			best := topK(bundle.Model.Classes, p, *pK)
			mlTopic := best[0].Class
			mlConf := float32(best[0].Prob)

			// show
			fmt.Printf("[%d] topic=%s  title=%s\n", d.ID, d.TopicID, d.Title)
			fmt.Printf("  url: %s\n", d.URL)
			for _, b := range best {
				fmt.Printf("  pred: %-16s %.4f\n", b.Class, b.Prob)
			}

			// write back
			if *pWrite {
				// store full score vector as json object {class: prob}
				m := map[string]float64{}
				for i, c := range bundle.Model.Classes {
					if i < len(p) {
						m[c] = p[i]
					}
				}
				b, _ := json.Marshal(m)
				if err := db.UpdateDocumentML(ctx, conn, d.ID, *modelName, ver, mlTopic, mlConf, string(b), now); err != nil {
					log.Warn().Err(err).Int64("doc_id", d.ID).Msg("predict: update documents.ml_* failed")
				}
			}
		}

	default:
		log.Fatal().Str("cmd", cmd).Msg("unknown command")
	}
}

func splitCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := parts[:0]
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
