package api

import (
	"encoding/json"
	"io"
	"math/bits"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	orchestration "Agent_Crawl/internal/application/orchestrator"
	appschedule "Agent_Crawl/internal/application/schedule"
	"Agent_Crawl/internal/domain/config"
	"Agent_Crawl/internal/domain/model"
	"Agent_Crawl/internal/domain/repository"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

// writeJSON is a helper that serializes v as JSON and writes it to w.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	data, err := json.Marshal(v)
	if err != nil {
		log.Error().Err(err).Msg("writeJSON marshal failed")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(status)
	w.Write(data)
}

// writeError sends a JSON error response.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func pathParam(r *http.Request, prefix string) string {
	if !strings.HasPrefix(r.URL.Path, prefix) {
		return ""
	}

	trimmed := strings.TrimPrefix(r.URL.Path, prefix)
	trimmed = strings.Trim(trimmed, "/")
	if trimmed == "" {
		return ""
	}

	parts := strings.Split(trimmed, "/")
	return parts[0]
}

func pathSegments(r *http.Request, prefix string) []string {
	if !strings.HasPrefix(r.URL.Path, prefix) {
		return nil
	}

	trimmed := strings.TrimPrefix(r.URL.Path, prefix)
	trimmed = strings.Trim(trimmed, "/")
	if trimmed == "" {
		return []string{}
	}

	return strings.Split(trimmed, "/")
}

func normalizeTopicIDs(ids []string) []string {
	out := make([]string, 0, len(ids))
	seen := map[string]bool{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

type dashboardTopicCountDTO struct {
	TopicID   string `json:"topic_id"`
	TopicName string `json:"topic_name"`
	Count     int64  `json:"count"`
}

type dashboardDayCountDTO struct {
	Date      string `json:"date"`
	TopicID   string `json:"topic_id"`
	TopicName string `json:"topic_name"`
	Count     int64  `json:"count"`
}

type dashboardSourceDTO struct {
	SourceID string `json:"source_id"`
	Name     string `json:"name"`
	Count    int64  `json:"count"`
}

type dashboardSLAEntryDTO struct {
	SourceID          string     `json:"source_id"`
	Name              string     `json:"name"`
	Enabled           bool       `json:"enabled"`
	LastPostAt        *time.Time `json:"last_post_at"`
	FailCount         int64      `json:"fail_count"`
	DaysSinceLastPost int        `json:"days_since_last_post"`
	Stale             bool       `json:"stale"`
}

type dashboardOverviewDTO struct {
	CountsByTopic []dashboardTopicCountDTO `json:"counts_by_topic"`
	CountsByDay   []dashboardDayCountDTO   `json:"counts_by_day"`
	TopSources    []dashboardSourceDTO     `json:"top_sources"`
	FailRate      float64                  `json:"fail_rate"`
	SLASources    []dashboardSLAEntryDTO   `json:"sla_sources"`
}

func hammingDistance(a, b uint64) int {
	return bits.OnesCount64(a ^ b)
}

func (s *Server) saveTopics() error {
	b, err := yaml.Marshal(s.appCfg.Topics)
	if err != nil {
		return err
	}
	return os.WriteFile(s.topicsPath, b, 0o644)
}

func (s *Server) saveSources() error {
	b, err := yaml.Marshal(s.appCfg.Sources)
	if err != nil {
		return err
	}
	return os.WriteFile(s.sourcesPath, b, 0o644)
}

func (s *Server) handleTopics(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListTopics(w, r)
	case http.MethodPost:
		s.handleCreateTopic(w, r)
	case http.MethodPatch:
		s.handleUpdateTopic(w, r)
	default:
		w.Header().Set("Allow", "GET, POST, PATCH")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleSources(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListSources(w, r)
	case http.MethodPost:
		s.handleCreateSource(w, r)
	case http.MethodPatch:
		s.handleUpdateSource(w, r)
	default:
		w.Header().Set("Allow", "GET, POST, PATCH")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
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

func (s *Server) handleListSources(w http.ResponseWriter, r *http.Request) {
	type sourceDTO struct {
		ID           string   `json:"id"`
		Name         string   `json:"name"`
		URL          string   `json:"url"`
		Enabled      bool     `json:"enabled"`
		ScheduleFreq string   `json:"schedule_freq"`
		TopicIDs     []string `json:"topic_ids"`
	}
	out := make([]sourceDTO, 0, len(s.appCfg.Sources.Sources))
	for _, src := range s.appCfg.Sources.Sources {
		out = append(out, sourceDTO{
			ID:           src.ID,
			Name:         src.Name,
			URL:          src.RSSURL,
			Enabled:      src.Enabled,
			ScheduleFreq: src.ScheduleFreq,
			TopicIDs:     src.TopicIDs,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleCreateTopic(w http.ResponseWriter, r *http.Request) {
	type topicInput struct {
		ID       string           `json:"id"`
		Name     string           `json:"name"`
		Keywords []config.Keyword `json:"keywords,omitempty"`
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var body topicInput
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	body.ID = strings.TrimSpace(body.ID)
	body.Name = strings.TrimSpace(body.Name)
	if body.ID == "" || body.Name == "" {
		writeError(w, http.StatusBadRequest, "id and name are required")
		return
	}

	for _, t := range s.appCfg.Topics.Topics {
		if t.ID == body.ID {
			writeError(w, http.StatusConflict, "topic already exists")
			return
		}
	}

	newTopic := config.Topic{
		ID:       body.ID,
		Name:     body.Name,
		Keywords: body.Keywords,
	}
	s.appCfg.Topics.Topics = append(s.appCfg.Topics.Topics, newTopic)

	if err := s.saveTopics(); err != nil {
		log.Error().Err(err).Msg("save topics failed")
		writeError(w, http.StatusInternalServerError, "failed to save topics")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"id": newTopic.ID, "name": newTopic.Name})
}

func (s *Server) handleUpdateTopic(w http.ResponseWriter, r *http.Request) {
	type topicUpdate struct {
		ID       string           `json:"id"`
		Name     string           `json:"name,omitempty"`
		Keywords []config.Keyword `json:"keywords,omitempty"`
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var body topicUpdate
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	body.ID = strings.TrimSpace(body.ID)
	if body.ID == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	found := false
	for idx, t := range s.appCfg.Topics.Topics {
		if t.ID == body.ID {
			if body.Name != "" {
				s.appCfg.Topics.Topics[idx].Name = strings.TrimSpace(body.Name)
			}
			if body.Keywords != nil {
				s.appCfg.Topics.Topics[idx].Keywords = body.Keywords
			}
			found = true
			break
		}
	}

	if !found {
		writeError(w, http.StatusNotFound, "topic not found")
		return
	}

	if err := s.saveTopics(); err != nil {
		log.Error().Err(err).Msg("save topics failed")
		writeError(w, http.StatusInternalServerError, "failed to save topics")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) handleCreateSource(w http.ResponseWriter, r *http.Request) {
	type sourceInput struct {
		ID           string   `json:"id"`
		Name         string   `json:"name"`
		URL          string   `json:"url"`
		Enabled      *bool    `json:"enabled,omitempty"`
		ScheduleFreq string   `json:"schedule_freq,omitempty"`
		TopicIDs     []string `json:"topic_ids,omitempty"`
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var body sourceInput
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	body.ID = strings.TrimSpace(body.ID)
	body.URL = strings.TrimSpace(body.URL)
	if body.ID == "" || body.URL == "" {
		writeError(w, http.StatusBadRequest, "id and url are required")
		return
	}

	for _, src := range s.appCfg.Sources.Sources {
		if src.ID == body.ID {
			writeError(w, http.StatusConflict, "source already exists")
			return
		}
	}

	parsed, err := url.Parse(body.URL)
	if err != nil || parsed.Hostname() == "" {
		writeError(w, http.StatusBadRequest, "url must be a valid absolute URL")
		return
	}

	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}

	src := config.Source{
		ID:           body.ID,
		Name:         strings.TrimSpace(body.Name),
		Domain:       parsed.Hostname(),
		RSSURL:       body.URL,
		ScheduleFreq: strings.TrimSpace(body.ScheduleFreq),
		TopicIDs:     normalizeTopicIDs(body.TopicIDs),
		Enabled:      enabled,
	}
	if src.Name == "" {
		src.Name = src.ID
	}

	s.appCfg.Sources.Sources = append(s.appCfg.Sources.Sources, src)

	if err := s.saveSources(); err != nil {
		log.Error().Err(err).Msg("save sources failed")
		writeError(w, http.StatusInternalServerError, "failed to save sources")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"id": src.ID})
}

func (s *Server) handleUpdateSource(w http.ResponseWriter, r *http.Request) {
	type sourceUpdate struct {
		ID           string   `json:"id"`
		Name         string   `json:"name,omitempty"`
		URL          string   `json:"url,omitempty"`
		Enabled      *bool    `json:"enabled,omitempty"`
		ScheduleFreq string   `json:"schedule_freq,omitempty"`
		TopicIDs     []string `json:"topic_ids,omitempty"`
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var body sourceUpdate
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	body.ID = strings.TrimSpace(body.ID)
	if body.ID == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	found := false
	for idx, src := range s.appCfg.Sources.Sources {
		if src.ID != body.ID {
			continue
		}

		if body.Name != "" {
			s.appCfg.Sources.Sources[idx].Name = strings.TrimSpace(body.Name)
		}
		if body.URL != "" {
			parsed, err := url.Parse(strings.TrimSpace(body.URL))
			if err != nil || parsed.Hostname() == "" {
				writeError(w, http.StatusBadRequest, "url must be a valid absolute URL")
				return
			}
			s.appCfg.Sources.Sources[idx].RSSURL = strings.TrimSpace(body.URL)
			s.appCfg.Sources.Sources[idx].Domain = parsed.Hostname()
		}
		if body.Enabled != nil {
			s.appCfg.Sources.Sources[idx].Enabled = *body.Enabled
		}
		if body.ScheduleFreq != "" {
			s.appCfg.Sources.Sources[idx].ScheduleFreq = strings.TrimSpace(body.ScheduleFreq)
		}
		if body.TopicIDs != nil {
			s.appCfg.Sources.Sources[idx].TopicIDs = normalizeTopicIDs(body.TopicIDs)
		}

		found = true
		break
	}

	if !found {
		writeError(w, http.StatusNotFound, "source not found")
		return
	}

	if err := s.saveSources(); err != nil {
		log.Error().Err(err).Msg("save sources failed")
		writeError(w, http.StatusInternalServerError, "failed to save sources")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
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
	idStr := pathParam(r, "/api/documents/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid document id")
		return
	}

	doc, err := s.document.GetDocumentWithKeywords(r.Context(), id, s.appCfg.Topics)
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
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

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
	workflowID := pathParam(r, "/api/workflows/")
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

func (s *Server) handleGetDashboard(w http.ResponseWriter, r *http.Request) {
	dayTopicCounts, err := s.stats.GetDocumentCountsByDayTopic(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("handleGetDashboard: day-topic counts")
		writeError(w, http.StatusInternalServerError, "failed to compute dashboard metrics")
		return
	}

	topicCounts, err := s.stats.GetDocumentCountsByTopic(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("handleGetDashboard: topic counts")
		writeError(w, http.StatusInternalServerError, "failed to compute dashboard metrics")
		return
	}

	topSources, err := s.stats.GetTopSources(r.Context(), 10)
	if err != nil {
		log.Error().Err(err).Msg("handleGetDashboard: top sources")
		writeError(w, http.StatusInternalServerError, "failed to compute dashboard metrics")
		return
	}

	failCounts, err := s.stats.GetSourceFailCounts(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("handleGetDashboard: source fail counts")
		writeError(w, http.StatusInternalServerError, "failed to compute dashboard metrics")
		return
	}

	lastPosts, err := s.stats.GetSourceLastPostTimes(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("handleGetDashboard: last post times")
		writeError(w, http.StatusInternalServerError, "failed to compute dashboard metrics")
		return
	}

	failed, processed, err := s.stats.GetQueueFailureMetrics(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("handleGetDashboard: queue failure metrics")
		writeError(w, http.StatusInternalServerError, "failed to compute dashboard metrics")
		return
	}

	topicNameMap := map[string]string{}
	for _, topic := range s.appCfg.Topics.Topics {
		topicNameMap[topic.ID] = topic.Name
	}

	sourceNameMap := map[string]string{}
	for _, source := range s.appCfg.Sources.Sources {
		sourceNameMap[source.ID] = source.Name
	}

	countsByTopic := make([]dashboardTopicCountDTO, 0, len(topicCounts))
	for _, item := range topicCounts {
		countsByTopic = append(countsByTopic, dashboardTopicCountDTO{
			TopicID:   item.TopicID,
			TopicName: topicNameMap[item.TopicID],
			Count:     item.Count,
		})
	}

	countsByDay := make([]dashboardDayCountDTO, 0, len(dayTopicCounts))
	for _, item := range dayTopicCounts {
		countsByDay = append(countsByDay, dashboardDayCountDTO{
			Date:      item.Date,
			TopicID:   item.TopicID,
			TopicName: topicNameMap[item.TopicID],
			Count:     item.Count,
		})
	}

	topSourceDTOs := make([]dashboardSourceDTO, 0, len(topSources))
	for _, item := range topSources {
		topSourceDTOs = append(topSourceDTOs, dashboardSourceDTO{
			SourceID: item.SourceID,
			Name:     sourceNameMap[item.SourceID],
			Count:    item.Count,
		})
	}

	failMap := map[string]int64{}
	for _, item := range failCounts {
		failMap[item.SourceID] = item.FailCount
	}

	lastPostMap := map[string]*time.Time{}
	for _, item := range lastPosts {
		lastPostMap[item.SourceID] = item.LastPostAt
	}

	slaEntries := make([]dashboardSLAEntryDTO, 0, len(s.appCfg.Sources.Sources))
	for _, source := range s.appCfg.Sources.Sources {
		lastPostAt := lastPostMap[source.ID]
		ageDays := -1
		if lastPostAt != nil {
			ageDays = int(time.Since(*lastPostAt).Hours() / 24)
		}
		stale := lastPostAt == nil || time.Since(*lastPostAt) > 5*24*time.Hour
		slaEntries = append(slaEntries, dashboardSLAEntryDTO{
			SourceID:          source.ID,
			Name:              source.Name,
			Enabled:           source.Enabled,
			LastPostAt:        lastPostAt,
			FailCount:         failMap[source.ID],
			DaysSinceLastPost: ageDays,
			Stale:             stale,
		})
	}

	sort.SliceStable(slaEntries, func(i, j int) bool {
		if slaEntries[i].FailCount != slaEntries[j].FailCount {
			return slaEntries[i].FailCount > slaEntries[j].FailCount
		}
		return slaEntries[i].DaysSinceLastPost > slaEntries[j].DaysSinceLastPost
	})

	failRate := 0.0
	if processed > 0 {
		failRate = float64(failed) / float64(processed)
	}

	writeJSON(w, http.StatusOK, dashboardOverviewDTO{
		CountsByTopic: countsByTopic,
		CountsByDay:   countsByDay,
		TopSources:    topSourceDTOs,
		FailRate:      failRate,
		SLASources:    slaEntries,
	})
}

func (s *Server) handleListNearDuplicates(w http.ResponseWriter, r *http.Request) {
	limit := 200
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	threshold := 5
	if d := r.URL.Query().Get("distance"); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n >= 0 && n <= 64 {
			threshold = n
		}
	}

	docs, err := s.stats.ListRecentSimhashDocuments(r.Context(), limit)
	if err != nil {
		log.Error().Err(err).Msg("handleListNearDuplicates")
		writeError(w, http.StatusInternalServerError, "failed to list near duplicates")
		return
	}

	n := len(docs)
	if n < 2 {
		writeJSON(w, http.StatusOK, []struct{}{})
		return
	}

	parent := make([]int, n)
	rank := make([]int, n)
	for i := range parent {
		parent[i] = i
	}

	var find func(int) int
	find = func(i int) int {
		if parent[i] != i {
			parent[i] = find(parent[i])
		}
		return parent[i]
	}

	union := func(a, b int) {
		ra, rb := find(a), find(b)
		if ra == rb {
			return
		}
		if rank[ra] < rank[rb] {
			ra, rb = rb, ra
		}
		parent[rb] = ra
		if rank[ra] == rank[rb] {
			rank[ra]++
		}
	}

	maxDistance := make(map[int]int)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			distance := hammingDistance(docs[i].ContentSimHash, docs[j].ContentSimHash)
			if distance <= threshold {
				union(i, j)
				root := find(i)
				if maxDistance[root] < distance {
					maxDistance[root] = distance
				}
			}
		}
	}

	groupsByRoot := make(map[int][]model.NearDuplicateDoc)
	for i, doc := range docs {
		root := find(i)
		groupsByRoot[root] = append(groupsByRoot[root], doc)
	}

	type duplicateGroupDTO struct {
		Docs        []model.NearDuplicateDoc `json:"docs"`
		MaxDistance int                      `json:"max_distance"`
	}

	groups := make([]duplicateGroupDTO, 0)
	for root, groupDocs := range groupsByRoot {
		if len(groupDocs) < 2 {
			continue
		}
		groups = append(groups, duplicateGroupDTO{
			Docs:        groupDocs,
			MaxDistance: maxDistance[root],
		})
	}

	sort.SliceStable(groups, func(i, j int) bool {
		if len(groups[i].Docs) != len(groups[j].Docs) {
			return len(groups[i].Docs) > len(groups[j].Docs)
		}
		return groups[i].MaxDistance < groups[j].MaxDistance
	})

	writeJSON(w, http.StatusOK, groups)
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

func (s *Server) handleLabelQueueAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	parts := pathSegments(r, "/api/label-queue/")
	if len(parts) != 2 {
		writeError(w, http.StatusBadRequest, "invalid label queue action")
		return
	}

	switch parts[1] {
	case "label":
		s.handleSubmitLabel(w, r)
	case "skip":
		s.handleSkipLabelQueue(w, r)
	default:
		writeError(w, http.StatusBadRequest, "invalid label queue action")
	}
}

// handleSubmitLabel records a gold label for a queue item.
//
//	POST /api/label-queue/{id}/label
//	Body: {"topic_id":"...", "labeled_by":"..."}
func (s *Server) handleSubmitLabel(w http.ResponseWriter, r *http.Request) {
	parts := pathSegments(r, "/api/label-queue/")
	if len(parts) != 2 || parts[1] != "label" {
		writeError(w, http.StatusBadRequest, "invalid label action")
		return
	}
	queueID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid queue item id")
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var body struct {
		TopicID   string `json:"topic_id"`
		LabeledBy string `json:"labeled_by"`
	}
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
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
	parts := pathSegments(r, "/api/label-queue/")
	if len(parts) != 2 || parts[1] != "skip" {
		writeError(w, http.StatusBadRequest, "invalid skip action")
		return
	}
	queueID, err := strconv.ParseInt(parts[0], 10, 64)
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
