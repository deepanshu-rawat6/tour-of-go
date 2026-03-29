# Distributed Tracing (OpenTelemetry): The Microservice Map

In a system with dozens of microservices, finding the root cause of a single failed request is like finding a needle in a haystack. Distributed tracing provides the map to navigate this complexity, following a single request as it hops across services, databases, and message queues.

---

## 🏗️ Core Concepts: Spans and Traces

1.  **Span**: Represents a single unit of work (e.g., an HTTP request, a database query, or a function call).
2.  **Trace**: A collection of spans that share a single **Trace ID**, representing the entire journey of a request.
3.  **Trace ID**: A unique identifier propagated across every service call.

---

## 🛰️ Context Propagation

The most critical part of tracing in Go is passing the `context.Context` through every layer of your application.

```go
// propagation in Go
func (s *Service) Process(ctx context.Context, data []byte) error {
    ctx, span := s.tracer.Start(ctx, "process_data")
    defer span.End()

    return s.db.Save(ctx, data)
}
```

If you lose the context, the trace is broken, and you lose visibility.

---

## 🧱 Key Components: OpenTelemetry (OTel)

OpenTelemetry is the industry standard for collecting traces, metrics, and logs. It provides a vendor-neutral SDK for Go.

### 🧩 How it works:
1.  **Tracer SDK**: Instruments your code to create spans.
2.  **Propagator**: Injects and extracts the Trace ID from HTTP headers (e.g., `W3C TraceContext`).
3.  **Exporter**: Sends the collected spans to a backend like Jaeger, Honeycomb, or AWS X-Ray.

---

## 🏎️ Why Tracing?

*   **Bottleneck Identification**: See exactly which service is causing a 2-second delay in a 10-service chain.
*   **Error Root-Cause**: Identify where an error originated, even if it propagated through multiple services.
*   **Dependency Mapping**: Automatically generate a map of how your services interact.

---

## 🚀 Key Benefits for Platform Engineers
*   **Observability**: Measure latency across service boundaries without manual logs.
*   **Cost Efficiency**: Use "Sampling" to only trace 1% of successful requests but 100% of errors.
*   **Developer Experience**: Give developers a visual timeline of their requests.

---

## 🛠️ Tracing Best Practices
1.  **Trace All External Calls**: HTTP clients, database drivers, and message queue producers should all be instrumented.
2.  **Add Attributes**: Attach business metadata to spans (e.g., `customer_id`, `order_type`) for better filtering.
3.  **Log Integration**: Link your traditional logs to your Trace IDs for the ultimate debugging experience.

---

## 🛠️ Popular Tracing Tools
*   **Jaeger**: An open-source, end-to-end distributed tracing system.
*   **Honeycomb**: A modern observability platform for high-cardinality data.
*   **Datadog**: A popular SaaS monitoring platform with deep tracing support.
