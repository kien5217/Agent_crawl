package postgres

import (
	"context"

	"Agent_Crawl/internal/domain/model"

	"github.com/jackc/pgx/v5"
)

func ListDocuments(ctx context.Context, conn *pgx.Conn, topicID string, limit int) ([]model.Document, error) {
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

	var out []model.Document
	for rows.Next() {
		var r model.Document
		if err := rows.Scan(&r.ID, &r.Title, &r.URL, &r.CanonicalURL, &r.TopicID, &r.PublishedAt, &r.ContentText, &r.SourceID); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func GetDocumentByID(ctx context.Context, conn *pgx.Conn, id int64) (*model.Document, error) {
	var r model.Document
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

func UpsertCrawledDocument(ctx context.Context, conn *pgx.Conn, in model.CrawledDocument) error {
	_, err := conn.Exec(ctx, `
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
	`, in.URL, in.CanonicalURL, in.Domain, in.SourceID,
		in.Title, in.PublishedAt, in.Author, in.ContentText,
		in.ContentHash, in.TopicID, in.TopicScoresJSON, in.Lang,
	)
	return err
}
