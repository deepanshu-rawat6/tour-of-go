# Networking Fundamentals

DNS resolution, TCP/TLS handshakes, HTTP/2 multiplexing, and mTLS certificate rotation — what every backend engineer must understand.

---

## DNS Resolution

```mermaid
sequenceDiagram
    participant App as Go App
    participant Res as OS Resolver
    participant Cache as Local Cache
    participant DNS as DNS Server
    participant Auth as Authoritative NS
    
    App->>Res: net.LookupHost("api.example.com")
    Res->>Cache: Check /etc/hosts + cache
    Cache-->>Res: miss
    Res->>DNS: Query A record
    DNS->>Auth: Recursive lookup
    Auth-->>DNS: 93.184.216.34 (TTL=300s)
    DNS-->>Res: 93.184.216.34
    Res-->>App: []string{"93.184.216.34"}
```

**Go DNS behavior:**
- Uses OS resolver by default (`/etc/resolv.conf`)
- `GODEBUG=netdns=go` forces pure-Go resolver (no CGO)
- DNS caching: Go does NOT cache DNS — each `Dial` triggers a lookup
- For high-throughput: use a custom `net.Resolver` or connection pooling

```go
resolver := &net.Resolver{
    PreferGo: true, // pure-Go resolver, no CGO
    Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
        d := net.Dialer{Timeout: 5 * time.Second}
        return d.DialContext(ctx, "udp", "8.8.8.8:53") // custom DNS
    },
}
```

---

## TCP Handshake

```mermaid
sequenceDiagram
    participant C as Client
    participant S as Server
    
    Note over C,S: Three-Way Handshake (~1 RTT)
    C->>S: SYN (seq=x)
    S->>C: SYN-ACK (seq=y, ack=x+1)
    C->>S: ACK (ack=y+1)
    Note over C,S: Connection established
    C->>S: Data...
```

**Latency**: 1 RTT before any data flows. For cross-region (100ms RTT), that's 100ms just to connect.

---

## TLS Handshake (TLS 1.3)

```mermaid
sequenceDiagram
    participant C as Client
    participant S as Server
    
    Note over C,S: TLS 1.3: 1-RTT handshake (vs 2-RTT in TLS 1.2)
    C->>S: ClientHello + key_share + supported_ciphers
    S->>C: ServerHello + key_share + certificate + Finished
    C->>S: Finished
    Note over C,S: Encrypted data flows (0-RTT resumption possible)
```

**Total connection cost**: TCP (1 RTT) + TLS (1 RTT) = **2 RTTs** minimum before first byte.

```go
// Go TLS config for production
tlsConfig := &tls.Config{
    MinVersion: tls.VersionTLS13,
    CipherSuites: nil, // TLS 1.3 ciphers are not configurable (always secure)
    CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
}
```

---

## HTTP/2 Multiplexing

```mermaid
graph TD
    subgraph HTTP/1.1
        C1[Request 1] --> CONN1[TCP Connection 1]
        C2[Request 2] --> CONN2[TCP Connection 2]
        C3[Request 3] --> CONN3[TCP Connection 3]
        Note1[Head-of-line blocking<br>6 connections per host]
    end
    
    subgraph HTTP/2
        R1[Stream 1] --> MUX[Single TCP Connection<br>multiplexed frames]
        R2[Stream 2] --> MUX
        R3[Stream 3] --> MUX
        Note2[No HOL blocking<br>all streams interleaved]
    end
```

**Why gRPC uses HTTP/2:**
- Multiplexing: many RPCs over one connection
- Header compression (HPACK)
- Server push / bidirectional streaming
- Flow control per-stream

```go
// Go's HTTP/2 is automatic with TLS
srv := &http.Server{
    Addr:      ":443",
    TLSConfig: tlsConfig,
    // HTTP/2 enabled by default when using TLS
}
srv.ListenAndServeTLS("cert.pem", "key.pem")
```

---

## mTLS (Mutual TLS)

```mermaid
sequenceDiagram
    participant C as Client (Service A)
    participant S as Server (Service B)
    participant CA as Certificate Authority
    
    Note over C,S: Standard TLS: only server proves identity
    Note over C,S: mTLS: BOTH sides prove identity
    
    C->>S: ClientHello
    S->>C: ServerHello + Server Certificate
    C->>C: Verify server cert against CA
    S->>C: CertificateRequest
    C->>S: Client Certificate
    S->>S: Verify client cert against CA
    Note over C,S: Both authenticated — encrypted channel
```

### mTLS in Go

```go
// Server: require client certificates
serverTLS := &tls.Config{
    ClientAuth: tls.RequireAndVerifyClientCert,
    ClientCAs:  caCertPool, // CA that signed client certs
    MinVersion: tls.VersionTLS13,
}

// Client: present certificate
clientTLS := &tls.Config{
    Certificates: []tls.Certificate{clientCert},
    RootCAs:      caCertPool, // CA that signed server cert
}
```

### Certificate Rotation

```mermaid
graph TD
    WATCH[File Watcher<br>fsnotify] -->|cert changed| RELOAD[Reload Certificate]
    RELOAD --> ATOMIC[atomic.Pointer swap<br>no downtime]
    ATOMIC --> TLS[tls.Config.GetCertificate<br>returns latest cert]
    
    CRON[Cert Renewal<br>certbot / vault] -->|new cert files| WATCH
```

```go
// Hot-reload certificates without restart
func (s *Server) GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
    return s.cert.Load(), nil // atomic pointer — lock-free
}
```

---

## Connection Cost Summary

| Protocol | RTTs to first byte | Keep-alive? |
|----------|-------------------|-------------|
| TCP | 1 | Yes (reuse connection) |
| TLS 1.2 | 2 (TCP) + 2 (TLS) = 4 | Yes |
| TLS 1.3 | 1 (TCP) + 1 (TLS) = 2 | Yes + 0-RTT resumption |
| HTTP/2 | Same as TLS 1.3 | Multiplexed streams |
| gRPC | Same as HTTP/2 | Long-lived connections |

**Takeaway**: Connection pooling and keep-alive are critical. Never create a new TCP+TLS connection per request.
