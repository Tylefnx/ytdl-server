# --- Stage 1: Build ---
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache make git

WORKDIR /app

# Copy go.mod and go.sum first to leverage Docker cache for dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the binary using your existing Makefile logic
RUN make build

# --- Stage 2: Runtime ---
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    ffmpeg \
    ca-certificates \
    tzdata \

WORKDIR /app

# Copy only the compiled binary and the generated .env from the builder
COPY --from=builder /app/ytdl-server /app/ytdl-server
COPY --from=builder /app/.env /app/.env

# Create directories defined in your .ytdl-config
# Note: Ensure your .ytdl-config paths match these for Docker (e.g., ./downloads)
RUN mkdir -p /app/temp /app/downloads

# Security: Run as a non-privileged system user (daemon)
RUN chown -R daemon:daemon /app

# Standard port (should match your .ytdl-config PORT defined in .ytdl-config)
EXPOSE 8080

# Switch to unprivileged user
USER daemon

# Start the application
CMD ["./ytdl-server"]