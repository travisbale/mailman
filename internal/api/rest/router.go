package rest

import (
	"context"
	"net/http"
	"sync"
)

type database interface {
	Health(ctx context.Context) error
}

// Router holds all HTTP handler dependencies in a single struct.
// Implements http.Handler — routes and middleware are initialized on first request.
type Router struct {
	DB database

	once    sync.Once
	handler http.Handler
}

// ServeHTTP implements http.Handler. Routes and middleware are initialized on first request.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.once.Do(r.init)
	r.handler.ServeHTTP(w, req)
}

func (r *Router) init() {
	mux := http.NewServeMux()
	r.registerRoutes(mux)

	r.handler = mux
}

// registerRoutes configures all HTTP routes with their handlers
func (r *Router) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("HEAD /healthz", r.handleHealth)
}
