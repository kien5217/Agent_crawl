package orchestration

import (
	"encoding/json"
	"strings"

	appschedule "Agent_Crawl/internal/application/schedule"
	"Agent_Crawl/internal/domain/config"
	"Agent_Crawl/internal/domain/repository"
)

// DailyCrawlWorkflow tạo workflow chạy đầy đủ pipeline:
// Discovery → Worker → WeakLabel → Train → Select → Predict
func DailyCrawlWorkflow(
	schedulerSvc *appschedule.Service,
	appCfg *config.AppConfig,
	queue repository.QueueRepository,
	crawlDoc repository.CrawlWriteRepository,
	learningRepo repository.LearningRepository,
	modelRepo repository.ModelRepository,
	docRepo repository.DocumentRepository,
	concurrency int,
	weakLabelLimit int,
	trainClasses []string,
	minWeakConf float32,
	modelName string,
	selectLimit int,
	batchSize int,
	predictLimit int,
) WorkflowDef {
	discoveryStep := NewDiscoveryStep(schedulerSvc)
	workerStep := NewWorkerStep(appCfg, queue, crawlDoc, concurrency)
	weakLabelStep := NewWeakLabelStep(learningRepo, weakLabelLimit)
	trainStep := NewTrainStep(learningRepo, modelRepo, trainClasses, minWeakConf, modelName, 0)
	selectStep := NewSelectStep(learningRepo, modelRepo, modelName, selectLimit, batchSize)
	predictStep := NewPredictStep(docRepo, modelRepo, modelName, "all", predictLimit, true)

	return WorkflowDef{
		Name: "daily-crawl-and-train",
		Steps: []Step{
			discoveryStep,
			workerStep,
			weakLabelStep,
			trainStep,
			selectStep,
			predictStep,
		},
		Gates: map[string]GateFn{
			// Halt nếu Discovery không enqueue được gì
			"Discovery": func(r StepResult) (bool, string) {
				if r == nil {
					return false, "no result"
				}
				var m map[string]any
				if err := json.Unmarshal([]byte(r.Summary()), &m); err != nil {
					return true, ""
				}
				rss, _ := m["rss_enqueued"].(float64)
				sitemap, _ := m["sitemap_enqueued"].(float64)
				if rss+sitemap == 0 {
					return false, "discovery enqueued 0 URLs, skipping pipeline"
				}
				return true, ""
			},
			// Halt nếu WeakLabel không ghi được nhãn nào
			"WeakLabel": func(r StepResult) (bool, string) {
				if r == nil {
					return false, "no result"
				}
				var m map[string]any
				if err := json.Unmarshal([]byte(r.Summary()), &m); err != nil {
					return true, ""
				}
				written, _ := m["labels_written"].(float64)
				if written == 0 {
					return false, "weak-label wrote 0 labels, skipping train"
				}
				return true, ""
			},
			// Halt nếu Train không có đủ samples
			"Train": func(r StepResult) (bool, string) {
				if r == nil {
					return false, "no result"
				}
				var m map[string]any
				if err := json.Unmarshal([]byte(r.Summary()), &m); err != nil {
					return true, ""
				}
				samples, _ := m["samples"].(float64)
				if samples == 0 {
					return false, "train had 0 samples"
				}
				return true, ""
			},
			// Halt nếu Select không pick được doc nào để label
			"Select": func(r StepResult) (bool, string) {
				if r == nil {
					return true, "" // không bắt buộc phải có kết quả
				}
				var m map[string]any
				if err := json.Unmarshal([]byte(r.Summary()), &m); err != nil {
					return true, ""
				}
				picked, _ := m["picked"].(float64)
				if picked == 0 {
					return false, "select picked 0 docs, skipping predict"
				}
				return true, ""
			},
		},
	}
}

// splitCSV là helper để parse danh sách class từ chuỗi CSV
func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
