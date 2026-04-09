package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"Agent_Crawl/internal/classify"
	"Agent_Crawl/internal/config"
	"Agent_Crawl/internal/db"
	"Agent_Crawl/internal/discovery"
	"Agent_Crawl/internal/worker"

	"strconv"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

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
	default:
		log.Fatal().Str("cmd", cmd).Msg("unknown command")
	}
}
