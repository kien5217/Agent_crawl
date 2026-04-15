package repository

import (
	"context"
	"time"

	"Agent_Crawl/internal/domain/config"
	"Agent_Crawl/internal/domain/model"
)

type BootstrapRepository interface {
	UpsertTopics(ctx context.Context, tf config.TopicsFile) error
	UpsertSources(ctx context.Context, sf config.SourcesFile) error
}

type QueueRepository interface {
	EnqueueURL(ctx context.Context, topicID, sourceID, url, domain string, priority int) (bool, error)
	DequeueBatch(ctx context.Context, batchSize int) ([]model.QueueItem, error)
	MarkDone(ctx context.Context, id int64) error
	MarkFailed(ctx context.Context, id int64, maxAttempts int, nextRunAt time.Time, lastErr string) error
}

type DocumentRepository interface {
	ListDocuments(ctx context.Context, topicID string, limit int) ([]model.Document, error)
	GetDocumentByID(ctx context.Context, id int64) (*model.Document, error)
	UpdateDocumentML(ctx context.Context, in model.PredictedDocumentML) error
}

type CrawlWriteRepository interface {
	UpsertCrawledDocument(ctx context.Context, in model.CrawledDocument) error
}

type LearningRepository interface {
	ListDocsForWeakLabel(ctx context.Context, limit int) ([]model.LearningDocument, error)
	UpsertWeakLabel(ctx context.Context, in model.WeakLabel) error
	ListTrainingSet(ctx context.Context, minWeakConf float32, limit int) ([]model.LearningDocument, error)
	ListUnlabeledDocs(ctx context.Context, limit int) (ids []int64, titles []string, contents []string, err error)
	EnqueueLabelQueue(ctx context.Context, in model.LabelQueueItem) error
}

type ModelRepository interface {
	SaveModel(ctx context.Context, name string, version int, classesJSON []byte, blob []byte) error
	LoadLatestModel(ctx context.Context, name string) (version int, blob []byte, err error)
	NextVersion(ctx context.Context, name string) (int, error)
}

type MigrationRepository interface {
	Migrate(ctx context.Context, migrationsDir string) error
}
