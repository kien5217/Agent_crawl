package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"Agent_Crawl/internal/application/api"
	"Agent_Crawl/internal/application/cli"
	"Agent_Crawl/internal/application/loader"
	orchestration "Agent_Crawl/internal/application/orchestrator"
	appschedule "Agent_Crawl/internal/application/schedule"
	config "Agent_Crawl/internal/domain/config"
	"Agent_Crawl/internal/infrastructure/discovery"
	"Agent_Crawl/internal/infrastructure/persistence/postgres"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type clsProb struct {
	Class string
	Prob  float64
}

type commandOptions struct {
	configPath  string
	concurrency int
	limit       int
	minWeakConf float64
	modelName   string
	modelVer    int
	batchSize   int
	classesCSV  string
	pTopic      string
	pK          int
	pWrite      bool
	apiAddr     string
	args        []string
}

type runtime struct {
	appCfg *config.AppConfig
	db     *pgxpool.Pool
	store  *postgres.Store
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

	cmd, opts := parseCLI(os.Args)
	ctx := context.Background()
	rt, err := initRuntime(ctx, opts.configPath)
	if err != nil {
		log.Fatal().Err(err).Msg("bootstrap failed")
	}
	defer rt.db.Close()

	runCommand(ctx, cmd, opts, rt)
}

func parseCLI(args []string) (string, commandOptions) {
	if len(args) < 2 {
		fmt.Println("usage: crawler <migrate|schedule|worker|list|show|serve|api> --config ./config/config.yaml")
		os.Exit(2)
	}

	cmd := args[1]
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
	apiAddr := fs.String("addr", ":8080", "HTTP listen address (only for api cmd)")
	_ = fs.Parse(args[2:])

	return cmd, commandOptions{
		configPath:  *configPath,
		concurrency: *concurrency,
		limit:       *limit,
		minWeakConf: *minWeakConf,
		modelName:   *modelName,
		modelVer:    *modelVersion,
		batchSize:   *batchSize,
		classesCSV:  *classesCSV,
		pTopic:      *pTopic,
		pK:          *pK,
		pWrite:      *pWrite,
		apiAddr:     *apiAddr,
		args:        fs.Args(),
	}
}

func initRuntime(ctx context.Context, configPath string) (*runtime, error) {
	appCfg, err := loader.LoadAll(configPath, "./config/topics.yaml", "./config/sources.yaml")
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	databaseURL := os.ExpandEnv(appCfg.Config.DatabaseURL)
	appCfg.Config.Auth.APIKey = os.ExpandEnv(appCfg.Config.Auth.APIKey)
	appCfg.Config.Auth.JWTSecret = os.ExpandEnv(appCfg.Config.Auth.JWTSecret)
	db, err := postgres.Open(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	return &runtime{appCfg: appCfg, db: db, store: postgres.NewStore(db)}, nil
}

func runCommand(ctx context.Context, cmd string, opts commandOptions, rt *runtime) {
	// HTTP API server command handled separately (does not use CLI handlers map)
	if cmd == "api" {
		scheduler := appschedule.NewService(
			discovery.NewRSSDiscovery(rt.appCfg.Config, rt.appCfg.Topics, rt.appCfg.Sources, rt.store),
			discovery.NewSitemapDiscovery(rt.appCfg.Config, rt.appCfg.Topics, rt.appCfg.Sources, rt.store),
		)
		afterSchedule := func(ctx context.Context) error {
			const workerTimeout = 180 * time.Second

			classes := splitCSV(opts.classesCSV)
			if len(classes) == 0 {
				return errors.New("classes must not be empty")
			}

			workerStep := orchestration.NewWorkerStep(rt.appCfg, rt.store, rt.store, opts.concurrency)
			workerCtx, cancelWorker := context.WithTimeout(ctx, workerTimeout)
			_, err := workerStep.Run(workerCtx)
			cancelWorker()
			if err != nil && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
				return fmt.Errorf("%s failed: %w", workerStep.Name(), err)
			}

			steps := []orchestration.Step{
				orchestration.NewWeakLabelStep(rt.store, opts.limit),
				orchestration.NewTrainStep(rt.store, rt.store, classes, float32(opts.minWeakConf), opts.modelName, opts.modelVer),
				orchestration.NewSelectStep(rt.store, rt.store, opts.modelName, opts.limit, opts.batchSize),
				orchestration.NewPredictStep(rt.store, rt.store, opts.modelName, opts.pTopic, opts.limit, opts.pWrite),
			}

			for _, step := range steps {
				if _, err := step.Run(ctx); err != nil {
					return fmt.Errorf("%s failed: %w", step.Name(), err)
				}
			}
			return nil
		}

		srv := api.NewServer(opts.apiAddr, rt.appCfg, rt.store, rt.store, rt.store, scheduler, afterSchedule)
		if err := srv.Run(); err != nil {
			log.Fatal().Err(err).Msg("api server failed")
		}
		return
	}

	handlers := cli.NewHandlers(cli.Executor{
		AppCfg:    rt.appCfg,
		Bootstrap: rt.store,
		Migrate:   rt.store,
		Scheduler: appschedule.NewService(
			discovery.NewRSSDiscovery(rt.appCfg.Config, rt.appCfg.Topics, rt.appCfg.Sources, rt.store),
			discovery.NewSitemapDiscovery(rt.appCfg.Config, rt.appCfg.Topics, rt.appCfg.Sources, rt.store),
		),
		Queue:    rt.store,
		Document: rt.store,
		CrawlDoc: rt.store,
		Learning: rt.store,
		Model:    rt.store,
	})
	handler, ok := handlers[cmd]
	if !ok {
		log.Fatal().Str("cmd", cmd).Msg("unknown command")
	}
	handler(ctx, cli.Options{
		Concurrency: opts.concurrency,
		Limit:       opts.limit,
		MinWeakConf: opts.minWeakConf,
		ModelName:   opts.modelName,
		ModelVer:    opts.modelVer,
		BatchSize:   opts.batchSize,
		ClassesCSV:  opts.classesCSV,
		PTopic:      opts.pTopic,
		PK:          opts.pK,
		PWrite:      opts.pWrite,
		Args:        opts.args,
	})
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
