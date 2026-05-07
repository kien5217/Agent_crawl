package postgres

import (
	"context"

	"Agent_Crawl/internal/domain/model"
)

func GetDocumentCountsByDayTopic(ctx context.Context, db DB) ([]model.DocumentCountByDayTopic, error) {
	rows, err := db.Query(ctx, `
		SELECT TO_CHAR(DATE(COALESCE(published_at, created_at)), 'YYYY-MM-DD'), topic_id, COUNT(*)
		FROM documents
		GROUP BY 1, 2
		ORDER BY 1 DESC, 2 ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.DocumentCountByDayTopic
	for rows.Next() {
		var r model.DocumentCountByDayTopic
		if err := rows.Scan(&r.Date, &r.TopicID, &r.Count); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func GetDocumentCountsByTopic(ctx context.Context, db DB) ([]model.DocumentCountByTopic, error) {
	rows, err := db.Query(ctx, `
		SELECT topic_id, COUNT(*)
		FROM documents
		GROUP BY topic_id
		ORDER BY COUNT(*) DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.DocumentCountByTopic
	for rows.Next() {
		var r model.DocumentCountByTopic
		if err := rows.Scan(&r.TopicID, &r.Count); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func GetTopSources(ctx context.Context, db DB, limit int) ([]model.SourceCount, error) {
	rows, err := db.Query(ctx, `
		SELECT source_id, COUNT(*)
		FROM documents
		GROUP BY source_id
		ORDER BY COUNT(*) DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.SourceCount
	for rows.Next() {
		var r model.SourceCount
		if err := rows.Scan(&r.SourceID, &r.Count); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func GetSourceFailCounts(ctx context.Context, db DB) ([]model.SourceFailCount, error) {
	rows, err := db.Query(ctx, `
		SELECT source_id, COUNT(*)
		FROM crawl_queue
		WHERE status = 'failed'
		GROUP BY source_id
		ORDER BY COUNT(*) DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.SourceFailCount
	for rows.Next() {
		var r model.SourceFailCount
		if err := rows.Scan(&r.SourceID, &r.FailCount); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func GetSourceLastPostTimes(ctx context.Context, db DB) ([]model.SourceLastPost, error) {
	rows, err := db.Query(ctx, `
		SELECT source_id, MAX(COALESCE(published_at, created_at))
		FROM documents
		GROUP BY source_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.SourceLastPost
	for rows.Next() {
		var r model.SourceLastPost
		if err := rows.Scan(&r.SourceID, &r.LastPostAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func GetQueueFailureMetrics(ctx context.Context, db DB) (int64, int64, error) {
	var failCount int64
	var processedCount int64
	if err := db.QueryRow(ctx, `
		SELECT
		  COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0),
		  COALESCE(SUM(CASE WHEN status IN ('failed', 'done') THEN 1 ELSE 0 END), 0)
		FROM crawl_queue
	`).Scan(&failCount, &processedCount); err != nil {
		return 0, 0, err
	}
	return failCount, processedCount, nil
}

func ListRecentSimhashDocuments(ctx context.Context, db DB, limit int) ([]model.NearDuplicateDoc, error) {
	rows, err := db.Query(ctx, `
		SELECT id, title, url, source_id, published_at, content_simhash
		FROM documents
		WHERE content_simhash IS NOT NULL AND content_simhash <> 0
		ORDER BY COALESCE(published_at, created_at) DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []model.NearDuplicateDoc
	for rows.Next() {
		var r model.NearDuplicateDoc
		var hash int64
		if err := rows.Scan(&r.ID, &r.Title, &r.URL, &r.SourceID, &r.PublishedAt, &hash); err != nil {
			return nil, err
		}
		r.ContentSimHash = uint64(hash)
		out = append(out, r)
	}
	return out, rows.Err()
}
