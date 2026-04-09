package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

type DocumentRow struct {
	ID           int64
	Title        string
	URL          string
	CanonicalURL string
	TopicID      string
	PublishedAt  *time.Time
	ContentText  string
	SourceID     string
}

func ListDocuments(ctx context.Context, conn *pgx.Conn, topicID string, limit int) ([]DocumentRow, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := conn.Query(ctx, `
		SELECT id, title, url, canonical_url, topic_id, published_at, content_text, source_id
		FROM documents
		WHERE ($1 = '' OR $1 = 'all' OR topic_id = $1)
		ORDER BY COALESCE(published_at, created_at) DESC
		LIMIT $2
	`, topicID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []DocumentRow
	for rows.Next() {
		var r DocumentRow
		if err := rows.Scan(&r.ID, &r.Title, &r.URL, &r.CanonicalURL, &r.TopicID, &r.PublishedAt, &r.ContentText, &r.SourceID); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func GetDocumentByID(ctx context.Context, conn *pgx.Conn, id int64) (*DocumentRow, error) {
	var r DocumentRow
	err := conn.QueryRow(ctx, `
		SELECT id, title, url, canonical_url, topic_id, published_at, content_text, source_id
		FROM documents
		WHERE id = $1
	`, id).Scan(&r.ID, &r.Title, &r.URL, &r.CanonicalURL, &r.TopicID, &r.PublishedAt, &r.ContentText, &r.SourceID)
	if err != nil {
		return nil, err
	}
	return &r, nil
}
