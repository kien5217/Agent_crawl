package api

import (
	"context"
	"net/http"

	orchestration "Agent_Crawl/internal/application/orchestrator"
	"Agent_Crawl/internal/domain/config"
	"Agent_Crawl/internal/domain/repository"

	"github.com/rs/zerolog/log"
)

// Server holds all dependencies for the HTTP API.
type Server struct {
	addr         string
	appCfg       *config.AppConfig
	topicsPath   string
	sourcesPath  string
	document     repository.DocumentRepository
	workflow     repository.WorkflowRepository
	health       repository.HealthRepository
	stats        repository.StatsRepository
	labeling     repository.LabelingRepository
	scheduleFlow func(context.Context) (*orchestration.RunResult, error)
}

// NewServer constructs an API server.
func NewServer(
	addr string,
	appCfg *config.AppConfig,
	topicsPath string,
	sourcesPath string,
	document repository.DocumentRepository,
	workflow repository.WorkflowRepository,
	health repository.HealthRepository,
	stats repository.StatsRepository,
	labeling repository.LabelingRepository,
	scheduleFlow func(context.Context) (*orchestration.RunResult, error),
) *Server {
	return &Server{
		addr:         addr,
		appCfg:       appCfg,
		topicsPath:   topicsPath,
		sourcesPath:  sourcesPath,
		document:     document,
		workflow:     workflow,
		health:       health,
		stats:        stats,
		labeling:     labeling,
		scheduleFlow: scheduleFlow,
	}
}

// routes registers all HTTP routes and returns the handler.
func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	// REST API endpoints
	mux.HandleFunc("/api/topics", s.handleTopics)
	mux.HandleFunc("/api/sources", s.handleSources)
	mux.HandleFunc("/api/documents/near-duplicates", s.handleListNearDuplicates)
	mux.HandleFunc("/api/documents", s.handleListDocuments)
	mux.HandleFunc("/api/documents/", s.handleGetDocument)
	mux.Handle("/api/schedule", s.requireWriteAuth(http.HandlerFunc(s.handleTriggerSchedule)))
	mux.HandleFunc("/api/workflows", s.handleListWorkflows)
	mux.HandleFunc("/api/workflows/", s.handleListSteps)
	mux.HandleFunc("/api/health", s.handleGetHealth)
	mux.HandleFunc("/api/dashboard", s.handleGetDashboard)
	mux.HandleFunc("/api/label-queue", s.handleListLabelQueue)
	mux.Handle("/api/label-queue/", s.requireWriteAuth(http.HandlerFunc(s.handleLabelQueueAction)))
	// Serve compiled React frontend from ../frontend/dist (relative to backend/)
	mux.Handle("/", http.FileServer(http.Dir("../frontend/dist")))

	return corsMiddleware(mux)
}

// Run starts the HTTP server and blocks until an error occurs.
func (s *Server) Run() error {
	log.Info().Str("addr", s.addr).Msg("HTTP server listening")
	return http.ListenAndServe(s.addr, s.routes())
}

// corsMiddleware adds permissive CORS headers for local React dev server.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
