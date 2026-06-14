# Zero-Downtime Deployments & Graceful Draining

In a Kubernetes world, your Go service will be killed and restarted constantly (Rolling Updates). If you don't handle this correctly, your users will see `502 Bad Gateway` or `Connection Refused` errors.

## 1. The Kubernetes Lifecycle
When a Pod is deleted:
1. K8s sends a **`SIGTERM`** to your process.
2. It waits for a "Grace Period" (usually 30 seconds).
3. If the process is still running, it sends **`SIGKILL`** (hard kill).

## 2. Platform Pattern: The 5-10 Second "Draining" Period
Even after K8s sends `SIGTERM`, it takes a few seconds for the Network Load Balancer to stop sending new traffic to your Pod. You **must** wait a few seconds before actually shutting down your HTTP server.

### Go Snippet (Production Shutdown)
```go
func main() {
    server := &http.Server{Addr: ":8080"}

    // 1. Listen for OS signals in a channel
    stop := make(chan os.Signal, 1)
    signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

    go func() {
        if err := server.ListenAndServe(); err != http.ErrServerClosed {
            log.Fatalf("HTTP server failed: %v", err)
        }
    }()

    // 2. Block until we receive a SIGTERM or SIGINT
    <-stop
    log.Println("SIGTERM received, starting graceful shutdown...")

    // 3. CRITICAL: Wait 5-10 seconds for K8s service discovery to propagate
    // This prevents 502 errors from new requests coming in.
    time.Sleep(10 * time.Second)

    // 4. Create a context with a timeout for the shutdown
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // 5. Shutdown() gracefully closes all idle connections and waits for active ones.
    if err := server.Shutdown(ctx); err != nil {
        log.Fatalf("Graceful shutdown failed: %v", err)
    }
    log.Println("Server exited cleanly")
}
```

## 3. Real-World Use Case: Google Cloud Run & AWS Fargate
Serverless platforms like **Cloud Run** or **Fargate** use this exact mechanism. When they scale down your containers, they send a `SIGTERM`. If you have a long-running process (like a video transcoder), you must use this pattern to save the current progress to a DB before the container disappears.

## 4. Readiness & Liveness Probes
- **Liveness:** "Am I dead?" (K8s restarts you if false).
- **Readiness:** "Am I ready for traffic?" (K8s stops sending traffic if false).
- **Graceful Shutdown Tip:** As soon as you receive `SIGTERM`, your Readiness probe should start returning `503 Service Unavailable` so the Load Balancer removes you even faster.
