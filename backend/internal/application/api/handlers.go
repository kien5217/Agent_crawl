package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	result, err := s.scheduler.Run(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("handleTriggerSchedule")
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if s.afterSchedule != nil {
		if err := s.afterSchedule(r.Context()); err != nil {
			log.Error().Err(err).Msg("handleTriggerSchedule post-schedule pipeline")
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	writeJSON(w, http.StatusOK, result)
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
