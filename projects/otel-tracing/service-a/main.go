package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

func initTracer(serviceName string) func() {
	exporter, _ := stdouttrace.New(stdouttrace.WithPrettyPrint())
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		)),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	return func() { tp.Shutdown(context.Background()) }
}

func callServiceB(ctx context.Context, name string) (string, error) {
	serviceBURL := os.Getenv("SERVICE_B_URL")
	if serviceBURL == "" {
		serviceBURL = "http://localhost:8081"
	}

	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/greet?name=%s", serviceBURL, name), nil)
	if err != nil {
		return "", err
	}

	// Inject the current trace context into outgoing HTTP headers
	// This is how the trace ID propagates to service-b
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return string(body), nil
}

func main() {
	shutdown := initTracer("service-a")
	defer shutdown()

	tracer := otel.Tracer("service-a")

	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		// Start a new root span for this request
		ctx, span := tracer.Start(r.Context(), "service-a.hello")
		defer span.End()

		name := r.URL.Query().Get("name")
		if name == "" {
			name = "Gopher"
		}

		log.Printf("[service-a] handling /hello traceID=%s", span.SpanContext().TraceID())

		// Call service-b — the trace context is propagated via HTTP headers
		msg, err := callServiceB(ctx, name)
		if err != nil {
			http.Error(w, "service-b unavailable: "+err.Error(), http.StatusServiceUnavailable)
			return
		}

		fmt.Fprintf(w, "service-a got from service-b: %s", msg)
	})

	log.Println("service-a listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
