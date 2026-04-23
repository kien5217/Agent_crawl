package model

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
