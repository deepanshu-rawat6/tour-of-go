// slog_example.go — Structured logging with log/slog (Go 1.21+).
//
// slog is the official structured logging package added in Go 1.21.
// It replaces ad-hoc use of log, fmt.Printf, and third-party loggers for
// most production use cases.
//
// Key concepts:
//   - Logger: the entry point (wraps a Handler)
//   - Handler: the backend (JSON, text, custom)
//   - Attr: a key-value pair (slog.String, slog.Int, slog.Any, ...)
//   - Level: Debug < Info < Warn < Error
//   - Context: carry trace IDs, request metadata through call chains
//
// Performance notes (nanoseconds per log call, roughly):
//   slog (text):    ~200 ns/op   — stdlib, zero alloc at Info level
//   slog (JSON):    ~300 ns/op   — stdlib, zero alloc at Info level
//   zerolog:        ~150 ns/op   — fastest, builder pattern
//   zap (sugar):    ~250 ns/op   — fast, ergonomic
//   zap (core):     ~100 ns/op   — fastest ergonomic option
//   logrus:         ~900 ns/op   — slowest, reflection-heavy
//
// For 99% of services, slog is fast enough. Switch to zap/zerolog only
// if profiling shows logging in your hot path.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
)

// ─────────────────────────────────────────────────────────────────────────────
// CUSTOM HANDLER — demonstrates the slog.Handler interface
// ─────────────────────────────────────────────────────────────────────────────

// PrefixHandler is a minimal custom slog.Handler that prepends a service
// name to every log line. Real custom handlers are used for:
//   - Routing logs to different destinations (Kafka, Datadog, etc.)
//   - Filtering or redacting sensitive fields (passwords, tokens)
//   - Injecting mandatory fields (service name, version, pod name)
type PrefixHandler struct {
	inner  slog.Handler // delegate to the real handler
	prefix string
}

// Enabled reports whether the handler handles records at the given level.
// This is called before Handle to avoid building expensive Attr values.
func (h *PrefixHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

// Handle formats and emits the log record.
// It injects the service prefix as an attribute before delegating.
func (h *PrefixHandler) Handle(ctx context.Context, r slog.Record) error {
	// Inject service prefix as the first attribute
	r.AddAttrs(slog.String("service", h.prefix))
	return h.inner.Handle(ctx, r)
}

// WithAttrs returns a new handler with additional persistent attributes.
// These attributes appear in every subsequent log record.
func (h *PrefixHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &PrefixHandler{inner: h.inner.WithAttrs(attrs), prefix: h.prefix}
}

// WithGroup returns a new handler that namespaces subsequent attributes
// under the given group name (e.g., "request" → request.method, request.path).
func (h *PrefixHandler) WithGroup(name string) slog.Handler {
	return &PrefixHandler{inner: h.inner.WithGroup(name), prefix: h.prefix}
}

// ─────────────────────────────────────────────────────────────────────────────
// CONTEXT KEY — for carrying trace IDs
// ─────────────────────────────────────────────────────────────────────────────

// contextKey is an unexported type to avoid key collisions in context.
// Always use a private type for context keys — never a string or int directly.
type contextKey string

const traceIDKey contextKey = "trace_id"

// withTraceID stores a trace ID in the context.
func withTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// traceIDFromCtx retrieves the trace ID from context, or "" if absent.
func traceIDFromCtx(ctx context.Context) string {
	if v, ok := ctx.Value(traceIDKey).(string); ok {
		return v
	}
	return ""
}

// ─────────────────────────────────────────────────────────────────────────────
// SLOG EXAMPLE — main demo function
// ─────────────────────────────────────────────────────────────────────────────

// SlogExample demonstrates all major slog features.
// Call this from main() or another entry point.
func SlogExample() {
	fmt.Println("=== log/slog (Go 1.21+) ===")
	fmt.Println()

	// ─────────────────────────────────────────────────────────────────────────
	// 1. slog.Default() — the global logger.
	//    Out of the box it writes human-readable text to stderr at Info level.
	//    Call slog.SetDefault(logger) to swap it application-wide.
	// ─────────────────────────────────────────────────────────────────────────
	fmt.Println("--- 1. Default logger (slog.Info/Warn/Error/Debug) ---")

	// Basic structured calls: pass alternating key, value pairs after the message.
	slog.Info("server starting", "addr", ":8080", "version", "1.0.0")
	slog.Warn("high memory usage", "used_mb", 800, "limit_mb", 1024)
	slog.Error("database error", "query", "SELECT 1", "err", "connection refused")

	// Debug is suppressed by default (level >= Info).
	// You'd see this only if you set the level to Debug (see level demo below).
	slog.Debug("this is hidden by default") // NOT printed

	// ─────────────────────────────────────────────────────────────────────────
	// 2. JSON handler — structured JSON output for log aggregation systems
	//    (Datadog, Loki, CloudWatch, ELK stack).
	//    Every field is machine-parseable.
	// ─────────────────────────────────────────────────────────────────────────
	fmt.Println("\n--- 2. JSON handler ---")

	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		// ReplaceAttr lets you rename or remove attributes.
		// Common use: rename "msg" → "message", remove "time" for tests.
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Rename the built-in "time" key to "timestamp"
			if a.Key == slog.TimeKey {
				a.Key = "timestamp"
			}
			return a
		},
	})
	jsonLogger := slog.New(jsonHandler)
	jsonLogger.Info("user logged in", "user_id", 42, "ip", "192.168.1.1")
	jsonLogger.Warn("rate limit approached", "requests", 950, "limit", 1000)

	// ─────────────────────────────────────────────────────────────────────────
	// 3. Text handler — human-readable output for development / tailing logs.
	//    Key=value format that's still grep-friendly.
	// ─────────────────────────────────────────────────────────────────────────
	fmt.Println("\n--- 3. Text handler ---")

	textHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug, // show Debug and above
	})
	textLogger := slog.New(textHandler)
	textLogger.Debug("cache miss", "key", "user:42")
	textLogger.Info("cache hit",  "key", "user:99", "ttl_s", 300)

	// ─────────────────────────────────────────────────────────────────────────
	// 4. slog.With — create a logger with PERSISTENT attributes.
	//    Every message from this logger automatically includes these fields.
	//    Use for request-scoped loggers: inject request_id, user_id once.
	// ─────────────────────────────────────────────────────────────────────────
	fmt.Println("\n--- 4. slog.With (persistent attributes) ---")

	// Base logger with service info (injected once, appears in all records)
	baseLogger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With(
		"service", "payment-api",
		"version", "2.3.1",
	)

	// Per-request logger: add request_id and user_id for this request's lifetime
	reqLogger := baseLogger.With(
		"request_id", "req-abc-123",
		"user_id", 7890,
	)

	// Both request_id and user_id appear automatically in every call below
	reqLogger.Info("processing payment", "amount_cents", 4999)
	reqLogger.Info("payment succeeded",  "transaction_id", "txn-xyz-789")
	reqLogger.Error("card declined",     "reason", "insufficient_funds")

	// ─────────────────────────────────────────────────────────────────────────
	// 5. slog.Group — nest related attributes under a common prefix.
	//    Produces: {"request":{"method":"GET","path":"/users"}} in JSON.
	//    Produces: request.method=GET request.path=/users in text.
	// ─────────────────────────────────────────────────────────────────────────
	fmt.Println("\n--- 5. slog.Group (structured nesting) ---")

	groupLogger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	groupLogger.Info("incoming request",
		slog.Group("request",
			slog.String("method", "POST"),
			slog.String("path", "/api/v1/users"),
			slog.Int("content_length", 256),
		),
		slog.Group("response",
			slog.Int("status", 201),
			slog.Int("bytes", 128),
		),
	)

	// ─────────────────────────────────────────────────────────────────────────
	// 6. Context-aware logging — slog.InfoContext / slog.ErrorContext.
	//    Pass the context carrying trace IDs, so handlers can extract them.
	//    The standard slog handlers don't extract from context by default —
	//    you need a custom handler or a middleware that calls logger.With().
	// ─────────────────────────────────────────────────────────────────────────
	fmt.Println("\n--- 6. Context-aware logging ---")

	ctx := context.Background()
	ctx = withTraceID(ctx, "trace-00abc123def456")

	// Standard InfoContext: passes ctx to Handler.Handle(ctx, record).
	// A custom handler can extract trace_id from ctx inside Handle().
	slog.InfoContext(ctx, "processing request in context")

	// Pattern: extract from context in a helper and inject as attr
	traceLogger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if tid := traceIDFromCtx(ctx); tid != "" {
		traceLogger = traceLogger.With("trace_id", tid)
	}
	traceLogger.InfoContext(ctx, "user action", "action", "view_profile", "target_user", 123)

	// ─────────────────────────────────────────────────────────────────────────
	// 7. Dynamic log level with slog.LevelVar.
	//    LevelVar allows you to change the log level at runtime (e.g., via
	//    an HTTP endpoint /__log_level) without restarting the service.
	// ─────────────────────────────────────────────────────────────────────────
	fmt.Println("\n--- 7. Dynamic level with slog.LevelVar ---")

	var logLevel slog.LevelVar         // default: Info
	logLevel.Set(slog.LevelInfo)       // explicit (same as default)

	dynamicHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: &logLevel, // pointer — reads current level on each record
	})
	dynLogger := slog.New(dynamicHandler)

	dynLogger.Debug("not shown — level is Info") // suppressed
	dynLogger.Info("level is Info — this shows")

	// Simulate receiving a signal to enable debug logging
	logLevel.Set(slog.LevelDebug)
	dynLogger.Debug("now debug is enabled — this shows")
	dynLogger.Info("still shows at Info too")

	// Reset to Info (e.g., after a debug window)
	logLevel.Set(slog.LevelInfo)

	// ─────────────────────────────────────────────────────────────────────────
	// 8. Custom handler — PrefixHandler wrapping a JSON handler.
	// ─────────────────────────────────────────────────────────────────────────
	fmt.Println("\n--- 8. Custom handler (PrefixHandler) ---")

	inner := slog.NewJSONHandler(os.Stdout, nil)
	customLogger := slog.New(&PrefixHandler{inner: inner, prefix: "order-service"})
	customLogger.Info("order created", "order_id", "ord-999", "total_cents", 5000)
	// Output will include "service":"order-service" on every record.

	// ─────────────────────────────────────────────────────────────────────────
	// 9. slog.SetDefault — replace the global logger.
	//    After this call, slog.Info/Warn/Error/Debug use the new logger.
	//    Useful for setting up production logging once at startup in main().
	// ─────────────────────────────────────────────────────────────────────────
	fmt.Println("\n--- 9. slog.SetDefault (replace global logger) ---")

	productionHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	productionLogger := slog.New(productionHandler).With(
		"app", "my-service",
		"env", "production",
	)
	slog.SetDefault(productionLogger)

	// Now slog.Info etc. use the new logger with "app" and "env" always present
	slog.Info("application started", "pid", os.Getpid())

	// ─────────────────────────────────────────────────────────────────────────
	// 10. Typed Attr helpers — more efficient than passing key+value pairs.
	//     Use slog.String, slog.Int, slog.Bool, slog.Duration, slog.Any
	//     when you care about zero allocations in hot paths.
	// ─────────────────────────────────────────────────────────────────────────
	fmt.Println("\n--- 10. Typed Attr helpers (zero alloc) ---")

	attrLogger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	attrLogger.Info("typed attrs",
		slog.String("name", "Alice"),          // no boxing to interface{}
		slog.Int("age", 30),                   // no boxing
		slog.Bool("active", true),             // no boxing
		slog.Float64("score", 99.5),           // no boxing
		slog.Any("tags", []string{"go", "slog"}), // slog.Any for complex types
	)

	fmt.Println()
	fmt.Println("slog summary:")
	fmt.Println("  slog.Info/Warn/Error/Debug   — global logger")
	fmt.Println("  slog.New(handler)            — create scoped logger")
	fmt.Println("  logger.With(key, val, ...)   — persistent attributes")
	fmt.Println("  slog.Group(name, attrs...)   — nested JSON grouping")
	fmt.Println("  slog.InfoContext(ctx, msg)   — context-aware")
	fmt.Println("  &slog.LevelVar              — dynamic level changes")
	fmt.Println("  slog.SetDefault(logger)      — replace global at startup")
}
