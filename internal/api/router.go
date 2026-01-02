package api

import (
	"net/http"
)

func NewRouter(h *Handler) http.Handler {
    mux := http.NewServeMux()

    mux.HandleFunc("/api/job", h.CreateJob)
    mux.HandleFunc("/api/info", h.GetVideoInfo) // <-- h.CreateJob yerine h.GetVideoInfo
    mux.HandleFunc("/api/events/", h.SSE)
    mux.HandleFunc("/api/download/", h.Download)

    return CORSMiddleware(mux)
}
