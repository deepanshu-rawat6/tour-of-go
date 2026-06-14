package ctx_examples

import (
	"context"
	"fmt"
)

// Use unexported key types to avoid collisions with other packages
type contextKey string

const (
	requestIDKey contextKey = "requestID"
	userIDKey    contextKey = "userID"
)

func handleRequest(ctx context.Context) {
	reqID, _ := ctx.Value(requestIDKey).(string)
	userID, _ := ctx.Value(userIDKey).(string)
	fmt.Printf("  handling request: reqID=%s userID=%s\n", reqID, userID)
}

func valuesExample() {
	fmt.Println("context.WithValue:")

	// Build up context with request-scoped values
	ctx := context.Background()
	ctx = context.WithValue(ctx, requestIDKey, "req-abc-123")
	ctx = context.WithValue(ctx, userIDKey, "user-42")

	handleRequest(ctx)

	// Values are read-only and scoped — child contexts inherit parent values
	childCtx := context.WithValue(ctx, contextKey("extra"), "bonus")
	v := childCtx.Value(requestIDKey)
	fmt.Println("  child ctx still has requestID:", v)

	fmt.Println("\n  Rule: use context values only for request-scoped data")
	fmt.Println("  Rule: don't use context to pass optional function parameters")
}
