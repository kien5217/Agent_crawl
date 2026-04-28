package model

import "time"

// HealthStats aggregates quality-of-service metrics for the crawl pipeline.
type HealthStats struct {
	QueueSize       int64      `json:"queue_size"`
	LastCrawlTime   *time.Time `json:"last_crawl_time"`
	SourceFailCount int64      `json:"source_fail_count"`
}
