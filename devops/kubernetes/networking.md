# Kubernetes Networking

---

## Services — The Stable Endpoint Abstraction

Pods are ephemeral — they die and get new IPs. A **Service** is a stable virtual IP (ClusterIP) that load-balances to a dynamic set of pods selected by label.

```mermaid
graph TD
    classDef client fill:#3498db,stroke:#2980b9,color:#fff,rx:8
    classDef service fill:#9b59b6,stroke:#8e44ad,color:#fff,rx:8
    classDef pod fill:#2ecc71,stroke:#27ae60,color:#fff,rx:8
    classDef proxy fill:#e67e22,stroke:#d35400,color:#fff,rx:8
    classDef dns fill:#1abc9c,stroke:#16a085,color:#fff,rx:8

    CLIENT["Other Pod / Ingress"]:::client -->|"my-svc.default.svc.cluster.local:80"| DNS["CoreDNS resolves to ClusterIP 10.96.45.20"]:::dns
    DNS --> SVC["Service: my-svc ClusterIP: 10.96.45.20 port: 80 → targetPort: 8080"]:::service
    SVC -->|"kube-proxy iptables NAT"| P1["Pod 1 10.0.1.5:8080"]:::pod
    SVC --> P2["Pod 2 10.0.2.7:8080"]:::pod
    SVC --> P3["Pod 3 10.0.3.9:8080"]:::pod
```

---

## ClusterIP Internals — How kube-proxy Programs iptables

ClusterIP is **not** a real IP with a process listening on it. It's a virtual IP that exists only in iptables NAT rules. When a packet hits the ClusterIP, the kernel rewrites the destination to one of the backend pod IPs before sending.

```mermaid
sequenceDiagram
    participant APP as App Pod (10.0.1.5)
    participant KERN as Linux Kernel (iptables)
    participant EP as EndpointSlice (10.0.2.7, 10.0.3.9)
    participant DEST as Destination Pod

    APP->>KERN: connect(10.96.45.20:80)
    Note over KERN: PREROUTING chain hits KUBE-SERVICES
    KERN->>KERN: Match: dst=10.96.45.20 port=80 → jump KUBE-SVC-XXX
    KERN->>KERN: KUBE-SVC-XXX: random select 1 of N backends
    KERN->>KERN: DNAT: rewrite dst to 10.0.2.7:8080
    KERN->>DEST: Packet delivered to real pod IP
    DEST-->>KERN: Response src=10.0.2.7:8080
    KERN->>KERN: conntrack: rewrite src back to 10.96.45.20:80
    KERN-->>APP: Response appears to come from ClusterIP
```

**iptables chain hierarchy:**
```
PREROUTING
  └── KUBE-SERVICES
        └── KUBE-SVC-XXXXXXXX  (per Service)
              ├── KUBE-SEP-AAAA  (endpoint 1, 33% probability)
              ├── KUBE-SEP-BBBB  (endpoint 2, 50% of remaining)
              └── KUBE-SEP-CCCC  (endpoint 3, 100% of remaining)
                    └── DNAT to pod IP:port
```

**IPVS mode** (alternative to iptables): creates a virtual server in the kernel's IPVS table instead. Scales better for large clusters (1000s of services) — O(1) lookup vs O(N) iptables scan.

---

## Service Types

```mermaid
graph TD
    classDef svctype fill:#8e44ad,stroke:#6c3483,color:#fff,rx:6
    classDef infra fill:#2c3e50,stroke:#1a252f,color:#fff,rx:6
    classDef traffic fill:#27ae60,stroke:#1e8449,color:#fff,rx:6
    classDef note fill:#f39c12,stroke:#d68910,color:#000,rx:6

    subgraph Types["Service Types"]
        CIP["ClusterIP Virtual IP, cluster-internal only Default type"]:::svctype
        NP["NodePort Exposes on every node's IP:port (30000-32767) Reachable from outside cluster"]:::svctype
        LB["LoadBalancer Provisions cloud LB (ALB/NLB) Entrypoint for external traffic"]:::svctype
        HS["Headless (ClusterIP: None) No virtual IP DNS returns pod IPs directly"]:::svctype
    end

    EXT["External Traffic"]:::traffic -->|"DNS → ALB/NLB"| LB
    LB -->|"forwards to NodePort"| NP
    NP -->|"NodePort → ClusterIP → pod"| CIP
    HS -.- NOTE["Used by StatefulSets: payments-0.my-svc, payments-1.my-svc Each pod gets stable DNS name"]:::note
```

| Type | Access | Use case |
|------|--------|----------|
| `ClusterIP` | Inside cluster only | Service-to-service communication |
| `NodePort` | `<NodeIP>:<30000-32767>` | Dev/testing, bare-metal clusters |
| `LoadBalancer` | Cloud LB public IP | Production external traffic |
| `Headless` | DNS → pod IPs directly | StatefulSets, service discovery |
| `ExternalName` | CNAME to external DNS | Database in RDS, external service aliasing |

---

## DNS in Kubernetes

```mermaid
graph LR
    classDef pod fill:#3498db,stroke:#2980b9,color:#fff,rx:6
    classDef dns fill:#1abc9c,stroke:#16a085,color:#fff,rx:6
    classDef svc fill:#9b59b6,stroke:#8e44ad,color:#fff,rx:6

    POD["Pod in namespace: payments"]:::pod -->|"1. DNS query"| RESOLVE["Pod's /etc/resolv.conf nameserver: 10.96.0.10 (CoreDNS ClusterIP) search: payments.svc.cluster.local svc.cluster.local cluster.local"]:::pod

    RESOLVE --> COREDNS["CoreDNS 10.96.0.10:53"]:::dns

    COREDNS -->|"my-svc"| FULL1["Appends search domains: my-svc.payments.svc.cluster.local ✅ found"]:::dns
    COREDNS -->|"my-svc.other-ns"| FULL2["my-svc.other-ns.svc.cluster.local ✅"]:::dns
    COREDNS -->|"FQDN"| FULL3["my-svc.other-ns.svc.cluster.local ✅"]:::dns

    FULL1 --> SVC_IP["Returns ClusterIP 10.96.45.20"]:::svc
```

**DNS name formats:**

| Format | Resolves to | Notes |
|--------|------------|-------|
| `my-svc` | ClusterIP (same namespace) | Search domain appended |
| `my-svc.other-ns` | ClusterIP in other-ns | Cross-namespace |
| `my-svc.other-ns.svc.cluster.local` | ClusterIP (FQDN) | Explicit, always works |
| `payments-0.my-headless-svc.ns.svc.cluster.local` | Pod IP directly | StatefulSet pod DNS |
| `_http._tcp.my-svc.ns.svc.cluster.local` | SRV record | Port discovery |

**ndots:5** — pods have `ndots:5` in resolv.conf. Names with fewer than 5 dots trigger search domain expansion before trying as-is. `api.example.com` (3 dots) tries `api.example.com.payments.svc.cluster.local` first, then falls through. Use FQDN with trailing dot for external names to skip search: `api.example.com.`

---

## Ingress

Ingress is an HTTP(S) reverse proxy configuration. An Ingress resource defines routing rules; an **Ingress Controller** (nginx, AWS ALB Controller, Traefik) implements them.

```mermaid
graph TD
    classDef ext fill:#e74c3c,stroke:#c0392b,color:#fff,rx:6
    classDef lb fill:#f39c12,stroke:#d68910,color:#fff,rx:6
    classDef ing fill:#9b59b6,stroke:#8e44ad,color:#fff,rx:6
    classDef svc fill:#3498db,stroke:#2980b9,color:#fff,rx:6
    classDef pod fill:#2ecc71,stroke:#27ae60,color:#fff,rx:6

    USER["User: GET https://api.example.com/payments"]:::ext
    USER --> ALB["ALB / nginx (Ingress Controller)"]:::lb
    ALB -->|"host: api.example.com path: /payments/*"| SVC1["Service: payments-svc ClusterIP"]:::svc
    ALB -->|"host: api.example.com path: /orders/*"| SVC2["Service: orders-svc ClusterIP"]:::svc
    ALB -->|"host: admin.example.com"| SVC3["Service: admin-svc ClusterIP"]:::svc
    SVC1 --> P1["payments pods"]:::pod
    SVC2 --> P2["orders pods"]:::pod
    SVC3 --> P3["admin pods"]:::pod
```

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: api-ingress
  annotations:
    kubernetes.io/ingress.class: "alb"
    alb.ingress.kubernetes.io/scheme: "internet-facing"
    alb.ingress.kubernetes.io/certificate-arn: "arn:aws:acm:..."
spec:
  rules:
    - host: api.example.com
      http:
        paths:
          - path: /payments
            pathType: Prefix
            backend:
              service:
                name: payments-svc
                port:
                  number: 80
          - path: /orders
            pathType: Prefix
            backend:
              service:
                name: orders-svc
                port:
                  number: 80
  tls:
    - hosts: [api.example.com]
      secretName: api-tls-cert
```

---

## NetworkPolicy — Pod-Level Firewall

By default, all pods can talk to all pods. NetworkPolicy restricts this at the CNI level (Calico, Cilium enforce policies; vanilla flannel does not).

```mermaid
graph LR
    classDef allowed fill:#2ecc71,stroke:#27ae60,color:#fff,rx:6
    classDef blocked fill:#e74c3c,stroke:#c0392b,color:#fff,rx:6
    classDef policy fill:#f39c12,stroke:#d68910,color:#fff,rx:6
    classDef pod fill:#3498db,stroke:#2980b9,color:#fff,rx:6

    subgraph PaymentsNS["namespace: payments"]
        PP["payments pod app=payments"]:::pod
        POLICY["NetworkPolicy: allow ingress from app=api only allow egress to app=postgres only"]:::policy
    end

    API["api pod app=api"]:::pod -->|"TCP 8080 ✅ allowed"| PP
    MALICIOUS["other pod app=worker"]:::blocked -->|"TCP 8080 ❌ blocked by NetworkPolicy"| PP
    PP -->|"TCP 5432 ✅ allowed"| POSTGRES["postgres pod app=postgres"]:::pod
    PP -->|"TCP 443 ❌ blocked"| EXTERNAL["external API"]:::blocked
```

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: payments-policy
  namespace: payments
spec:
  podSelector:
    matchLabels:
      app: payments       # this policy applies to payments pods
  policyTypes: [Ingress, Egress]
  ingress:
    - from:
        - podSelector:
            matchLabels:
              app: api    # only allow from api pods
      ports:
        - port: 8080
  egress:
    - to:
        - podSelector:
            matchLabels:
              app: postgres
      ports:
        - port: 5432
    - to:                 # allow DNS (always needed)
        - namespaceSelector: {}
      ports:
        - port: 53
          protocol: UDP
```

**Important:** If you apply a NetworkPolicy to a pod, ALL traffic not explicitly allowed is denied. Don't forget to allow DNS (port 53 UDP) in egress — pods will break without it.
