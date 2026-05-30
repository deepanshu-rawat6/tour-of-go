package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
)

type ctxKey string

const traceIDKey ctxKey = "trace_id"

func generateTraceID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func TraceIDFrom(ctx context.Context) string {
	if id, ok := ctx.Value(traceIDKey).(string); ok {
		return id
	}
	return "unknown"
}

func LoggerFrom(ctx context.Context) *slog.Logger {
	return slog.Default().With("trace_id", TraceIDFrom(ctx))
}

// TraceMiddleware injects/propagates trace_id.
func TraceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get("X-Trace-ID")
		if traceID == "" {
			traceID = generateTraceID()
		}
		ctx := context.WithValue(r.Context(), traceIDKey, traceID)
		w.Header().Set("X-Trace-ID", traceID)

		log := slog.Default().With("trace_id", traceID, "method", r.Method, "path", r.URL.Path)
		start := time.Now()
		log.Info("request started")

		next.ServeHTTP(w, r.WithContext(ctx))

		log.Info("request completed", "duration_ms", time.Since(start).Milliseconds())
	})
}

func handleOrder(w http.ResponseWriter, r *http.Request) {
	log := LoggerFrom(r.Context())
	orderID := r.URL.Path[len("/orders/"):]

	log.Info("processing order", "order_id", orderID)

	// Simulate downstream call
	time.Sleep(50 * time.Millisecond)
	log.Info("order validated", "order_id", orderID)

	fmt.Fprintf(w, `{"order_id": %q, "status": "processed"}`, orderID)
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprint(w, `{"status": "ok"}`)
}

func main() {
	// Production JSON logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	mux := http.NewServeMux()
	mux.HandleFunc("/orders/", handleOrder)
	mux.HandleFunc("/health", handleHealth)

	handler := TraceMiddleware(mux)

	slog.Info("server starting", "addr", ":8080")
	fmt.Println("\nTry: curl -H 'X-Trace-ID: my-trace-123' http://localhost:8080/orders/42")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		slog.Error("server failed", "err", err)
	}
}
