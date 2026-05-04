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

type DocumentListFilter struct {
	TopicID         string
	SourceID        string
	FromDate        *time.Time
	ToDate          *time.Time
	MLConfidenceMin *float32
	Limit           int
}

type DocumentRepository interface {
	ListDocuments(ctx context.Context, filter DocumentListFilter) ([]model.Document, error)
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

type HealthRepository interface {
	GetHealthStats(ctx context.Context) (model.HealthStats, error)
}

type LabelingRepository interface {
	// ListPendingLabelQueue trả về các mục pending, ưu tiên margin thấp nhất (uncertain nhất).
	ListPendingLabelQueue(ctx context.Context, limit int) ([]model.LabelQueueEntry, error)
	// SubmitLabel ghi nhãn gold và đánh dấu queue item là 'labeled'.
	SubmitLabel(ctx context.Context, queueID int64, topicID string, labeledBy string) error
	// SkipLabelQueue đánh dấu queue item là 'skipped'.
	SkipLabelQueue(ctx context.Context, queueID int64) error
}

type WorkflowRepository interface {
	// Tạo mới một workflow execution (khi bắt đầu chạy)
	CreateWorkflow(ctx context.Context, wf model.WorkflowExecution) error

	// Cập nhật status/kết quả của workflow (khi xong/fail/halt)
	UpdateWorkflow(ctx context.Context, wf model.WorkflowExecution) error

	// Tạo mới một step execution
	CreateStep(ctx context.Context, step model.StepExecution) error

	// Cập nhật step khi bắt đầu chạy, thành công, hoặc thất bại
	UpdateStep(ctx context.Context, step model.StepExecution) error

	// Lấy danh sách workflow gần đây (cho lệnh `workflow list`)
	ListWorkflows(ctx context.Context, limit int) ([]model.WorkflowExecution, error)

	// Lấy tất cả steps của một workflow (cho lệnh `workflow logs`)
	ListSteps(ctx context.Context, workflowID string) ([]model.StepExecution, error)
}
