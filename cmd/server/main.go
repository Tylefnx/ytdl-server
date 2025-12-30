package main

import (
	"fmt"
	"log"
	"net/http"

	"ytdl-server/internal/api"
	"ytdl-server/internal/config"
	"ytdl-server/internal/jobs"
	"ytdl-server/internal/server"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	cfg := config.Load()

	// 1. Hazırlık: Dosya sistemi
	if err := server.PrepareFilesystem(cfg); err != nil {
		log.Fatalf(">>> ❌ Error preparing filesystem: %v", err)
	}

	// 2. Servisler: Job Manager ve Handler
	jobManager := jobs.NewManager(cfg)
	handler := api.NewHandler(jobManager)

	// 3. Router: Middleware dahil edilmiş haliyle
	router := api.NewRouter(handler)

	fmt.Println(">>> 🏭 YTDL Server Started")
	fmt.Printf(">>> ⚡ Port: %s\n", cfg.Port)

	// 4. Start
	log.Fatal(http.ListenAndServe(cfg.Port, router))
}
