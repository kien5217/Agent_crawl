package model

type QueueItem struct {
	ID       int64
	TopicID  string
	SourceID string
	URL      string
	Domain   string
	Attempts int
}
