# 02-http-server: Deep Dive

## HTTP/1.1 Wire Format

HTTP is a text protocol layered on top of TCP. Every request and response follows a strict format:

```
GET /path HTTP/1.1\r<br>
Host: localhost\r<br>
Content-Type: text/plain\r<br>
\r<br>
<optional body>
```

```mermaid
graph TD
    TCP[TCP byte stream] --> RL[Request Line<br>METHOD /path HTTP/version]
    RL --> HEADERS[Headers<br>Key: Value pairs<br>until blank line]
    HEADERS --> BLANK[Blank line\r<br>]
    BLANK --> BODY[Body<br>optional]
```

## Raw Parser Flow

```mermaid
graph TD
    CONN[net.Conn] --> BR[bufio.Reader<br>buffered reads]
    BR -->|ReadString<br>| RL[Parse request line<br>strings.Fields]
    RL --> LOOP[Header loop<br>ReadString until blank]
    LOOP --> ROUTE{router<br>map path→handler}
    ROUTE -->|found| HANDLER[HandlerFunc<br>w ResponseWriter, r Request]
    ROUTE -->|not found| 404[write 404]
    HANDLER --> RW[ResponseWriter.Write<br>status + headers + body]
    RW --> CONN
```

## Raw vs stdlib Comparison

```mermaid
graph LR
    subgraph Raw
        R1[net.Listen] --> R2[bufio.Reader<br>manual parse]
        R2 --> R3[map path→HandlerFunc]
        R3 --> R4[fmt.Fprintf<br>write response]
    end

    subgraph Stdlib
        S1[http.ListenAndServe] --> S2[net/http<br>auto parse]
        S2 --> S3[http.ServeMux<br>router]
        S3 --> S4[http.ResponseWriter<br>write response]
    end
```

The raw implementation teaches what `net/http` does internally. The stdlib version is what you'd use in production.

## Response Format

```mermaid
graph TD
    STATUS[Status line<br>HTTP/1.1 200 OK\r<br>] --> HDRS[Response headers<br>Content-Type: ...\r<br>Content-Length: ...\r<br>]
    HDRS --> BLANK[Blank line\r<br>]
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
