# Service Mesh (Istio / Linkerd)

## 1. The Problem

With N microservices, cross-cutting concerns appear in every service:

| Concern | Without mesh | With mesh |
|---|---|---|
| mTLS | Library per service | Sidecar auto-encrypts |
| Retries / timeouts | App code | DestinationRule |
| Circuit breaking | Hystrix/Resilience4j | DestinationRule |
| Distributed traces | Instrumentation code | Envoy auto-injects |
| Access logs | Custom logging | Envoy access log |

---

## 2. Architecture

```mermaid
graph TD
    subgraph CP["Control Plane"]
        istiod["istiod<br/>(Pilot+Citadel+Galley)"]
    end

    subgraph DP["Data Plane"]
        subgraph PodA["Pod A"]
            appA["App Container"]
            envoyA["Envoy Sidecar"]
        end
        subgraph PodB["Pod B"]
            appB["App Container"]
            envoyB["Envoy Sidecar"]
        end
    end

    istiod -->|xDS config| envoyA
    istiod -->|xDS config| envoyB
    istiod -->|issue certs| envoyA
    istiod -->|issue certs| envoyB
    envoyA -->|mTLS traffic| envoyB
    appA -->|localhost| envoyA
    envoyB -->|localhost| appB
```

- **istiod** combines Pilot (service discovery, xDS), Citadel (cert authority), Galley (config validation)
- **Envoy** sidecars intercept all inbound/outbound traffic via iptables rules injected by the init container
- xDS APIs (LDS, RDS, CDS, EDS) push config to proxies without restart

---

## 3. Traffic Management

### VirtualService — routing rules

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: reviews
spec:
  hosts: [reviews]
  http:
  - match:
    - headers:
        end-user:
          exact: test-user
    route:
    - destination:
        host: reviews
        subset: v2
  - route:                      # default: canary split
    - destination:
        host: reviews
        subset: v1
      weight: 90
    - destination:
        host: reviews
        subset: v2
      weight: 10
```

### DestinationRule — subset definitions

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: DestinationRule
metadata:
  name: reviews
spec:
  host: reviews
  subsets:
  - name: v1
    labels:
      version: v1
  - name: v2
    labels:
      version: v2
```

### Traffic Flow with Sidecar

```mermaid
sequenceDiagram
    participant C as Client Pod<br/>(Envoy)
    participant VS as VirtualService<br/>Rule
    participant S1 as reviews-v1<br/>(Envoy)
    participant S2 as reviews-v2<br/>(Envoy)

    C->>VS: HTTP GET /reviews
    VS-->>C: route: 90% v1 / 10% v2
    C->>S1: mTLS (90% traffic)
    C->>S2: mTLS (10% traffic)
    S1-->>C: response
    S2-->>C: response
```

---

## 4. Security

### mTLS — PeerAuthentication

```yaml
# STRICT: only mTLS accepted
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: default
  namespace: production
spec:
  mtls:
    mode: STRICT   # or PERMISSIVE (plain+mTLS)
```

### AuthorizationPolicy

```yaml
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: allow-reviews
  namespace: production
spec:
  selector:
    matchLabels:
      app: reviews
  rules:
  - from:
    - source:
        principals: ["cluster.local/ns/default/sa/productpage"]
    to:
    - operation:
        methods: ["GET"]
```

### mTLS Handshake

```mermaid
sequenceDiagram
    participant EA as Envoy A<br/>(client sidecar)
    participant CA as istiod CA
    participant EB as Envoy B<br/>(server sidecar)

    EA->>CA: CSR (SPIFFE SVID)
    CA-->>EA: signed cert
    EB->>CA: CSR (SPIFFE SVID)
    CA-->>EB: signed cert
    EA->>EB: TLS ClientHello
    EB-->>EA: TLS ServerHello + cert
    EA->>EB: verify cert (SPIFFE ID)
    EB-->>EA: mutual verify done
    EA->>EB: encrypted app traffic
```

SPIFFE ID format: `spiffe://cluster.local/ns/<namespace>/sa/<serviceaccount>`

---

## 5. Observability

Envoy emits metrics automatically — no app instrumentation needed.

Key metrics:
```
istio_requests_total{source_app, destination_app, response_code}
istio_request_duration_milliseconds_bucket
istio_tcp_connections_opened_total
```

Access logs, distributed traces (Jaeger/Zipkin via B3 headers), and Kiali topology graph come out of the box.

---

## 6. Circuit Breaking & Retries

```yaml
apiVersion: networking.istio.io/v1alpha3
kind: DestinationRule
metadata:
  name: payment
spec:
  host: payment
  trafficPolicy:
    connectionPool:
      tcp:
        maxConnections: 100
      http:
        http1MaxPendingRequests: 50
        maxRequestsPerConnection: 10
    outlierDetection:             # circuit breaker
      consecutiveGatewayErrors: 5
      interval: 10s
      baseEjectionTime: 30s
      maxEjectionPercent: 50
    retries:
      attempts: 3
      perTryTimeout: 2s
      retryOn: "5xx,connect-failure"
```

---

## 7. Linkerd vs Istio

| Feature | Linkerd | Istio |
|---|---|---|
| Proxy | Linkerd2-proxy (Rust) | Envoy (C++) |
| Control plane | Lightweight Go binaries | istiod (heavy) |
| Install complexity | Low (`linkerd install \| kubectl apply`) | High (many CRDs) |
| mTLS | Automatic, zero-config | Requires PeerAuthentication |
| Traffic management | Basic (traffic split) | Full (VirtualService, DR) |
| L7 policy | Limited | Full AuthorizationPolicy |
| Resource usage | ~200 MB / proxy | ~500 MB / proxy |
| Learning curve | Low | High |
| Best for | Simplicity, fast mTLS | Full traffic control |

**Rule of thumb**: start with Linkerd if you just need mTLS + basic observability. Use Istio when you need fine-grained traffic policies, canary deployments, or complex AuthorizationPolicies.
