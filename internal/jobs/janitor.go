package jobs

import (
	"log"
	"os"
	"time"
	"ytdl-server/internal/config"
)

func StartJanitor(cfg *config.Config) {
	ticker := time.NewTicker(cfg.CleanupAfter)

	go func() {
		for range ticker.C {
			log.Println("🧹 Janitor: Starting scheduled cleanup...")

			err := os.RemoveAll(cfg.TempDir)
			if err != nil {
				log.Printf("❌ Janitor Error: Could not clear temp: %v", err)
			}

			os.MkdirAll(cfg.TempDir, 0755)

			log.Println("✅ Janitor: Cleanup finished.")
		}
	}()
}
