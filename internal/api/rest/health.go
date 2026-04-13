package rest

import (
	"context"
	"net/http"
	"time"
)

// handleHealth checks database connectivity and returns health status
func (r *Router) handleHealth(w http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), 2*time.Second)
	defer cancel()

	if err := r.DB.Health(ctx); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
}
