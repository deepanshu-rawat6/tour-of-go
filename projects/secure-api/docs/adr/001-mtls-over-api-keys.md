# ADR-001: mTLS over API Keys for Service-to-Service Auth

**Status:** Accepted

## Decision

Use mutual TLS (mTLS) for service-to-service authentication instead of shared API keys.

## Rationale

| Concern | API Keys | mTLS |
|---|---|---|
| Secret rotation | Manual, error-prone | Certificate expiry + auto-rotation (cert-manager) |
| Identity proof | Possession of a string | Cryptographic proof via private key |
| Transport encryption | Separate (TLS still needed) | Built-in — auth and encryption in one layer |
| Revocation | Delete key from store | CRL / OCSP or short-lived certs |
| Kubernetes native | Secret mounts | cert-manager + service mesh (Istio/Linkerd) |

## Consequences

- Requires a CA and certificate lifecycle management (`make certs` for local dev).
- Client services must present a valid cert signed by the trusted CA.
- For human-facing clients (browsers, mobile), JWT over HTTPS is still used — mTLS is only for service-to-service.
