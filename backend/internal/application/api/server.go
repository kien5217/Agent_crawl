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
	document     repository.DocumentRepository
	workflow     repository.WorkflowRepository
	health       repository.HealthRepository
	labeling     repository.LabelingRepository
	scheduleFlow func(context.Context) (*orchestration.RunResult, error)
}

// NewServer constructs an API server.
func NewServer(
	addr string,
	appCfg *config.AppConfig,
	document repository.DocumentRepository,
	workflow repository.WorkflowRepository,
	health repository.HealthRepository,
	labeling repository.LabelingRepository,
	scheduleFlow func(context.Context) (*orchestration.RunResult, error),
) *Server {
	return &Server{
		addr:         addr,
		appCfg:       appCfg,
		document:     document,
		workflow:     workflow,
		health:       health,
		labeling:     labeling,
		scheduleFlow: scheduleFlow,
	}
}

// routes registers all HTTP routes and returns the handler.
func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	// REST API endpoints
	mux.HandleFunc("GET /api/topics", s.handleListTopics)
	mux.HandleFunc("GET /api/documents", s.handleListDocuments)
	mux.HandleFunc("GET /api/documents/{id}", s.handleGetDocument)
	mux.Handle("POST /api/schedule", s.requireWriteAuth(http.HandlerFunc(s.handleTriggerSchedule)))
	mux.HandleFunc("GET /api/workflows", s.handleListWorkflows)
	mux.HandleFunc("GET /api/workflows/{id}/steps", s.handleListSteps)
	mux.HandleFunc("GET /api/health", s.handleGetHealth)
	mux.HandleFunc("GET /api/label-queue", s.handleListLabelQueue)
	mux.Handle("POST /api/label-queue/{id}/label", s.requireWriteAuth(http.HandlerFunc(s.handleSubmitLabel)))
	mux.Handle("POST /api/label-queue/{id}/skip", s.requireWriteAuth(http.HandlerFunc(s.handleSkipLabelQueue)))

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
