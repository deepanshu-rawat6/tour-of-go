# 02-http-server

HTTP/1.1 built on top of a raw TCP connection — then the same routes with `net/http` for comparison.

## Architecture

```mermaid
graph TD
    C[Client<br>curl / browser] -->|TCP connect| LN[net.Listener]
    LN --> CONN[net.Conn]
    CONN --> PARSE[Parse request line<br>GET /path HTTP/1.1<br>+ headers]
    PARSE --> ROUTE{router<br>map path→handler}
    ROUTE -->|match| H[HandlerFunc]
    ROUTE -->|no match| 404[404 Not Found]
    H --> RW[ResponseWriter<br>write status + headers + body]
    RW --> CONN
```

## Key Concepts

- **Request line**: `METHOD /path HTTP/version\r<br>`
- **Headers**: `Key: Value\r<br>` until blank line `\r<br>`
- **Response**: status line + headers + blank line + body
- **Comparison**: `internal/raw` vs `internal/stdlib` — same routes, different implementation

## Quick Start

```bash
make run-raw     # raw HTTP/1.1 on :8080
make run-stdlib  # net/http on :8081
curl http://localhost:8080/health
curl http://localhost:8081/health
```

## Docs

- [`docs/deep-dive.md`](./docs/deep-dive.md)
