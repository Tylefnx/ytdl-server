package api

import (
	"net/http"
    "mime"
)

func NewRouter(h *Handler) http.Handler {
    mime.AddExtensionType(".js", "application/javascript")
    mime.AddExtensionType(".wasm", "application/wasm")
    mux := http.NewServeMux()

    mux.HandleFunc("/api/job", h.CreateJob)
    mux.HandleFunc("/api/info", h.GetVideoInfo) // <-- h.CreateJob yerine h.GetVideoInfo
    mux.HandleFunc("/api/events/", h.SSE)
    mux.HandleFunc("/api/download/", h.Download)

    fs := http.FileServer(http.Dir("./web")) 
    mux.Handle("/", fs)

    return CORSMiddleware(mux)
}
