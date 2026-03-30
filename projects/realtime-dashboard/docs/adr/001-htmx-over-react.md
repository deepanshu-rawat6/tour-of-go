# ADR-001: HTMX over React/Vue for the Dashboard

**Status:** Accepted

## Decision

Use HTMX for interactivity instead of a JavaScript framework.

## Rationale

| Concern | React/Vue | HTMX + html/template |
|---|---|---|
| Build step | npm, webpack/vite, node_modules | None — CDN script tag |
| Language boundary | Go backend + JS frontend | Go end-to-end |
| State management | Client-side state (Redux, Pinia) | Server is the source of truth |
| Deployment | Separate frontend build artifact | Single Go binary serves everything |
| Learning focus | JS ecosystem | Go html/template, WebSocket, SSE |

HTMX's model — "HTML over the wire" — is a natural fit for Go. The server renders HTML fragments and HTMX swaps them into the DOM. No JSON API needed for UI updates.

## WebSocket for Real-Time

HTMX polling (`hx-trigger="every 2s"`) handles the job table refresh. WebSocket handles instant concurrency pool updates pushed from the server — no polling latency for the most time-sensitive data.

## Consequences

- No npm, no node_modules, no build pipeline
- Dashboard works without JavaScript for basic functionality (progressive enhancement)
- Go developer can own the entire stack
