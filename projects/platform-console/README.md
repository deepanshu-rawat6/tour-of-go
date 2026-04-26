# platform-console

A web console for browsing Kubernetes `Greeting` custom resources. Lists CRs via the dynamic client and streams live updates to the browser via Server-Sent Events.

---

## Architecture

```mermaid
graph TD
    Browser -->|GET /| H[Handler\nhtml/template + Tailwind]
    H -->|list Greetings| K[K8s Dynamic Client\nclient-go]
    K -->|unstructured list| H
    H -->|render| PAGE[HTML Page]

    Browser -->|GET /watch SSE| W[K8s Watcher\nclient-go Watch API]
    W -->|watch.Event ADDED/MODIFIED/DELETED| SSE[SSE Stream\ntext/event-stream]
    SSE -->|push HTML fragment| Browser
    Browser -->|HTMX hx-swap| DOM[DOM update]
```

## SSE + HTMX Flow

```mermaid
sequenceDiagram
    participant B as Browser
    participant S as Server
    participant K as Kubernetes API

    B->>S: GET /watch (SSE)
    S->>K: Watch Greetings
    K-->>S: ADDED event
    S-->>B: data: <tr>...</tr>
    B->>B: HTMX swaps row into table
    K-->>S: MODIFIED event
    S-->>B: data: <tr>...</tr>
    B->>B: HTMX updates row
```

## Key Concepts

- **Dynamic client** — uses `client-go`'s dynamic client to list/watch any CRD without generated types. Returns `unstructured.Unstructured`.
- **Server-Sent Events** — one-way push from server to browser. Simpler than WebSocket for read-only live updates.
- **HTMX** — browser receives HTML fragments over SSE and swaps them into the DOM. No JavaScript written.
- **html/template** — renders both the full page and the SSE fragment updates from the same template.

## Quick Start

```bash
# Requires a running K8s cluster with the Greeting CRD installed
# See projects/k8s-controller for CRD setup
make run
# Open http://localhost:8080
```
