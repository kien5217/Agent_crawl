package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"Agent_Crawl/internal/application/learning"
	appschedule "Agent_Crawl/internal/application/schedule"
	"Agent_Crawl/internal/application/worker"
	config "Agent_Crawl/internal/domain/config"
	"Agent_Crawl/internal/domain/model"
	"Agent_Crawl/internal/domain/repository"
	"Agent_Crawl/internal/infrastructure/classify"
	ml "Agent_Crawl/internal/infrastructure/machine_learning"

	"github.com/rs/zerolog/log"
)

type Options struct {
	Concurrency int
	Limit       int
	MinWeakConf float64
	ModelName   string
	ModelVer    int
	BatchSize   int
	ClassesCSV  string
	PTopic      string
	PK          int
	PWrite      bool
	Args        []string
}

type Handler func(context.Context, Options)

type Executor struct {
	AppCfg    *config.AppConfig
	Bootstrap repository.BootstrapRepository
	Migrate   repository.MigrationRepository
	Scheduler *appschedule.Service
	Queue     repository.QueueRepository
	Document  repository.DocumentRepository
	CrawlDoc  repository.CrawlWriteRepository
	Learning  repository.LearningRepository
	Model     repository.ModelRepository
}

type clsProb struct {
	Class string
	Prob  float64
}

func NewHandlers(exec Executor) map[string]Handler {
	return map[string]Handler{
		"migrate":    exec.handleMigrate,
		"schedule":   exec.handleSchedule,
		"worker":     exec.handleWorker,
		"list":       exec.handleList,
		"show":       exec.handleShow,
		"weak_label": exec.handleWeakLabel,
		"train":      exec.handleTrain,
		"select":     exec.handleSelect,
		"predict":    exec.handlePredict,
	}
}

func (e Executor) handleMigrate(ctx context.Context, _ Options) {
	if err := e.Migrate.Migrate(ctx, "./migrations"); err != nil {
		log.Fatal().Err(err).Msg("migrate failed")
	}
	if err := e.Bootstrap.UpsertTopics(ctx, e.AppCfg.Topics); err != nil {
		log.Fatal().Err(err).Msg("upsert topics failed")
	}
	if err := e.Bootstrap.UpsertSources(ctx, e.AppCfg.Sources); err != nil {
		log.Fatal().Err(err).Msg("upsert sources failed")
	}
	log.Info().Msg("migrate done")
}

func (e Executor) handleSchedule(ctx context.Context, _ Options) {
	result, err := e.Scheduler.Run(ctx)
	if err != nil {
		var discovererErr *appschedule.DiscovererError
		if errors.As(err, &discovererErr) {
			log.Fatal().Err(discovererErr.Err).Str("discoverer", discovererErr.Discoverer).Msg("schedule failed")
		}
		log.Fatal().Err(err).Msg("schedule failed")
	}
	log.Info().
		Int("rss_enqueued", result.Counts["rss"]).
		Int("sitemap_enqueued", result.Counts["sitemap"]).
		Msg("schedule done")
}

func (e Executor) handleWorker(ctx context.Context, opts Options) {
	clf := classify.NewKeywordClassifier(e.AppCfg.Topics, e.AppCfg.Config.Classify.MinScoreToAccept)
	w := worker.New(e.AppCfg.Config, clf, e.Queue, e.CrawlDoc)
	if err := w.Run(ctx, opts.Concurrency); err != nil {
		log.Fatal().Err(err).Msg("worker failed")
	}
}

func (e Executor) handleList(ctx context.Context, opts Options) {
	topic := "all"
	limit := 20
	if len(opts.Args) >= 1 {
		topic = opts.Args[0]
	}
	if len(opts.Args) >= 2 {
		if v, err := strconv.Atoi(opts.Args[1]); err == nil {
			limit = v
		}
	}

	docs, err := e.Document.ListDocuments(ctx, repository.DocumentListFilter{TopicID: topic, Limit: limit})
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
}

func (e Executor) handleShow(ctx context.Context, opts Options) {
	if len(opts.Args) < 1 {
		log.Fatal().Msg("usage: crawler show <doc_id>")
	}
	id, err := strconv.ParseInt(opts.Args[0], 10, 64)
	if err != nil {
		log.Fatal().Err(err).Msg("invalid id")
	}

	d, err := e.Document.GetDocumentByID(ctx, id)
	if err != nil {
		log.Fatal().Err(err).Msg("show failed")
	}
	pub := ""
	if d.PublishedAt != nil {
		pub = d.PublishedAt.Format(time.RFC3339)
	}
	fmt.Printf("ID: %d\nTopic: %s\nTitle: %s\nURL: %s\nPublished: %s\n\n%s\n",
		d.ID, d.TopicID, d.Title, d.URL, pub, d.ContentText)
}

func (e Executor) handleWeakLabel(ctx context.Context, opts Options) {
	wl := learning.NewWeakLabeler()
	docs, err := e.Learning.ListDocsForWeakLabel(ctx, opts.Limit)
	if err != nil {
		log.Fatal().Err(err).Msg("weak-label: list docs failed")
	}
	applied := learning.ApplyWeakLabels(docs, wl)

	written := 0
	for _, a := range applied {
		if err := e.Learning.UpsertWeakLabel(ctx, model.WeakLabel{DocumentID: a.DocID, TopicID: a.TopicID, Confidence: a.Confidence, RuleID: a.RuleID}); err != nil {
			log.Warn().Err(err).Int64("doc_id", a.DocID).Msg("weak-label: upsert failed")
			continue
		}
		written++
	}
	log.Info().Int("docs_scanned", len(docs)).Int("weak_labels_written", written).Msg("weak-label done")
}

func (e Executor) handleTrain(ctx context.Context, opts Options) {
	classes := splitCSV(opts.ClassesCSV)
	if len(classes) == 0 {
		log.Fatal().Msg("train: classes is empty")
	}

	trainDocs, err := e.Learning.ListTrainingSet(ctx, float32(opts.MinWeakConf), 50000)
	if err != nil {
		log.Fatal().Err(err).Msg("train: list training set failed")
	}

	bundle, stats := learning.TrainFromDocs(trainDocs, classes, 3)
	blob, err := bundle.Marshal()
	if err != nil {
		log.Fatal().Err(err).Msg("train: marshal model failed")
	}
	classesJSON, _ := json.Marshal(classes)

	ver := opts.ModelVer
	if ver == 0 {
		ver, err = e.Model.NextVersion(ctx, opts.ModelName)
		if err != nil {
			log.Fatal().Err(err).Msg("train: get next model version failed")
		}
	}

	if err := e.Model.SaveModel(ctx, opts.ModelName, ver, classesJSON, blob); err != nil {
		log.Fatal().Err(err).Msg("train: save model failed")
	}
	log.Info().
		Str("model", opts.ModelName).
		Int("version", ver).
		Int("samples", stats.NumSamples).
		Int("classes", stats.NumClasses).
		Int("vocab", stats.VocabSize).
		Msg("train done")
}

func (e Executor) handleSelect(ctx context.Context, opts Options) {
	_, blob, err := e.Model.LoadLatestModel(ctx, opts.ModelName)
	if err != nil {
		log.Fatal().Err(err).Msg("select: load latest model failed")
	}
	bundle, err := ml.Unmarshal(blob)
	if err != nil {
		log.Fatal().Err(err).Msg("select: unmarshal model failed")
	}

	ids, titles, contents, err := e.Learning.ListUnlabeledDocs(ctx, opts.Limit)
	if err != nil {
		log.Fatal().Err(err).Msg("select: list unlabeled docs failed")
	}
	if len(ids) == 0 {
		log.Info().Msg("select: no unlabeled docs found")
		return
	}

	pickedIDs := learning.SelectBatchForLabeling(bundle, ids, titles, contents, opts.BatchSize)
	picks := learning.ComputeMargins(bundle, ids, titles, contents)
	marginByID := map[int64]float32{}
	for _, p := range picks {
		marginByID[p.DocID] = float32(p.Margin)
	}

	written := 0
	for _, id := range pickedIDs {
		if err := e.Learning.EnqueueLabelQueue(ctx, model.LabelQueueItem{DocumentID: id, Reason: "active:margin+diversity", Margin: marginByID[id]}); err != nil {
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
}

func (e Executor) handlePredict(ctx context.Context, opts Options) {
	ver, blob, err := e.Model.LoadLatestModel(ctx, opts.ModelName)
	if err != nil {
		log.Fatal().Err(err).Msg("predict: load model failed")
	}
	bundle, err := ml.Unmarshal(blob)
	if err != nil {
		log.Fatal().Err(err).Msg("predict: unmarshal model failed")
	}

	docs, err := e.Document.ListDocuments(ctx, repository.DocumentListFilter{TopicID: opts.PTopic, Limit: opts.Limit})
	if err != nil {
		log.Fatal().Err(err).Msg("predict: list documents failed")
	}
	if len(docs) == 0 {
		log.Info().Msg("predict: no documents found")
		return
	}

	now := time.Now()
	for _, d := range docs {
		x := bundle.Vectorizer.Transform(d.Title + "\n" + d.ContentText)
		p := bundle.Model.PredictProba(x)

		best := topK(bundle.Model.Classes, p, opts.PK)
		mlTopic := best[0].Class
		mlConf := float32(best[0].Prob)

		fmt.Printf("[%d] topic=%s  title=%s\n", d.ID, d.TopicID, d.Title)
		fmt.Printf("  url: %s\n", d.URL)
		for _, b := range best {
			fmt.Printf("  pred: %-16s %.4f\n", b.Class, b.Prob)
		}

		if opts.PWrite {
			scores := map[string]float64{}
			for i, c := range bundle.Model.Classes {
				if i < len(p) {
					scores[c] = p[i]
				}
			}
			b, _ := json.Marshal(scores)
			if err := e.Document.UpdateDocumentML(ctx, model.PredictedDocumentML{
				DocumentID:    d.ID,
				ModelName:     opts.ModelName,
				ModelVersion:  ver,
				MLTopicID:     mlTopic,
				MLConfidence:  mlConf,
				MLScoresJSON:  string(b),
				MLPredictedAt: now,
			}); err != nil {
				log.Warn().Err(err).Int64("doc_id", d.ID).Msg("predict: update documents.ml_* failed")
			}
		}
	}
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
