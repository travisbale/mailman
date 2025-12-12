package http

import (
	"context"
	"net/http"
	"time"
)

// healthChecker defines the interface for checking service health
type healthChecker interface {
	Health(ctx context.Context) error
}

// Config holds HTTP server configuration
type Config struct {
	Address string
	DB      healthChecker
}

// Server represents the HTTP server
type Server struct {
	*http.Server
}

// NewServer creates a new HTTP server
func NewServer(config *Config) *Server {
	// Create domain handlers
	healthHandler := &HealthHandler{
		DB: config.DB,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthHandler.Health)

	return &Server{
		Server: &http.Server{
			Addr:              config.Address,
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second, // Prevents Slowloris attacks
		},
	}
}

// Shutdown gracefully shuts down the HTTP server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.Server.Shutdown(ctx)
}
