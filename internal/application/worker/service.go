package worker

import (
	"context"
	"encoding/json"
	"time"

	"Agent_Crawl/internal/domain/config"
	"Agent_Crawl/internal/domain/model"
	"Agent_Crawl/internal/domain/repository"
	"Agent_Crawl/internal/infrastructure/classify"
	"Agent_Crawl/internal/infrastructure/extract"
	"Agent_Crawl/internal/infrastructure/fetcher"
	util "Agent_Crawl/internal/platform"

	"github.com/rs/zerolog/log"
)

type Worker struct {
	cfg config.Config
	clf *classify.KeywordClassifier
	f   *fetcher.Fetcher
	q   repository.QueueRepository
	doc repository.CrawlWriteRepository
}

func New(cfg config.Config, clf *classify.KeywordClassifier, q repository.QueueRepository, doc repository.CrawlWriteRepository) *Worker {
	return &Worker{
		cfg: cfg,
		clf: clf,
		f:   fetcher.New(cfg),
		q:   q,
		doc: doc,
	}
}

func (w *Worker) Run(ctx context.Context, concurrency int) error {
	log.Info().Int("concurrency", concurrency).Msg("worker started")

	sem := make(chan struct{}, concurrency)

	for {
		items, err := w.q.DequeueBatch(ctx, w.cfg.Worker.BatchSize)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			time.Sleep(2 * time.Second)
			continue
		}

		for _, it := range items {
			it := it
			sem <- struct{}{}
			go func() {
				defer func() { <-sem }()
				w.handleOne(ctx, it)
			}()
		}
	}
}

func (w *Worker) handleOne(ctx context.Context, it model.QueueItem) {
	body, _, _, err := w.f.Get(ctx, it.URL)
	if err != nil {
		w.fail(ctx, it, err.Error())
		return
	}

	ex, err := extract.FromHTML(it.URL, body)
	if err != nil {
		w.fail(ctx, it, "extract: "+err.Error())
		return
	}

	cls := w.clf.Classify(ex.Title, ex.ContentText)

	// Drop if content too short (MVP quality gate)
	if len(ex.ContentText) < 200 || ex.Title == "" {
		w.fail(ctx, it, "content too short or missing title")
		return
	}

	pub := util.ParseTimeBestEffort(ex.PublishedAt)
	hash := util.HashText(ex.Title + "\n" + ex.ContentText)

	scoresJSON, _ := json.Marshal(cls.Scores)

	err = w.doc.UpsertCrawledDocument(ctx, model.CrawledDocument{
		URL:             it.URL,
		CanonicalURL:    ex.CanonicalURL,
		Domain:          it.Domain,
		SourceID:        it.SourceID,
		Title:           ex.Title,
		PublishedAt:     pub,
		Author:          ex.Author,
		ContentText:     ex.ContentText,
		ContentHash:     hash,
		TopicID:         cls.TopicID,
		TopicScoresJSON: string(scoresJSON),
		Lang:            ex.Lang,
	})
	if err != nil {
		w.fail(ctx, it, "db upsert: "+err.Error())
		return
	}

	if err := w.q.MarkDone(ctx, it.ID); err != nil {
		log.Warn().Err(err).Int64("queue_id", it.ID).Msg("mark done failed")
		return
	}
}

func (w *Worker) fail(ctx context.Context, it model.QueueItem, msg string) {
	next := time.Now().Add(time.Duration(w.cfg.Worker.RetryBackoffSeconds) * time.Second)
	_ = w.q.MarkFailed(ctx, it.ID, w.cfg.Worker.MaxAttempts, next, msg)
	log.Warn().
		Int64("queue_id", it.ID).
		Str("url", it.URL).
		Str("err", msg).
		Msg("task failed")
}
