package postgres

import (
	"context"
	"time"

	"Agent_Crawl/internal/domain/config"
	"Agent_Crawl/internal/domain/model"
	"Agent_Crawl/internal/domain/repository"

	"github.com/jackc/pgx/v5"
)

type Store struct {
	conn *pgx.Conn
}

func NewStore(conn *pgx.Conn) *Store {
	return &Store{conn: conn}
}

var _ repository.BootstrapRepository = (*Store)(nil)
var _ repository.MigrationRepository = (*Store)(nil)
var _ repository.QueueRepository = (*Store)(nil)
var _ repository.DocumentRepository = (*Store)(nil)
var _ repository.CrawlWriteRepository = (*Store)(nil)
var _ repository.LearningRepository = (*Store)(nil)
var _ repository.ModelRepository = (*Store)(nil)

func (s *Store) Migrate(ctx context.Context, migrationsDir string) error {
	return Migrate(ctx, s.conn, migrationsDir)
}

func (s *Store) UpsertTopics(ctx context.Context, tf config.TopicsFile) error {
	return UpsertTopics(ctx, s.conn, tf)
}

func (s *Store) UpsertSources(ctx context.Context, sf config.SourcesFile) error {
	return UpsertSources(ctx, s.conn, sf)
}

func (s *Store) EnqueueURL(ctx context.Context, topicID, sourceID, url, domain string, priority int) (bool, error) {
	return EnqueueURL(ctx, s.conn, topicID, sourceID, url, domain, priority)
}

func (s *Store) DequeueBatch(ctx context.Context, batchSize int) ([]model.QueueItem, error) {
	return DequeueBatch(ctx, s.conn, batchSize)
}

func (s *Store) MarkDone(ctx context.Context, id int64) error {
	return MarkDone(ctx, s.conn, id)
}

func (s *Store) MarkFailed(ctx context.Context, id int64, maxAttempts int, nextRunAt time.Time, lastErr string) error {
	return MarkFailed(ctx, s.conn, id, maxAttempts, nextRunAt, lastErr)
}

func (s *Store) ListDocuments(ctx context.Context, topicID string, limit int) ([]model.Document, error) {
	return ListDocuments(ctx, s.conn, topicID, limit)
}

func (s *Store) GetDocumentByID(ctx context.Context, id int64) (*model.Document, error) {
	return GetDocumentByID(ctx, s.conn, id)
}

func (s *Store) UpdateDocumentML(ctx context.Context, in model.PredictedDocumentML) error {
	return UpdateDocumentML(ctx, s.conn, in.DocumentID, in.ModelName, in.ModelVersion, in.MLTopicID, in.MLConfidence, in.MLScoresJSON, in.MLPredictedAt)
}

func (s *Store) UpsertCrawledDocument(ctx context.Context, in model.CrawledDocument) error {
	return UpsertCrawledDocument(ctx, s.conn, in)
}

func (s *Store) ListDocsForWeakLabel(ctx context.Context, limit int) ([]model.LearningDocument, error) {
	return ListDocsForWeakLabel(ctx, s.conn, limit)
}

func (s *Store) UpsertWeakLabel(ctx context.Context, in model.WeakLabel) error {
	return UpsertWeakLabel(ctx, s.conn, in.DocumentID, in.TopicID, in.Confidence, in.RuleID)
}

func (s *Store) ListTrainingSet(ctx context.Context, minWeakConf float32, limit int) ([]model.LearningDocument, error) {
	return ListTrainingSet(ctx, s.conn, minWeakConf, limit)
}

func (s *Store) SaveModel(ctx context.Context, name string, version int, classesJSON []byte, blob []byte) error {
	return SaveModel(ctx, s.conn, name, version, classesJSON, blob)
}

func (s *Store) LoadLatestModel(ctx context.Context, name string) (version int, blob []byte, err error) {
	return LoadLatestModel(ctx, s.conn, name)
}

func (s *Store) NextVersion(ctx context.Context, name string) (int, error) {
	var maxVer int
	err := s.conn.QueryRow(ctx, `SELECT COALESCE(MAX(version), 0) FROM models WHERE name=$1`, name).Scan(&maxVer)
	if err != nil {
		return 0, err
	}
	return maxVer + 1, nil
}

func (s *Store) ListUnlabeledDocs(ctx context.Context, limit int) (ids []int64, titles []string, contents []string, err error) {
	return ListUnlabeledDocs(ctx, s.conn, limit)
}

func (s *Store) EnqueueLabelQueue(ctx context.Context, in model.LabelQueueItem) error {
	return EnqueueLabelQueue(ctx, s.conn, in.DocumentID, in.Reason, in.Margin)
}
