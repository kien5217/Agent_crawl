package api

import (
	"context"
	"net/http"

	appschedule "Agent_Crawl/internal/application/schedule"
	"Agent_Crawl/internal/domain/config"
	"Agent_Crawl/internal/domain/repository"

	"github.com/rs/zerolog/log"
)

// Server holds all dependencies for the HTTP API.
type Server struct {
	addr          string
	appCfg        *config.AppConfig
	document      repository.DocumentRepository
	workflow      repository.WorkflowRepository
	scheduler     *appschedule.Service
	afterSchedule func(context.Context) error
}

// NewServer constructs an API server.
func NewServer(
	addr string,
	appCfg *config.AppConfig,
	document repository.DocumentRepository,
	workflow repository.WorkflowRepository,
	scheduler *appschedule.Service,
	afterSchedule func(context.Context) error,
) *Server {
	return &Server{
		addr:          addr,
		appCfg:        appCfg,
		document:      document,
		workflow:      workflow,
		scheduler:     scheduler,
		afterSchedule: afterSchedule,
	}
}

// routes registers all HTTP routes and returns the handler.
func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	// REST API endpoints
	mux.HandleFunc("GET /api/topics", s.handleListTopics)
	mux.HandleFunc("GET /api/documents", s.handleListDocuments)
	mux.HandleFunc("GET /api/documents/{id}", s.handleGetDocument)
	mux.HandleFunc("POST /api/schedule", s.handleTriggerSchedule)
	mux.HandleFunc("GET /api/workflows", s.handleListWorkflows)
	mux.HandleFunc("GET /api/workflows/{id}/steps", s.handleListSteps)

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
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
