package main

import (
	"context"
	"fmt"
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
	// W3C TraceContext propagation — this is how trace IDs cross service boundaries
	otel.SetTextMapPropagator(propagation.TraceContext{})
	return func() { tp.Shutdown(context.Background()) }
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	shutdown := initTracer("service-b")
	defer shutdown()

	tracer := otel.Tracer("service-b")

	http.HandleFunc("/greet", func(w http.ResponseWriter, r *http.Request) {
		// Extract the trace context from incoming HTTP headers
		// This continues the trace started by service-a
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

		_, span := tracer.Start(ctx, "service-b.greet")
		defer span.End()

		name := r.URL.Query().Get("name")
		if name == "" {
			name = "World"
		}

		log.Printf("[service-b] handling /greet traceID=%s", span.SpanContext().TraceID())
		fmt.Fprintf(w, "Hello from service-b, %s!", name)
	})

	log.Printf("service-b listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
