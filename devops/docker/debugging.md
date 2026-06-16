# Docker Debugging Scenarios

---

## 1. Container Exits Immediately

**Symptom:** `docker run` exits instantly, `docker ps` shows no running container.

```mermaid
flowchart TD
    A[docker run exits] --> B[docker inspect ExitCode]
    B --> C{Exit code?}
    C -->|0| D[CMD completed normally]
    C -->|1| E[App error]
    C -->|137| F[OOM / SIGKILL]
    C -->|139| G[Segfault]
    E --> H[docker logs]
    F --> H
    G --> H
    H --> I{Logs helpful?}
    I -->|No| J[Override entrypoint:<br/>docker run -it --entrypoint sh]
    I -->|Yes| K[Fix app error]
```

```bash
# Check exit code and error
docker inspect <container> --format='{{.State.ExitCode}} | {{.State.Error}}'

# View logs of exited container
docker logs <container>

# Override entrypoint to debug interactively
docker run -it --entrypoint sh <image>
```

---

## 2. Container OOM Killed

**Symptom:** Container exits with code 137. App was consuming too much memory.

```mermaid
flowchart TD
    A[Exit code 137] --> B[docker inspect OOMKilled]
    B --> C{OOMKilled=true?}
    C -->|Yes| D[docker stats — live memory]
    C -->|No| E[Killed by external signal]
    D --> F{Over limit?}
    F -->|Yes| G[Raise --memory limit]
    F -->|No| H[Check memory leak in app]
    G --> I[docker run --memory 512m<br/>--memory-swap 512m]
```

```bash
# Check OOMKilled flag
docker inspect <container> --format='{{.State.OOMKilled}}'

# Live memory stats
docker stats <container>

# Set memory limit
docker run --memory 512m --memory-swap 512m <image>
```

---

## 3. Can't Connect to Container Port

**Symptom:** `curl localhost:8080` fails even though container runs with `-p 8080:80`.

```mermaid
flowchart TD
    A[curl localhost:8080 fails] --> B{Container running?}
    B -->|No| C[docker start / docker run]
    B -->|Yes| D{Port published?}
    D -->|No| E[Add -p 8080:80 flag]
    D -->|Yes| F{App listening inside?}
    F -->|No| G[Fix app bind address]
    F -->|Yes| H{App on 0.0.0.0?}
    H -->|No: 127.0.0.1| I[Bind app to 0.0.0.0]
    H -->|Yes| J{iptables rule exists?}
    J -->|No| K[Restart Docker daemon]
    J -->|Yes| L{Host firewall blocking?}
    L -->|Yes| M[Open host port]
    L -->|No| N[Check docker inspect Ports]
```

```bash
# Check published ports
docker ps --format 'table {{.Names}}\t{{.Ports}}'

# Check what the app is listening on inside the container
docker exec <container> ss -tlnp

# Inspect NAT rules
iptables -t nat -L DOCKER -n

# Inspect port bindings
docker inspect <container> --format='{{json .NetworkSettings.Ports}}'
```

---

## 4. Container Can't Reach the Internet

**Symptom:** `curl` inside container times out. DNS resolves fine.

```mermaid
flowchart TD
    A[No internet inside container] --> B[ping 8.8.8.8]
    B --> C{Ping works?}
    C -->|Yes| D[DNS issue only — check resolv.conf]
    C -->|No| E{ip_forward enabled?}
    E -->|No| F[sysctl -w net.ipv4.ip_forward=1]
    E -->|Yes| G{MASQUERADE rule exists?}
    G -->|No| H[iptables -t nat -A POSTROUTING<br/>-j MASQUERADE]
    G -->|Yes| I{docker0 bridge OK?}
    I -->|No| J[docker network inspect bridge]
    I -->|Yes| K[Custom network routing issue]
```

```bash
# Ping from inside container
docker exec <container> ping -c3 8.8.8.8

# Check IP forwarding
sysctl net.ipv4.ip_forward

# Check MASQUERADE rule
iptables -t nat -L POSTROUTING -n -v

# Inspect docker network
docker network inspect bridge
```

---

## 5. Containers on Same Host Can't Reach Each Other

**Symptom:** Container A can't ping/connect container B on the same host.

```mermaid
flowchart TD
    A[A can't reach B] --> B{Same network?}
    B -->|No| C[Connect both to same<br/>custom network]
    B -->|Yes| D{Default or custom bridge?}
    D -->|Default bridge| E[No DNS — use IP address]
    D -->|Custom bridge| F[DNS works — use container name]
    E --> G[docker inspect for IP]
    F --> H[ping container-name]
    H --> I{Still fails?}
    I -->|Yes| J[Check DOCKER-ISOLATION<br/>iptables chain]
    J --> K[docker network inspect]
```

```bash
# Inspect network and connected containers
docker network inspect <network>

# Ping by container name (custom network only)
docker exec <containerA> ping <containerB>

# DNS lookup inside container
docker exec <containerA> nslookup <containerB>

# Check container's network config
docker inspect <container> --format='{{json .NetworkSettings.Networks}}'
```

---

## 6. docker build Is Slow / Cache Not Working

**Symptom:** Every build takes full time even with no changes.

```mermaid
flowchart TD
    A[Cache always misses] --> B{COPY . . before RUN?}
    B -->|Yes| C[Move COPY after RUN deps —<br/>cache invalidated on any file change]
    B -->|No| D{.dockerignore missing?}
    D -->|Yes| E[Add .dockerignore —<br/>exclude node_modules / .git]
    D -->|No| F{Build args changing?}
    F -->|Yes| G[ARG invalidates all layers below it]
    F -->|No| H{--no-cache flag?}
    H -->|Yes| I[Remove --no-cache from CI]
    H -->|No| J[Correct layer order:<br/>deps → code → COPY .]
```

**Correct Dockerfile layer order:**
```dockerfile
# Good: stable layers first
COPY go.mod go.sum ./
RUN go mod download          # cached unless go.mod changes
COPY . .                     # only invalidates layers below
RUN go build -o app .
```

---

## 7. Image Too Large

**Symptom:** `docker images` shows 1.5 GB image. CI pulls take minutes.

```mermaid
flowchart TD
    A[Image too large] --> B[docker history — find fat layer]
    B --> C{Base image bloated?}
    C -->|Yes| D[Switch to alpine / distroless]
    C -->|No| E{apt cache not cleaned?}
    E -->|Yes| F[Add rm -rf /var/lib/apt/lists/*<br/>in same RUN]
    E -->|No| G{Single stage build?}
    G -->|Yes| H[Use multi-stage build]
    G -->|No| I{Source files copied?}
    I -->|Yes| J[Add .dockerignore]
```

```bash
# Show layer sizes
docker history --no-trunc <image>

# Interactive layer explorer (install separately)
dive <image>

# List image sizes
docker images --format 'table {{.Repository}}\t{{.Size}}'
```

**Multi-stage example:**
```dockerfile
FROM golang:1.22 AS builder
WORKDIR /app
COPY . .
RUN go build -o app .

FROM gcr.io/distroless/static
COPY --from=builder /app/app /app
ENTRYPOINT ["/app"]
```

---

## 8. Volume Mount Not Working

**Symptom:** App gets permission denied or file not found on mounted path.

```mermaid
flowchart TD
    A[Volume mount fails] --> B{Bind or named volume?}
    B -->|Bind mount| C{Host path exists?}
    C -->|No| D[mkdir on host path]
    C -->|Yes| E{UID mismatch?}
    E -->|Yes| F[chown on host OR<br/>--user flag in docker run]
    E -->|No| G{SELinux / AppArmor?}
    G -->|Yes| H[Add :z or :Z to mount]
    B -->|Named volume| I[docker volume inspect]
    I --> J{Volume has data?}
    J -->|No| K[Check Dockerfile VOLUME directive]
```

```bash
# Inspect mounts on running container
docker inspect <container> --format='{{json .Mounts}}'

# Check host path permissions
ls -la /host/path

# Mount with read-only
docker run -v /host/path:/container/path:ro <image>

# List and inspect named volumes
docker volume ls
docker volume inspect <volume>
```

---

## 9. docker-compose Service Won't Start

**Symptom:** `docker compose up` — dependency shows healthy but dependent fails immediately.

```mermaid
flowchart TD
    A[Service fails on start] --> B{depends_on condition met?}
    B -->|No| C[Check healthcheck command —<br/>test / interval / retries]
    B -->|Yes| D{Env vars correct?}
    D -->|No| E[docker compose config —<br/>validate resolved values]
    D -->|Yes| F{Port conflict on host?}
    F -->|Yes| G[Change host port in compose]
    F -->|No| H[docker compose logs service]
    H --> I[Fix app startup error]
```

```bash
# View service logs
docker compose logs <service>

# Show all service states
docker compose ps

# Validate and print resolved compose config
docker compose config

# Recreate containers (pick up config changes)
docker compose up --force-recreate
```

---

## 10. Disk Space Exhausted by Docker

**Symptom:** `/var/lib/docker` consuming 50 GB+. Host disk full.

```mermaid
flowchart TD
    A[Disk full] --> B[docker system df]
    B --> C{Stopped containers?}
    C -->|Yes| D[docker container prune]
    C -->|No| E{Dangling images?}
    E -->|Yes| F[docker image prune]
    E -->|No| G{Unused volumes?}
    G -->|Yes| H[docker volume prune]
    G -->|No| I{Build cache large?}
    I -->|Yes| J[docker builder prune]
    I -->|No| K[docker system prune -af --volumes]
```

```bash
# Show disk usage breakdown
docker system df

# Prune everything unused (containers, images, networks, cache)
docker system prune -af --volumes

# Targeted pruning
docker image prune -f          # dangling images only
docker image prune -af         # all unused images
docker container prune -f      # stopped containers
docker volume prune -f         # unused volumes
docker builder prune -af       # build cache
```
