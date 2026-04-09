package worker

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"Agent_Crawl/internal/classify"
	"Agent_Crawl/internal/config"
	"Agent_Crawl/internal/db"
	"Agent_Crawl/internal/extract"
	"Agent_Crawl/internal/fetcher"
	util "Agent_Crawl/internal/utils"

	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

type Worker struct {
	cfg config.Config
	clf *classify.KeywordClassifier
	f   *fetcher.Fetcher
}

func New(cfg config.Config, clf *classify.KeywordClassifier) *Worker {
	return &Worker{
		cfg: cfg,
		clf: clf,
		f:   fetcher.New(cfg),
	}
}

func (w *Worker) Run(ctx context.Context, conn *pgx.Conn, concurrency int) error {
	log.Info().Int("concurrency", concurrency).Msg("worker started")

	sem := make(chan struct{}, concurrency)
	var dbMu sync.Mutex
	var wg sync.WaitGroup

	for {
		dbMu.Lock()
		items, err := db.DequeueBatch(ctx, conn, w.cfg.Worker.BatchSize)
		dbMu.Unlock()
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
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				w.handleOne(ctx, conn, it, &dbMu)
			}()
		}
	}
}

func (w *Worker) handleOne(ctx context.Context, conn *pgx.Conn, it db.QueueItem, dbMu *sync.Mutex) {
	body, _, _, err := w.f.Get(ctx, it.URL)
	if err != nil {
		w.fail(ctx, conn, it, err.Error(), dbMu)
		return
	}

	ex, err := extract.FromHTML(it.URL, body)
	if err != nil {
		w.fail(ctx, conn, it, "extract: "+err.Error(), dbMu)
		return
	}

	cls := w.clf.Classify(ex.Title, ex.ContentText)

	// Drop if content too short (MVP quality gate)
	if len(ex.ContentText) < 200 || ex.Title == "" {
		w.fail(ctx, conn, it, "content too short or missing title", dbMu)
		return
	}

	pub := util.ParseTimeBestEffort(ex.PublishedAt)
	hash := util.HashText(ex.Title + "\n" + ex.ContentText)

	scoresJSON, _ := json.Marshal(cls.Scores)

	dbMu.Lock()
	_, err = conn.Exec(ctx, `
		INSERT INTO documents (
		  url, canonical_url, domain, source_id,
		  title, published_at, author, content_text,
		  content_hash, topic_id, topic_scores, lang
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11::jsonb,$12)
		ON CONFLICT (canonical_url) DO UPDATE SET
		  title = EXCLUDED.title,
		  published_at = COALESCE(EXCLUDED.published_at, documents.published_at),
		  author = EXCLUDED.author,
		  content_text = EXCLUDED.content_text,
		  content_hash = EXCLUDED.content_hash,
		  topic_id = EXCLUDED.topic_id,
		  topic_scores = EXCLUDED.topic_scores,
		  lang = EXCLUDED.lang,
		  updated_at = now()
	`, it.URL, ex.CanonicalURL, it.Domain, it.SourceID,
		ex.Title, pub, ex.Author, ex.ContentText,
		hash, cls.TopicID, string(scoresJSON), ex.Lang,
	)
	dbMu.Unlock()
	if err != nil {
		w.fail(ctx, conn, it, "db upsert: "+err.Error(), dbMu)
		return
	}

	dbMu.Lock()
	if err := db.MarkDone(ctx, conn, it.ID); err != nil {
		dbMu.Unlock()
		log.Warn().Err(err).Int64("queue_id", it.ID).Msg("mark done failed")
		return
	}
	dbMu.Unlock()
}

func (w *Worker) fail(ctx context.Context, conn *pgx.Conn, it db.QueueItem, msg string, dbMu *sync.Mutex) {
	next := time.Now().Add(time.Duration(w.cfg.Worker.RetryBackoffSeconds) * time.Second)
	dbMu.Lock()
	_ = db.MarkFailed(ctx, conn, it.ID, w.cfg.Worker.MaxAttempts, next, msg)
	dbMu.Unlock()
	log.Warn().
		Int64("queue_id", it.ID).
		Str("url", it.URL).
		Str("err", msg).
		Msg("task failed")
}
