package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"ytdl-server/internal/jobs"
	"ytdl-server/internal/models"
)

var videoIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{11}$`)

type Handler struct {
	Manager *jobs.Manager
}

func NewHandler(m *jobs.Manager) *Handler {
	return &Handler{Manager: m}
}

func (h *Handler) CreateJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req models.CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if !videoIDRegex.MatchString(req.VideoID) {
		http.Error(w, "Invalid Video ID", http.StatusBadRequest)
		return
	}
	if req.Quality == "" {
		req.Quality = "1080p"
	}

	job := h.Manager.Create(req)

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"job_id":       job.ID,
		"status":       "pending",
		"stream_url":   fmt.Sprintf("/api/events/%s", job.ID),
		"download_url": fmt.Sprintf("/api/download/%s", job.ID),
	})
}

func (h *Handler) SSE(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		return
	}
	jobID := parts[3]

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")

	rc := http.NewResponseController(w)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			job, ok := h.Manager.Get(jobID)
			if !ok {
				fmt.Fprintf(w, "event: error\ndata: Job not found\n\n")
				rc.Flush()
				return
			}
			data, _ := json.Marshal(job)
			fmt.Fprintf(w, "data: %s\n\n", data)
			rc.Flush()

			if job.Status == "ready" || job.Status == "failed" {
				return
			}
		}
	}
}

func (h *Handler) Download(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		return
	}
	jobID := parts[3]

	job, ok := h.Manager.Get(jobID)
	if !ok {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}
	if job.Status != "ready" {
		http.Error(w, "Not ready", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", job.Filename))
	w.Header().Set("Content-Type", "video/mp4")
	http.ServeFile(w, r, job.FilePath)
}
