# OpenTelemetry Distributed Tracing

Two HTTP services instrumented with OpenTelemetry. A request to `service-a` triggers a call to `service-b`, and both emit spans with the **same trace ID** — showing how a single request is tracked across service boundaries.

## Concepts

- **Span**: A single unit of work (one HTTP handler, one DB call). Has a start time, end time, and attributes.
- **Trace**: A tree of spans sharing one Trace ID — the full journey of a request.
- **Context Propagation**: The Trace ID travels via HTTP headers (`traceparent`) from service-a to service-b.
- **Exporter**: Where spans are sent. This demo uses stdout; swap for Jaeger/Honeycomb in production.

## How to Run (stdout mode)

```shell
# Terminal 1 — start service-b first
go run ./service-b/

# Terminal 2 — start service-a
go run ./service-a/

# Terminal 3 — send a request
curl "http://localhost:8080/hello?name=Gopher"
```

Watch both terminals — you'll see spans printed with matching `TraceID` values.

## How to Run (with Jaeger UI)

```shell
docker-compose up -d jaeger
# Then run service-a and service-b as above
# Open http://localhost:16686 to see the trace waterfall
```

## Key Code Patterns

**Starting a span** (service-a):
```go
ctx, span := tracer.Start(r.Context(), "service-a.hello")
defer span.End()
```

**Injecting trace context into outgoing HTTP** (service-a → service-b):
```go
otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
```

**Extracting trace context from incoming HTTP** (service-b):
```go
ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
```

## What to Learn Next

- Add span attributes: `span.SetAttributes(attribute.String("user.id", userID))`
- Add span events: `span.AddEvent("cache miss")`
- Instrument a database call as a child span
- See [Distributed Tracing README](../../more-internals/system-design/tracing/README.md) for theory
