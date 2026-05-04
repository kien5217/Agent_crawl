package postgres

import (
	"context"
	"time"

	"Agent_Crawl/internal/domain/model"
)

// ListPendingLabelQueue returns pending items ordered by margin ASC (most uncertain first).
func ListPendingLabelQueue(ctx context.Context, db DB, limit int) ([]model.LabelQueueEntry, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	rows, err := db.Query(ctx, `
		SELECT lq.id, lq.document_id, lq.status, lq.reason, lq.margin, lq.created_at,
		       d.title, d.url, d.topic_id, d.content_text
		FROM label_queue lq
		JOIN documents d ON d.id = lq.document_id
		WHERE lq.status = 'pending'
		ORDER BY lq.margin ASC, lq.created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.LabelQueueEntry
	for rows.Next() {
		var e model.LabelQueueEntry
		if err := rows.Scan(
			&e.ID, &e.DocumentID, &e.Status, &e.Reason, &e.Margin, &e.CreatedAt,
			&e.Title, &e.URL, &e.TopicID, &e.ContentText,
		); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// SubmitLabel writes a gold label and marks the queue item as 'labeled'.
func SubmitLabel(ctx context.Context, db TxBeginner, queueID int64, topicID string, labeledBy string) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Resolve document_id from queue entry.
	var docID int64
	if err := tx.QueryRow(ctx,
		`SELECT document_id FROM label_queue WHERE id = $1`, queueID,
	).Scan(&docID); err != nil {
		return err
	}

	// Upsert gold label.
	if _, err := tx.Exec(ctx, `
		INSERT INTO labels_gold (document_id, topic_id, labeled_by, labeled_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (document_id) DO UPDATE SET
		  topic_id   = EXCLUDED.topic_id,
		  labeled_by = EXCLUDED.labeled_by,
		  labeled_at = EXCLUDED.labeled_at
	`, docID, topicID, labeledBy); err != nil {
		return err
	}

	// Mark queue item labeled.
	now := time.Now().UTC()
	if _, err := tx.Exec(ctx, `
		UPDATE label_queue SET status = 'labeled', labeled_at = $2 WHERE id = $1
	`, queueID, now); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// SkipLabelQueue marks a queue item as 'skipped'.
func SkipLabelQueue(ctx context.Context, db DB, queueID int64) error {
	_, err := db.Exec(ctx, `
		UPDATE label_queue SET status = 'skipped' WHERE id = $1
	`, queueID)
	return err
}
