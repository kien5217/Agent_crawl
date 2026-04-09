package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// QueueItem represents an item in the crawl queue. It contains the ID of the queue entry, the associated topic and source IDs, the URL to be crawled, the domain of the URL, and the number of attempts made to crawl it.
type QueueItem struct {
	ID       int64
	TopicID  string
	SourceID string
	URL      string
	Domain   string
	Attempts int
}

// EnqueueURL adds a new URL to the crawl queue with the specified topic ID, source ID, URL, domain, and priority. It returns true if the URL was successfully enqueued (i.e., it was not already in the queue), or false if it was a duplicate and was not added.
func EnqueueURL(ctx context.Context, conn *pgx.Conn, topicID, sourceID, url, domain string, priority int) (bool, error) {
	ct, err := conn.Exec(ctx, `
		INSERT INTO crawl_queue (topic_id, source_id, url, domain, priority, status, next_run_at)
		VALUES ($1, $2, $3, $4, $5, 'pending', now())
		ON CONFLICT (url) DO NOTHING
	`, topicID, sourceID, url, domain, priority)
	if err != nil {
		return false, err
	}
	return ct.RowsAffected() == 1, nil
}

// DequeueBatch retrieves a batch of pending queue items that are ready to be processed. It locks the selected rows to prevent other workers from processing the same items concurrently. The items are ordered by priority (descending) and ID (ascending) to ensure that higher priority items are processed first. After selecting the items, it updates their status to 'processing' and returns the list of QueueItem structs.
func DequeueBatch(ctx context.Context, conn *pgx.Conn, batchSize int) ([]QueueItem, error) {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, `
		SELECT id, topic_id, source_id, url, domain, attempts
		FROM crawl_queue
		WHERE status = 'pending'
		  AND next_run_at <= now()
		ORDER BY priority DESC, id ASC
		FOR UPDATE SKIP LOCKED
		LIMIT $1
	`, batchSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []QueueItem
	for rows.Next() {
		var it QueueItem
		if err := rows.Scan(&it.ID, &it.TopicID, &it.SourceID, &it.URL, &it.Domain, &it.Attempts); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	// mark as processing
	if len(items) > 0 {
		ids := make([]int64, 0, len(items))
		for _, it := range items {
			ids = append(ids, it.ID)
		}
		_, err = tx.Exec(ctx, `
			UPDATE crawl_queue
			SET status = 'processing', updated_at = now()
			WHERE id = ANY($1)
		`, ids)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return items, nil
}

func MarkDone(ctx context.Context, conn *pgx.Conn, id int64) error {
	_, err := conn.Exec(ctx, `
		UPDATE crawl_queue
		SET status = 'done', updated_at = now()
		WHERE id = $1
	`, id)
	return err
}

func MarkFailed(ctx context.Context, conn *pgx.Conn, id int64, attempts int, nextRunAt time.Time, lastErr string) error {
	_, err := conn.Exec(ctx, `
		UPDATE crawl_queue
		SET status = CASE WHEN $2 >= attempts THEN 'failed' ELSE 'pending' END,
		    attempts = attempts + 1,
		    next_run_at = $3,
		    last_error = $4,
		    updated_at = now()
		WHERE id = $1
	`, id, attempts, nextRunAt, lastErr)
	return err
}
