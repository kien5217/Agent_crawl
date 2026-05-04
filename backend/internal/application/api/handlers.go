package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	orchestration "Agent_Crawl/internal/application/orchestrator"
	appschedule "Agent_Crawl/internal/application/schedule"
	"Agent_Crawl/internal/domain/repository"

	"github.com/rs/zerolog/log"
)

// writeJSON is a helper that serializes v as JSON and writes it to w.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Error().Err(err).Msg("writeJSON encode failed")
	}
}

// writeError sends a JSON error response.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// handleListTopics returns all topics from config.
//
//	GET /api/topics
func (s *Server) handleListTopics(w http.ResponseWriter, r *http.Request) {
	type topicDTO struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	out := make([]topicDTO, 0, len(s.appCfg.Topics.Topics))
	for _, t := range s.appCfg.Topics.Topics {
		out = append(out, topicDTO{ID: t.ID, Name: t.Name})
	}
	writeJSON(w, http.StatusOK, out)
}

// handleListDocuments returns a paginated list of documents.
// GET /api/documents?topic=<id>&source=<id>&from_date=<date>&to_date=<date>&ml_confidence_min=<v>&limit=<n>
func (s *Server) handleListDocuments(w http.ResponseWriter, r *http.Request) {
	topic := r.URL.Query().Get("topic")
	if topic == "" {
		topic = "all"
	}

	source := strings.TrimSpace(r.URL.Query().Get("source"))
	fromDate, err := parseDateQuery(r.URL.Query().Get("from_date"), false)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid from_date (use RFC3339 or YYYY-MM-DD)")
		return
	}
	toDate, err := parseDateQuery(r.URL.Query().Get("to_date"), true)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid to_date (use RFC3339 or YYYY-MM-DD)")
		return
	}

	var mlConfMin *float32
	if raw := strings.TrimSpace(r.URL.Query().Get("ml_confidence_min")); raw != "" {
		v, err := strconv.ParseFloat(raw, 32)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid ml_confidence_min")
			return
		}
		fv := float32(v)
		mlConfMin = &fv
	}

	rawLimit := strings.TrimSpace(r.URL.Query().Get("limit"))
	limit := 50
	hasAnyFilter := topic != "all" || source != "" || fromDate != nil || toDate != nil || mlConfMin != nil
	if rawLimit == "" && hasAnyFilter {
		// Filtered queries default to full result set unless caller explicitly sets limit.
		limit = 0
	}
	if rawLimit != "" {
		n, err := strconv.Atoi(rawLimit)
		if err == nil {
			if n == 0 {
				limit = 0
			} else if n > 0 {
				limit = n
			}
		}
	}

	docs, err := s.document.ListDocuments(r.Context(), repository.DocumentListFilter{
		TopicID:         topic,
		SourceID:        source,
		FromDate:        fromDate,
		ToDate:          toDate,
		MLConfidenceMin: mlConfMin,
		Limit:           limit,
	})
	if err != nil {
		log.Error().Err(err).Msg("handleListDocuments")
		writeError(w, http.StatusInternalServerError, "failed to list documents")
		return
	}
	writeJSON(w, http.StatusOK, docs)
}

func parseDateQuery(raw string, endOfDay bool) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return &t, nil
	}

	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil, err
	}
	if endOfDay {
		t = t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	}
	return &t, nil
}

// handleGetDocument returns a single document by ID.
//
//	GET /api/documents/{id}
func (s *Server) handleGetDocument(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid document id")
		return
	}

	doc, err := s.document.GetDocumentByID(r.Context(), id)
	if err != nil {
		log.Error().Err(err).Int64("id", id).Msg("handleGetDocument")
		writeError(w, http.StatusInternalServerError, "failed to get document")
		return
	}
	if doc == nil {
		writeError(w, http.StatusNotFound, "document not found")
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

// handleTriggerSchedule runs a discovery schedule cycle and returns counts.
//
//	POST /api/schedule
func (s *Server) handleTriggerSchedule(w http.ResponseWriter, r *http.Request) {
	if s.scheduleFlow == nil {
		writeError(w, http.StatusServiceUnavailable, "schedule workflow is not configured")
		return
	}

	runResult, err := s.scheduleFlow(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("handleTriggerSchedule")
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, scheduleResultFromRun(runResult))
}

func scheduleResultFromRun(runResult *orchestration.RunResult) appschedule.Result {
	result := appschedule.Result{Counts: map[string]int{}}
	if runResult == nil {
		return result
	}

	discoveryResult, ok := runResult.StepResults["Discovery"]
	if !ok || discoveryResult == nil {
		return result
	}

	var summary struct {
		RSSEnqueued     int `json:"rss_enqueued"`
		SitemapEnqueued int `json:"sitemap_enqueued"`
	}
	if err := json.Unmarshal([]byte(discoveryResult.Summary()), &summary); err != nil {
		return result
	}

	result.Counts["rss"] = summary.RSSEnqueued
	result.Counts["sitemap"] = summary.SitemapEnqueued
	return result
}

// handleListWorkflows returns recent workflow executions.
//
//	GET /api/workflows?limit=<n>
func (s *Server) handleListWorkflows(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	wfs, err := s.workflow.ListWorkflows(r.Context(), limit)
	if err != nil {
		log.Error().Err(err).Msg("handleListWorkflows")
		writeError(w, http.StatusInternalServerError, "failed to list workflows")
		return
	}
	writeJSON(w, http.StatusOK, wfs)
}

// handleListSteps returns all step executions for a workflow.
//
//	GET /api/workflows/{id}/steps
func (s *Server) handleListSteps(w http.ResponseWriter, r *http.Request) {
	workflowID := r.PathValue("id")
	if workflowID == "" {
		writeError(w, http.StatusBadRequest, "missing workflow id")
		return
	}

	steps, err := s.workflow.ListSteps(r.Context(), workflowID)
	if err != nil {
		log.Error().Err(err).Str("workflow_id", workflowID).Msg("handleListSteps")
		writeError(w, http.StatusInternalServerError, "failed to list steps")
		return
	}
	writeJSON(w, http.StatusOK, steps)
}

// handleGetHealth returns quality-of-service metrics for the crawl pipeline.
//
//	GET /api/health
func (s *Server) handleGetHealth(w http.ResponseWriter, r *http.Request) {
	stats, err := s.health.GetHealthStats(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("handleGetHealth")
		writeError(w, http.StatusInternalServerError, "failed to get health stats")
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

// handleListLabelQueue returns pending items from label_queue for human review.
//
//	GET /api/label-queue?limit=<n>
func (s *Server) handleListLabelQueue(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	items, err := s.labeling.ListPendingLabelQueue(r.Context(), limit)
	if err != nil {
		log.Error().Err(err).Msg("handleListLabelQueue")
		writeError(w, http.StatusInternalServerError, "failed to list label queue")
		return
	}
	if items == nil {
		writeJSON(w, http.StatusOK, []struct{}{})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// handleSubmitLabel records a gold label for a queue item.
//
//	POST /api/label-queue/{id}/label
//	Body: {"topic_id":"...", "labeled_by":"..."}
func (s *Server) handleSubmitLabel(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	queueID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid queue item id")
		return
	}

	var body struct {
		TopicID   string `json:"topic_id"`
		LabeledBy string `json:"labeled_by"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(body.TopicID) == "" {
		writeError(w, http.StatusBadRequest, "topic_id is required")
		return
	}

	if err := s.labeling.SubmitLabel(r.Context(), queueID, body.TopicID, body.LabeledBy); err != nil {
		log.Error().Err(err).Int64("queue_id", queueID).Msg("handleSubmitLabel")
		writeError(w, http.StatusInternalServerError, "failed to submit label")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "labeled"})
}

// handleSkipLabelQueue marks a queue item as skipped.
//
//	POST /api/label-queue/{id}/skip
func (s *Server) handleSkipLabelQueue(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	queueID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid queue item id")
		return
	}

	if err := s.labeling.SkipLabelQueue(r.Context(), queueID); err != nil {
		log.Error().Err(err).Int64("queue_id", queueID).Msg("handleSkipLabelQueue")
		writeError(w, http.StatusInternalServerError, "failed to skip queue item")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "skipped"})
}
