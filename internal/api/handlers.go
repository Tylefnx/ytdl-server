package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
	"github.com/kkdai/youtube/v2"
	"ytdl-server/internal/jobs"
	"ytdl-server/internal/models"
    "sort"
    "strconv"
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


func (h *Handler) GetVideoInfo(w http.ResponseWriter, r *http.Request) {
    videoID := r.URL.Query().Get("video_id")
    if videoID == "" {
        http.Error(w, "video_id required", http.StatusBadRequest)
        return
    }

    client := youtube.Client{}
    video, err := client.GetVideo(videoID)
    if err != nil {
        http.Error(w, "Video bilgileri alınamadı", 500)
        return
    }

    qualityMap := make(map[int]string)

    for _, f := range video.Formats {
        if strings.Contains(f.MimeType, "video") && f.QualityLabel != "" {
            height := localParseHeightOnly(f.QualityLabel)
            
            if height > 0 {
                // Etiketi formatlayalım: "1080p60" -> "1080p 60fps"
                label := formatQualityLabel(f.QualityLabel)
                qualityMap[height] = label
            }
        }
    }

    // Sıralama...
    var heights []int
    for h := range qualityMap {
        heights = append(heights, h)
    }
    sort.Sort(sort.Reverse(sort.IntSlice(heights)))

    var sortedQualities []string
    for _, h := range heights {
        sortedQualities = append(sortedQualities, qualityMap[h])
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "qualities": sortedQualities,
        "title":     video.Title,
    })
}

// Yeni yardımcı fonksiyon: Etiketi güzelleştirir
func formatQualityLabel(q string) string {
    // Örn: "1080p60" -> "1080p" ve "60" olarak ayırır
    re := regexp.MustCompile(`^(\d+p)(\d+)?$`)
    matches := re.FindStringSubmatch(q)

    if len(matches) > 1 {
        base := matches[1] // "1080p"
        if len(matches) > 2 && matches[2] != "" {
            return fmt.Sprintf("%s %sfps", base, matches[2]) // "1080p 60fps"
        }
        return base // Sadece "720p"
    }
    return q // Eşleşmezse orijinali dön
}

func localParseHeightOnly(q string) int {
    digits := ""
    for _, c := range q {
        if c >= '0' && c <= '9' {
            digits += string(c)
        } else if digits != "" {
            break
        }
    }
    val, _ := strconv.Atoi(digits)
    return val
}
