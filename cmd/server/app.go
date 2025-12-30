package server

import (
	"os"
	"ytdl-server/internal/config"
)

// PrepareFilesystem ensures necessary directories exist
func PrepareFilesystem(cfg *config.Config) error {
	dirs := []string{cfg.DownloadDir, cfg.TempDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}
