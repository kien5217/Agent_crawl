package orchestration

import (
	"context"
	"encoding/json"
	"time"

	"Agent_Crawl/internal/application/learning"
	appschedule "Agent_Crawl/internal/application/schedule"
	"Agent_Crawl/internal/application/worker"
	"Agent_Crawl/internal/domain/config"
	"Agent_Crawl/internal/domain/model"
	"Agent_Crawl/internal/domain/repository"
	"Agent_Crawl/internal/infrastructure/classify"
	ml "Agent_Crawl/internal/infrastructure/machine_learning"
)

// --- StepResult implementations ---

type jsonResult struct{ data map[string]any }

func (r *jsonResult) Summary() string {
	b, _ := json.Marshal(r.data)
	return string(b)
}

// --- DiscoveryStep ---

type DiscoveryStep struct{ svc *appschedule.Service }

func NewDiscoveryStep(svc *appschedule.Service) *DiscoveryStep {
	return &DiscoveryStep{svc: svc}
}
func (s *DiscoveryStep) Name() string { return "Discovery" }
func (s *DiscoveryStep) Run(ctx context.Context) (StepResult, error) {
	result, err := s.svc.Run(ctx)
	if err != nil {
		return nil, err
	}
	return &jsonResult{data: map[string]any{
		"rss_enqueued":     result.Counts["rss"],
		"sitemap_enqueued": result.Counts["sitemap"],
	}}, nil
}

// --- WorkerStep ---

type WorkerStep struct {
	cfg         *config.AppConfig
	queue       repository.QueueRepository
	crawlDoc    repository.CrawlWriteRepository
	concurrency int
}

func NewWorkerStep(cfg *config.AppConfig, queue repository.QueueRepository,
	crawlDoc repository.CrawlWriteRepository, concurrency int) *WorkerStep {
	return &WorkerStep{cfg: cfg, queue: queue, crawlDoc: crawlDoc, concurrency: concurrency}
}
func (s *WorkerStep) Name() string { return "Worker" }
func (s *WorkerStep) Run(ctx context.Context) (StepResult, error) {
	clf := classify.NewKeywordClassifier(s.cfg.Topics, s.cfg.Config.Classify.MinScoreToAccept)
	w := worker.New(s.cfg.Config, clf, s.queue, s.crawlDoc)
	if err := w.Run(ctx, s.concurrency); err != nil {
		return nil, err
	}
	return &jsonResult{data: map[string]any{"status": "done"}}, nil
}

// --- WeakLabelStep ---

type WeakLabelStep struct {
	repo    repository.LearningRepository
	labeler *learning.WeakLabeler
	limit   int
}

func NewWeakLabelStep(repo repository.LearningRepository, limit int) *WeakLabelStep {
	return &WeakLabelStep{
		repo:    repo,
		labeler: learning.NewWeakLabeler(),
		limit:   limit,
	}
}

func (s *WeakLabelStep) Name() string { return "WeakLabel" }

func (s *WeakLabelStep) Run(ctx context.Context) (StepResult, error) {
	docs, err := s.repo.ListDocsForWeakLabel(ctx, s.limit)
	if err != nil {
		return nil, err
	}

	applied := learning.ApplyWeakLabels(docs, s.labeler)

	written := 0
	for _, a := range applied {
		err := s.repo.UpsertWeakLabel(ctx, model.WeakLabel{
			DocumentID: a.DocID,
			TopicID:    a.TopicID,
			Confidence: a.Confidence,
			RuleID:     a.RuleID,
		})
		if err != nil {
			// nếu muốn strict thì return err luôn
			// nếu muốn tolerant thì continue như dưới
			continue
		}
		written++
	}

	return &jsonResult{data: map[string]any{
		"docs_scanned":   len(docs),
		"labels_matched": len(applied),
		"labels_written": written,
	}}, nil
}

// Tương tự bạn thêm TrainStep, SelectStep, PredictStep theo cùng pattern

// --- TrainStep ---

type TrainStep struct {
	learningRepo repository.LearningRepository
	modelRepo    repository.ModelRepository
	classes      []string
	minWeakConf  float32
	modelName    string
	modelVer     int
}

func NewTrainStep(
	learningRepo repository.LearningRepository,
	modelRepo repository.ModelRepository,
	classes []string,
	minWeakConf float32,
	modelName string,
	modelVer int,
) *TrainStep {
	return &TrainStep{
		learningRepo: learningRepo,
		modelRepo:    modelRepo,
		classes:      classes,
		minWeakConf:  minWeakConf,
		modelName:    modelName,
		modelVer:     modelVer,
	}
}

func (s *TrainStep) Name() string { return "Train" }

func (s *TrainStep) Run(ctx context.Context) (StepResult, error) {
	trainDocs, err := s.learningRepo.ListTrainingSet(ctx, s.minWeakConf, 50000)
	if err != nil {
		return nil, err
	}

	bundle, stats := learning.TrainFromDocs(trainDocs, s.classes, 3)
	blob, err := bundle.Marshal()
	if err != nil {
		return nil, err
	}

	ver := s.modelVer
	if ver == 0 {
		ver, err = s.modelRepo.NextVersion(ctx, s.modelName)
		if err != nil {
			return nil, err
		}
	}

	classesJSON := learning.ClassesJSON(s.classes)
	if err := s.modelRepo.SaveModel(ctx, s.modelName, ver, classesJSON, blob); err != nil {
		return nil, err
	}

	return &jsonResult{data: map[string]any{
		"model":   s.modelName,
		"version": ver,
		"samples": stats.NumSamples,
		"classes": stats.NumClasses,
		"vocab":   stats.VocabSize,
	}}, nil
}

// --- SelectStep ---

type SelectStep struct {
	learningRepo repository.LearningRepository
	modelRepo    repository.ModelRepository
	modelName    string
	limit        int
	batchSize    int
}

func NewSelectStep(
	learningRepo repository.LearningRepository,
	modelRepo repository.ModelRepository,
	modelName string,
	limit int,
	batchSize int,
) *SelectStep {
	return &SelectStep{
		learningRepo: learningRepo,
		modelRepo:    modelRepo,
		modelName:    modelName,
		limit:        limit,
		batchSize:    batchSize,
	}
}

func (s *SelectStep) Name() string { return "Select" }

func (s *SelectStep) Run(ctx context.Context) (StepResult, error) {
	_, blob, err := s.modelRepo.LoadLatestModel(ctx, s.modelName)
	if err != nil {
		return nil, err
	}
	bundle, err := ml.Unmarshal(blob)
	if err != nil {
		return nil, err
	}

	ids, titles, contents, err := s.learningRepo.ListUnlabeledDocs(ctx, s.limit)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return &jsonResult{data: map[string]any{"unlabeled_scanned": 0, "picked": 0, "label_queue_written": 0}}, nil
	}

	pickedIDs := learning.SelectBatchForLabeling(bundle, ids, titles, contents, s.batchSize)
	picks := learning.ComputeMargins(bundle, ids, titles, contents)
	marginByID := map[int64]float32{}
	for _, p := range picks {
		marginByID[p.DocID] = float32(p.Margin)
	}

	written := 0
	for _, id := range pickedIDs {
		err := s.learningRepo.EnqueueLabelQueue(ctx, model.LabelQueueItem{
			DocumentID: id,
			Reason:     "active:margin+diversity",
			Margin:     marginByID[id],
		})
		if err != nil {
			continue
		}
		written++
	}

	return &jsonResult{data: map[string]any{
		"unlabeled_scanned":   len(ids),
		"picked":              len(pickedIDs),
		"label_queue_written": written,
	}}, nil
}

// --- PredictStep ---

type PredictStep struct {
	docRepo   repository.DocumentRepository
	modelRepo repository.ModelRepository
	modelName string
	topic     string
	limit     int
	write     bool
}

func NewPredictStep(
	docRepo repository.DocumentRepository,
	modelRepo repository.ModelRepository,
	modelName string,
	topic string,
	limit int,
	write bool,
) *PredictStep {
	return &PredictStep{
		docRepo:   docRepo,
		modelRepo: modelRepo,
		modelName: modelName,
		topic:     topic,
		limit:     limit,
		write:     write,
	}
}

func (s *PredictStep) Name() string { return "Predict" }

func (s *PredictStep) Run(ctx context.Context) (StepResult, error) {
	ver, blob, err := s.modelRepo.LoadLatestModel(ctx, s.modelName)
	if err != nil {
		return nil, err
	}
	bundle, err := ml.Unmarshal(blob)
	if err != nil {
		return nil, err
	}

	docs, err := s.docRepo.ListDocuments(ctx, repository.DocumentListFilter{TopicID: s.topic, Limit: s.limit})
	if err != nil {
		return nil, err
	}

	now := time.Now()
	predicted := 0
	for _, d := range docs {
		x := bundle.Vectorizer.Transform(d.Title + "\n" + d.ContentText)
		p := bundle.Model.PredictProba(x)

		mlTopic := bundle.Model.Classes[0]
		mlConf := float32(p[0])
		for i, prob := range p {
			if float32(prob) > mlConf {
				mlTopic = bundle.Model.Classes[i]
				mlConf = float32(prob)
			}
		}

		if s.write {
			scores := map[string]float64{}
			for i, c := range bundle.Model.Classes {
				if i < len(p) {
					scores[c] = p[i]
				}
			}
			b, _ := json.Marshal(scores)
			if err := s.docRepo.UpdateDocumentML(ctx, model.PredictedDocumentML{
				DocumentID:    d.ID,
				ModelName:     s.modelName,
				ModelVersion:  ver,
				MLTopicID:     mlTopic,
				MLConfidence:  mlConf,
				MLScoresJSON:  string(b),
				MLPredictedAt: now,
			}); err != nil {
				continue
			}
		}
		predicted++
	}

	return &jsonResult{data: map[string]any{
		"model":          s.modelName,
		"version":        ver,
		"docs_scanned":   len(docs),
		"docs_predicted": predicted,
	}}, nil
}
