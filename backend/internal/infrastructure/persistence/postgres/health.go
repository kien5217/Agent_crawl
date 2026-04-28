package postgres

import (
	"context"
	"time"

	"Agent_Crawl/internal/domain/model"
)

// GetHealthStats returns queue_size (pending), last successful crawl time, and
// number of distinct sources that currently have at least one failed URL.
func GetHealthStats(ctx context.Context, db DB) (model.HealthStats, error) {
	var stats model.HealthStats

	// Pending queue size
	if err := db.QueryRow(ctx,
		`SELECT COUNT(*) FROM crawl_queue WHERE status = 'pending'`,
	).Scan(&stats.QueueSize); err != nil {
		return stats, err
	}

	// Last time a URL was successfully crawled
	var lastCrawl *time.Time
	if err := db.QueryRow(ctx,
		`SELECT MAX(updated_at) FROM crawl_queue WHERE status = 'done'`,
	).Scan(&lastCrawl); err != nil {
		return stats, err
	}
	stats.LastCrawlTime = lastCrawl

	// Distinct sources with at least one permanently-failed URL
	if err := db.QueryRow(ctx,
		`SELECT COUNT(DISTINCT source_id) FROM crawl_queue WHERE status = 'failed'`,
	).Scan(&stats.SourceFailCount); err != nil {
		return stats, err
	}

	return stats, nil
}
