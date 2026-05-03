# 01-tcp-server: Deep Dive

## How TCP Works

TCP (Transmission Control Protocol) is a connection-oriented protocol. Before any data flows, a 3-way handshake establishes the connection:

```mermaid
sequenceDiagram
    participant C as Client
    participant S as Server

    C->>S: SYN (I want to connect)
    S-->>C: SYN-ACK (OK, I'm ready)
    C->>S: ACK (Great, let's go)
    Note over C,S: Connection established
    C->>S: Data bytes
    S-->>C: Echo bytes back
    C->>S: FIN (I'm done)
    S-->>C: FIN-ACK
```

## Accept Loop

The server's core is an infinite loop that blocks on `ln.Accept()` until a client connects, then hands the connection to a goroutine:

```mermaid
graph TD
    BIND[net.Listen\nbind :9000] --> LOOP[Accept loop]
    LOOP -->|blocks until client connects| ACCEPT[ln.Accept\nreturns net.Conn]
    ACCEPT --> GOROUTINE[go handle conn\nnew goroutine per connection]
    GOROUTINE --> LOOP
    GOROUTINE --> ECHO[io.Copy conn→conn\necho all bytes]
    ECHO --> CLOSE[conn.Close\nwhen client disconnects]
```

## Goroutine-per-Connection Model

Go goroutines start with a 2KB stack (vs 1MB for OS threads), making it practical to spawn one per connection:

```mermaid
graph LR
    LN[Listener] -->|Accept| G1[goroutine 1\nclient A]
    LN --> G2[goroutine 2\nclient B]
    LN --> G3[goroutine 3\nclient C]
    LN --> GN[goroutine N\nclient N]
    G1 & G2 & G3 & GN -->|scheduled by| GMP[Go runtime\nG-M-P scheduler]
    GMP -->|OS threads| CPU[CPU cores]
```

Each goroutine blocks on `io.Copy` (a read syscall) while waiting for data. The Go runtime parks the goroutine and runs others — no OS thread is wasted.

## io.Copy Internals

`io.Copy(dst, src)` reads from `src` into a 32KB buffer and writes to `dst` in a loop:

```mermaid
graph LR
    SRC[net.Conn\nread side] -->|read up to 32KB| BUF[buffer]
    BUF -->|write| DST[net.Conn\nwrite side]
    DST -->|loop until EOF| SRC
```

On Linux, when both `src` and `dst` are TCP sockets, the kernel can use `splice(2)` to transfer data between file descriptors without copying through userspace — zero-copy.

## Graceful Close

When the client calls `CloseWrite()`, it sends a TCP FIN. The server's `io.Copy` returns `io.EOF`, and the server closes its side:

```mermaid
sequenceDiagram
    participant C as Client
    participant S as Server

    C->>S: CloseWrite() → FIN
    Note over S: io.Copy returns EOF
    S->>S: conn.Close()
    S-->>C: FIN-ACK
```
