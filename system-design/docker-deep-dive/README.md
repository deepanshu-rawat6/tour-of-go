# Docker Deep Dive

Layer caching, `.dockerignore`, health checks, docker-compose for local dev, and debugging containers.

---

## Image Layers

```mermaid
graph TD
    BASE[FROM golang:1.23-alpine<br>~300MB shared base] --> DEPS[COPY go.mod go.sum<br>RUN go mod download<br>~100MB cached if go.mod unchanged]
    DEPS --> SRC[COPY . .<br>~10MB invalidated on any code change]
    SRC --> BUILD[RUN go build<br>~15MB binary]
    BUILD --> FINAL[FROM distroless<br>COPY binary<br>~20MB final image]
```

**Key insight**: Each Dockerfile instruction creates a layer. Layers are cached — order instructions from least-changing to most-changing.

---

## Layer Caching Optimization

```dockerfile
# ❌ BAD: any code change invalidates dep download
FROM golang:1.23-alpine
COPY . .
RUN go mod download
RUN go build -o /app .

# ✅ GOOD: deps cached separately from source
FROM golang:1.23-alpine AS builder
WORKDIR /app

# Layer 1: deps (cached unless go.mod changes)
COPY go.mod go.sum ./
RUN go mod download

# Layer 2: source (invalidated on code change, but deps cached)
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /app/server ./cmd/server

# Layer 3: minimal runtime
FROM gcr.io/distroless/static:nonroot
COPY --from=builder /app/server /server
USER nonroot:nonroot
ENTRYPOINT ["/server"]
```

---

## .dockerignore

```mermaid
graph LR
    SRC[Project Dir<br>500MB] -->|without .dockerignore| CTX1[Build Context<br>500MB sent to daemon]
    SRC -->|with .dockerignore| CTX2[Build Context<br>5MB sent to daemon<br>.git excluded]
    CTX2 --> FAST[Faster builds]
```

```gitignore
# .dockerignore — reduces build context size
.git
.github
*.md
docs/
bin/
tmp/
.env*
docker-compose*.yml
Makefile
**/*_test.go
```

**Why it matters**: Docker sends the entire build context to the daemon. A 500MB `.git` directory slows every build.

---

## Health Checks

```dockerfile
HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/server", "-health-check"]
```

```go
// In your Go app: lightweight health endpoint
func healthHandler(w http.ResponseWriter, _ *http.Request) {
    // Check critical dependencies
    if err := db.PingContext(context.Background()); err != nil {
        w.WriteHeader(503)
        fmt.Fprint(w, `{"status":"unhealthy","db":"down"}`)
        return
    }
    fmt.Fprint(w, `{"status":"healthy"}`)
}
```

---

## Docker Compose for Local Dev

```mermaid
graph TD
    COMPOSE[docker-compose.yml] --> APP[app<br>Go service :8080]
    COMPOSE --> PG[postgres<br>:5432]
    COMPOSE --> REDIS[redis<br>:6379]
    
    APP -->|DATABASE_URL| PG
    APP -->|REDIS_URL| REDIS
    APP -->|volume mount| SRC[./src → /app<br>hot reload]
    
    PG -->|healthcheck| APP
    PG --> VOL[(pgdata volume<br>persistent)]
```

```yaml
# docker-compose.yml
services:
  app:
    build:
      context: .
      target: builder  # use build stage for hot-reload
    ports: ["8080:8080"]
    volumes:
      - .:/app  # mount source for live reload
    environment:
      - DATABASE_URL=postgres://dev:dev@postgres:5432/mydb?sslmode=disable
      - REDIS_URL=redis:6379
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_started

  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: dev
      POSTGRES_PASSWORD: dev
      POSTGRES_DB: mydb
    ports: ["5432:5432"]
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U dev"]
      interval: 5s
      timeout: 3s
      retries: 5

  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]

volumes:
  pgdata:
```

```bash
docker compose up -d          # start all services
docker compose logs -f app    # follow app logs
docker compose down -v        # stop + remove volumes
```

---

## Debugging Containers

```bash
# Shell into running container
docker exec -it <container> sh

# Shell into a distroless container (no shell!)
docker run --rm -it --entrypoint="" <image> /busybox sh  # won't work
# Instead: use debug image or multi-stage with debug target

# View logs
docker logs <container> --tail 100 -f

# Inspect networking
docker inspect <container> | jq '.[0].NetworkSettings'

# Resource usage
docker stats

# Copy files out
docker cp <container>:/app/dump.json ./dump.json

# Build with specific target (debug vs production)
docker build --target=builder -t myapp:debug .
```

### Debug Stage Pattern

```dockerfile
# Production
FROM gcr.io/distroless/static AS production
COPY --from=builder /app/server /server
ENTRYPOINT ["/server"]

# Debug (has shell, curl, etc.)
FROM alpine:3.19 AS debug
RUN apk add --no-cache curl jq
COPY --from=builder /app/server /server
ENTRYPOINT ["/server"]
```

```bash
docker build --target=debug -t myapp:debug .
docker run -it myapp:debug sh  # now you have a shell
```

---

## Image Size Optimization

| Base Image | Size | Shell? | Use Case |
|-----------|------|--------|----------|
| `golang:1.23` | ~800MB | Yes | Build stage only |
| `alpine:3.19` | ~7MB | Yes | Need shell/tools |
| `distroless/static` | ~2MB | No | Production Go (CGO_ENABLED=0) |
| `scratch` | 0MB | No | Absolute minimum |

```bash
# Check image size
docker images myapp
# REPOSITORY   TAG       SIZE
# myapp        latest    18MB  (distroless + Go binary)
```

---

## Multi-Platform Builds

```mermaid
graph LR
    SRC[Source] --> BUILDX[docker buildx]
    BUILDX --> AMD[linux/amd64<br>EC2, GCE]
    BUILDX --> ARM[linux/arm64<br>Graviton, M-series Mac]
    AMD --> REG[Registry<br>multi-arch manifest]
    ARM --> REG
```

```bash
# Build for linux/amd64 + linux/arm64
docker buildx build --platform linux/amd64,linux/arm64 \
    -t ghcr.io/org/myapp:latest --push .
```
