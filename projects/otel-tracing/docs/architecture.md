# Architecture

## Overview

Two HTTP services propagating distributed traces via OpenTelemetry, exporting spans to Jaeger.

## Components

```
service-a/main.go      → HTTP service on :8080, calls Service B
service-b/main.go      → HTTP service on :8081, downstream
docker-compose.yml     → Jaeger all-in-one for trace visualization
```

## Trace Flow

1. Client calls Service A
2. Service A creates a span, propagates trace context via W3C `traceparent` header
3. Service B receives context, creates child span
4. Both services export spans to Jaeger via OTLP
5. Jaeger UI shows the full request journey

## Key Concepts

- **Trace**: End-to-end journey of a request across services
- **Span**: A single unit of work within a trace
- **Context propagation**: W3C TraceContext headers carry trace_id across HTTP boundaries
- **OTLP**: OpenTelemetry Protocol for exporting telemetry data
