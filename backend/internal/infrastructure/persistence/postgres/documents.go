package postgres

import (
	"context"

	"Agent_Crawl/internal/domain/model"
	"Agent_Crawl/internal/domain/repository"

	"github.com/jackc/pgx/v5"
)

func ListDocuments(ctx context.Context, db DB, filter repository.DocumentListFilter) ([]model.Document, error) {
	topicID := filter.TopicID
	sourceID := filter.SourceID
	limit := filter.Limit
	if limit < 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 50
	}

	var fromDate any
	if filter.FromDate != nil {
		fromDate = *filter.FromDate
	}

	var toDate any
	if filter.ToDate != nil {
		toDate = *filter.ToDate
	}

	var mlConfidenceMin any
	if filter.MLConfidenceMin != nil {
		mlConfidenceMin = *filter.MLConfidenceMin
	}

	query := `
		SELECT id, title, url, canonical_url, topic_id, published_at, content_text, source_id
		FROM documents
		WHERE ($1 = '' OR $1 = 'all' OR topic_id = $1)
		  AND ($2 = '' OR $2 = 'all' OR source_id = $2)
		  AND ($3::timestamptz IS NULL OR COALESCE(published_at, created_at) >= $3::timestamptz)
		  AND ($4::timestamptz IS NULL OR COALESCE(published_at, created_at) <= $4::timestamptz)
		  AND ($5::real IS NULL OR ml_confidence >= $5::real)
		ORDER BY COALESCE(published_at, created_at) DESC`

	var rows pgx.Rows
	var err error
	if limit == 0 {
		rows, err = db.Query(ctx, query, topicID, sourceID, fromDate, toDate, mlConfidenceMin)
	} else {
		rows, err = db.Query(ctx, query+` LIMIT $6`, topicID, sourceID, fromDate, toDate, mlConfidenceMin, limit)
	}
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

func GetDocumentByID(ctx context.Context, db DB, id int64) (*model.Document, error) {
	var r model.Document
	err := db.QueryRow(ctx, `
		SELECT id, title, url, canonical_url, topic_id, published_at, content_text, source_id
		FROM documents
		WHERE id = $1
	`, id).Scan(&r.ID, &r.Title, &r.URL, &r.CanonicalURL, &r.TopicID, &r.PublishedAt, &r.ContentText, &r.SourceID)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func UpsertCrawledDocument(ctx context.Context, db DB, in model.CrawledDocument) error {
	_, err := db.Exec(ctx, `
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
