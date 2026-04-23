package model

import "time"

type Document struct {
	ID           int64
	Title        string
	URL          string
	CanonicalURL string
	TopicID      string
	PublishedAt  *time.Time
	ContentText  string
	SourceID     string
}

// PredictedDocumentML is write-back payload for ML prediction fields on a document.
type PredictedDocumentML struct {
	DocumentID    int64
	ModelName     string
	ModelVersion  int
	MLTopicID     string
	MLConfidence  float32
	MLScoresJSON  string
	MLPredictedAt time.Time
}

type CrawledDocument struct {
	URL             string
	CanonicalURL    string
	Domain          string
	SourceID        string
	Title           string
	PublishedAt     *time.Time
	Author          string
	ContentText     string
	ContentHash     string
	TopicID         string
	TopicScoresJSON string
	Lang            string
}
