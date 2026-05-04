package model

import "time"

type LearningDocument struct {
	ID      int64
	Title   string
	Content string
	TopicID string
}

type WeakLabel struct {
	DocumentID int64
	TopicID    string
	Confidence float32
	RuleID     string
}

type LabelQueueItem struct {
	DocumentID int64
	Reason     string
	Margin     float32
}

// LabelQueueEntry is a label_queue row joined with its document, returned to the Labeling UI.
type LabelQueueEntry struct {
	ID          int64
	DocumentID  int64
	Status      string
	Reason      string
	Margin      float32
	CreatedAt   time.Time
	Title       string
	URL         string
	TopicID     string
	ContentText string
}
