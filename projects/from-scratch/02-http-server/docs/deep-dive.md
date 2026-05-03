# 02-http-server: Deep Dive

## HTTP/1.1 Wire Format

HTTP is a text protocol layered on top of TCP. Every request and response follows a strict format:

```
GET /path HTTP/1.1\r\n
Host: localhost\r\n
Content-Type: text/plain\r\n
\r\n
<optional body>
```

```mermaid
graph TD
    TCP[TCP byte stream] --> RL[Request Line\nMETHOD /path HTTP/version]
    RL --> HEADERS[Headers\nKey: Value pairs\nuntil blank line]
    HEADERS --> BLANK[Blank line\r\n]
    BLANK --> BODY[Body\noptional]
```

## Raw Parser Flow

```mermaid
graph TD
    CONN[net.Conn] --> BR[bufio.Reader\nbuffered reads]
    BR -->|ReadString\n| RL[Parse request line\nstrings.Fields]
    RL --> LOOP[Header loop\nReadString until blank]
    LOOP --> ROUTE{router\nmap path→handler}
    ROUTE -->|found| HANDLER[HandlerFunc\nw ResponseWriter, r Request]
    ROUTE -->|not found| 404[write 404]
    HANDLER --> RW[ResponseWriter.Write\nstatus + headers + body]
    RW --> CONN
```

## Raw vs stdlib Comparison

```mermaid
graph LR
    subgraph Raw
        R1[net.Listen] --> R2[bufio.Reader\nmanual parse]
        R2 --> R3[map path→HandlerFunc]
        R3 --> R4[fmt.Fprintf\nwrite response]
    end

    subgraph Stdlib
        S1[http.ListenAndServe] --> S2[net/http\nauto parse]
        S2 --> S3[http.ServeMux\nrouter]
        S3 --> S4[http.ResponseWriter\nwrite response]
    end
```

The raw implementation teaches what `net/http` does internally. The stdlib version is what you'd use in production.

## Response Format

```mermaid
graph TD
    STATUS[Status line\nHTTP/1.1 200 OK\r\n] --> HDRS[Response headers\nContent-Type: ...\r\nContent-Length: ...\r\n]
    HDRS --> BLANK[Blank line\r\n]
    BLANK --> BODY[Response body]
```

## Keep-Alive vs Connection-per-Request

Our raw implementation closes the connection after each request (HTTP/1.0 style). Real HTTP/1.1 uses persistent connections:

```mermaid
sequenceDiagram
    participant C as Client
    participant S as Server

    Note over C,S: HTTP/1.0 (our raw impl)
    C->>S: GET /
    S-->>C: 200 OK
    S->>S: close connection

    Note over C,S: HTTP/1.1 (persistent)
    C->>S: GET /
    S-->>C: 200 OK
    C->>S: GET /health
    S-->>C: 200 OK
    Note over C,S: connection reused
```
