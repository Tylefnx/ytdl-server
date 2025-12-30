YTDL-Server: Robust Media Ingestion Backend

A high-performance, concurrent YouTube-to-MP4 processing server written in Go. Resolves DASH streams internally and muxes them via FFmpeg. Built for stability across low-power home servers and high-core-count workstations.
🚀 Key Features

    Native Resolution: Handles YouTube stream resolution internally via Go (no yt-dlp dependency).

    Deterministic Muxing: Fetches audio and video streams independently for precise DASH-based MP4 assembly.

    Bounded Concurrency: Uses a managed worker queue to prevent CPU, I/O, or memory exhaustion.

    Multi-Platform Automation: Unified Makefile detects and configures Gentoo (OpenRC/Systemd) and FreeBSD (rc.d).

    Automatic Janitor: Background goroutine purges stale or abandoned temporary files periodically.

🛠️ Prerequisites

    Go 1.21+

    FFmpeg (compiled with libx264 support)

    gmake (Required on FreeBSD) or make (On Linux)

📦 Installation & Setup
1. Initialize Configuration

The project uses a Single Source of Truth model. You only edit .ytdl-config. The system handles the rest.
Bash

cp .ytdl-config.example .ytdl-config
# Set your paths, service user, CORS origins, and resource limits
nano .ytdl-config

2. Automated Setup

The Makefile validates your configuration and generates the runtime .env file required by both the binary and Docker:
Bash

make setup

3. Install to System

Compiles the binary, sets directory permissions for the SERVICE_USER, and registers the service script. Note: Requires root privileges.
Bash

make install

4. Service Management

Gentoo (OpenRC):
Bash

rc-update add ytdl-server default
rc-service ytdl-server start

FreeBSD:
Bash

sysrc ytdl_server_enable=YES
service ytdl_server start


🐍 Client Integration Example

The following Python script demonstrates the full workflow: acquiring a job ticket, monitoring real-time progress via Server-Sent Events (SSE), and downloading the final MP4 file.
Python
```
import requests
import json
import sys

# --- CONFIGURATION ---
SERVER_URL = "http://localhost:8080"
VIDEO_ID = "<video-id-here>" 
QUALITY = "1080p"         

def main():
    print(f">>> 🎬 Sending Request: {VIDEO_ID} [{QUALITY}]")
    
    # STEP 1: Initialize Job (Get Ticket)
    try:
        resp = requests.post(f"{SERVER_URL}/api/job", json={
            "video_id": VIDEO_ID,
            "quality": QUALITY
        })
        resp.raise_for_status()
        job_id = resp.json()["job_id"]
        print(f">>> 🎫 Ticket Acquired! Job ID: {job_id}")
    except Exception as e:
        print(f"❌ Server Error: {e}"); return

    # STEP 2: Live Monitoring (SSE)
    print(">>> ⏳ Processing...")
    sse_url = f"{SERVER_URL}/api/events/{job_id}"
    
    with requests.get(sse_url, stream=True) as event_stream:
        for line in event_stream.iter_lines():
            if line:
                decoded_line = line.decode('utf-8')
                if decoded_line.startswith("data:"):
                    job_data = json.loads(decoded_line.replace("data: ", ""))
                    status, percent = job_data["status"], job_data["percentage"]
                    
                    draw_progress_bar(percent, status)
                    
                    if status == "ready":
                        print("\n>>> ✅ Success!")
                        download_file(job_data["filename"], f"{SERVER_URL}/api/download/{job_id}")
                        break
                    if status == "failed":
                        print(f"\n❌ Error: {job_data.get('error')}"); break

def draw_progress_bar(percent, status):
    bar_len = 30
    filled_len = int(bar_len * percent // 100)
    msg = "Finalizing (Muxing)..." if percent >= 99 else "Processing..."
    bar = '█' * filled_len + '░' * (bar_len - filled_len)
    sys.stdout.write(f"\r[{bar}] %{percent:.1f} - {msg}   ")
    sys.stdout.flush()

def download_file(filename, url):
    print(f">>> ⬇️  Downloading: {filename}")
    with requests.get(url, stream=True) as r:
        with open(filename, 'wb') as f:
            for chunk in r.iter_content(chunk_size=8192): f.write(chunk)
    print(f">>> 🎉 Saved: {filename}")

if __name__ == "__main__":
    main()
```

📂 Project Structure
Plaintext

```

.
├── cmd/
│   └── server/
│       └── main.go           # Entry Point (Bootstrap & Orchestration)
├── internal/
│   ├── api/
│   │   ├── handler.go        # HTTP logic
│   │   ├── middleware.go     # CORS parsing & Logic
│   │   └── router.go         # Endpoint & Middleware mapping
│   ├── config/
│   │   └── config.go         # .env parsing
│   ├── downloader/
│   │   └── engine.go         # YouTube resolution & FFmpeg muxing
│   ├── jobs/
│   │   ├── manager.go        # Concurrent queue & Worker pool
│   │   └── janitor.go        # Automated file cleanup
│   ├── models/
│   │   └── job.go            # Global Structs
│   └── server/
│       └── filesystem.go     # OS-level directory preparation
├── Makefile                  # Cross-platform build & Setup automation
├── .ytdl-config.example      # Master config template
└── README.md                 # Project documentation

```

🛡️ Security & Hardening

    Dynamic CORS Policy: Controlled via ALLOWED_ORIGINS in .ytdl-config.

        Note: In a production environment, avoid using *. Explicitly define your frontend domain (e.g., https://media.mydomain.com) to prevent Cross-Origin hijacking.

    Unprivileged Execution: The service is designed to run under the user defined in .ytdl-config (default: daemon).

    Network Isolation: Only the configured listening port is exposed. It is highly recommended to use a reverse proxy (Nginx/Caddy) for SSL termination and authentication.

    ZFS Quotas (FreeBSD): Strongly recommended to prevent disk exhaustion on shared pools: zfs set quota=50G zpool/dataset/ytdl-data

🐳 Docker Deployment

The project uses a Multi-Stage Dockerfile to ensure a small footprint and maximum security. The image consumes the .env file generated during the make setup phase.
1. The Dockerfile

Create a file named Dockerfile in your root directory:
Dockerfile

# --- Stage 1: Build ---
FROM golang:1.21-alpine AS builder
RUN apk add --no-cache make git
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN make build

# --- Stage 2: Runtime ---
FROM alpine:latest
RUN apk add --no-cache ffmpeg ca-certificates tzdata
WORKDIR /app
# Copy binary and generated config from builder
COPY --from=builder /app/ytdl-server .
COPY --from=builder /app/.env .
# Create storage directories and set permissions
RUN mkdir -p temp downloads && chown -R daemon:daemon /app
USER daemon
EXPOSE 8080
CMD ["./ytdl-server"]

2. Docker Compose

Use docker-compose to manage volumes and environment injection:
YAML

services:
  ytdl-server:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - /path/to/downloads:/app/downloads
    env_file: .env
    restart: unless-stopped

3. Execution

Ensure you have run the setup on the host first to transform your .ytdl-config into the environment:
Bash

# 1. Generate the .env on the host
make setup

# 2. Build and run the container
docker-compose up -d

🛡️ Docker Security Hardening

    Read-Only Root FS: You can run the container with a read-only filesystem for extra security, provided you mount the downloads and temp folders as writable volumes.

    Resource Limits: In your compose file, limit memory usage (e.g., mem_limit: 512m) to ensure the host remains responsive during heavy FFmpeg muxing.

🧹 Maintenance & Monitoring

    Janitor Service: Automated cleanup of TEMP_DIR every X minutes (configurable interval defined in config).

    Logs: Written to the configured data directory. Monitor progress with: tail -f /your/data/path/output.log

🛠️ Contribution & Development

    Test locally: make build && ./ytdl-server

    Cross-compile (FreeBSD): make build-freebsd

    Full Deploy: make install