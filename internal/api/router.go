package api

import (
	"net/http"
)

// NewRouter setup routes and apply global middleware
func NewRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/job", h.CreateJob)
	mux.HandleFunc("/api/events/", h.SSE)
	mux.HandleFunc("/api/download/", h.Download)

	// Wrap everything with our robust CORS logic
	return CORSMiddleware(mux)
}
