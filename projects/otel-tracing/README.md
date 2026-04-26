# OpenTelemetry Distributed Tracing

Two HTTP services instrumented with OpenTelemetry. A request to `service-a` triggers a call to `service-b`, and both emit spans with the **same trace ID** — showing how a single request is tracked across service boundaries.

---

## Architecture

```mermaid
graph LR
    Client -->|GET /hello?name=Gopher| A[Service A\n:8080]
    A -->|HTTP + W3C traceparent header| B[Service B\n:8081]
    B -->|span: service-b.greet| B
    A -->|span: service-a.hello| A
    A -->|OTLP gRPC| J[Jaeger\n:16686]
    B -->|OTLP gRPC| J
```

## Trace Propagation

```mermaid
sequenceDiagram
    participant C as Client
    participant A as Service A
    participant B as Service B
    participant J as Jaeger

    C->>A: GET /hello?name=Gopher
    A->>A: tracer.Start("service-a.hello") → span₁
    A->>B: GET /greet (traceparent: 00-<traceID>-<spanID>-01)
    B->>B: Extract context from header
    B->>B: tracer.Start("service-b.greet") → span₂ (child of span₁)
    B-->>A: 200 OK
    A-->>C: 200 OK
    A->>J: export span₁
    B->>J: export span₂
    Note over J: Both spans share the same TraceID
```

## Concepts

- **Span** — a single unit of work with start time, end time, and attributes
- **Trace** — a tree of spans sharing one Trace ID — the full journey of a request
- **Context Propagation** — Trace ID travels via `traceparent` HTTP header (W3C standard)
- **Exporter** — where spans are sent; this demo uses stdout or Jaeger via OTLP

## How to Run

```shell
# stdout mode (no Docker needed)
go run ./service-b/ &
go run ./service-a/ &
curl "http://localhost:8080/hello?name=Gopher"
# Watch both terminals for matching TraceID values

# With Jaeger UI
docker-compose up -d jaeger
# Open http://localhost:16686
```

## Key Code Patterns

```go
// Start a span
ctx, span := tracer.Start(r.Context(), "service-a.hello")
defer span.End()

// Inject trace context into outgoing HTTP (service-a → service-b)
otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

// Extract trace context from incoming HTTP (service-b)
ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
```
