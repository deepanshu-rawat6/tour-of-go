# --- Stage 1: Build ---
FROM golang:1.25-alpine3.22 AS builder

# Install build dependencies (if your app uses CGO or needs git for private modules)
RUN apk add --no-cache git

WORKDIR /app

# 1. Cache dependencies (improves build speed)
COPY go.mod go.sum ./
RUN go mod download

# 2. Copy source and build
COPY . .
# CGO_ENABLED=0 ensures a static binary that works perfectly on Alpine
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o reaper ./cmd/reaper

# --- Stage 2: Final ---
FROM alpine:3.22

# 3. Security: Run as a non-root user
RUN addgroup -S reapergroup && adduser -S reaperuser -G reapergroup
USER reaperuser

WORKDIR /home/reaperuser

# Copy binary and config
COPY --from=builder /app/reaper /usr/local/bin/reaper
COPY --from=builder /app/config.example.yaml /etc/reaper/config.yaml

ENTRYPOINT ["reaper"]