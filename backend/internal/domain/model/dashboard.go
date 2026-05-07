package model

import "time"

type DocumentCountByDayTopic struct {
	Date    string `json:"date"`
	TopicID string `json:"topic_id"`
	Count   int64  `json:"count"`
}

type DocumentCountByTopic struct {
	TopicID string `json:"topic_id"`
	Count   int64  `json:"count"`
}

type SourceCount struct {
	SourceID string `json:"source_id"`
	Count    int64  `json:"count"`
}

type SourceFailCount struct {
	SourceID  string `json:"source_id"`
	FailCount int64  `json:"fail_count"`
}

type SourceLastPost struct {
	SourceID   string     `json:"source_id"`
	LastPostAt *time.Time `json:"last_post_at"`
}

type NearDuplicateDoc struct {
	ID             int64      `json:"id"`
	Title          string     `json:"title"`
	URL            string     `json:"url"`
	SourceID       string     `json:"source_id"`
	PublishedAt    *time.Time `json:"published_at"`
	ContentSimHash uint64     `json:"-"`
}
