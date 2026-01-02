package models

import (
	"time"
)

// Job: Bir indirme işleminin tüm durumunu tutar
type Job struct {
	ID         string    `json:"id"`
	VideoID    string    `json:"video_id"`
	Status     string    `json:"status"` // "pending", "processing", "ready", "failed"
	Percentage float64   `json:"percentage"`
	Filename   string    `json:"filename"`
	FilePath   string    `json:"-"`
	Error      string    `json:"error,omitempty"`
	CreatedAt  time.Time `json:"-"`
}

type CreateJobRequest struct {
	VideoID string `json:"video_id"`
	Quality string `json:"quality"`
}