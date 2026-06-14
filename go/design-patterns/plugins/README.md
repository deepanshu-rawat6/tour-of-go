# Plugin Architecture: Extending Your Platform

A well-designed platform should be extensible without requiring a full recompilation of the core binary. This is essential for building developer tools, observability platforms, or any system where third-party logic is needed.

---

## 🏗️ Three Primary Approaches

### 1. `hashicorp/go-plugin` (via RPC) - **Recommended**
Used by HashiCorp (Terraform, Vault, Nomad) and Grafana.
*   **How it works**: The plugin is a standalone binary. The main application launches it as a subprocess and communicates via gRPC or NetRPC.
*   **Pros**: 
    *   Plugins can't crash the main application.
    *   Plugins can be written in any language (if using gRPC).
    *   Zero versioning conflicts (each process has its own dependencies).
*   **Cons**: Communication overhead (IPC).

### 2. WASM (WebAssembly)
The modern, sandboxed approach for high-performance extensions.
*   **How it works**: Plugins are compiled to `.wasm` and executed by a host runtime (like `wazero` or `wasmer-go`) within the main Go process.
*   **Pros**:
    *   Strong sandboxing (security).
    *   Near-native performance.
*   **Cons**: Complex to pass data across the "WASM boundary".

### 3. Native Go Plugins (`plugin` package)
*   **How it works**: Uses `.so` files (Shared Objects) that are dynamically loaded at runtime.
*   **Pros**: Zero communication overhead (direct memory access).
*   **Cons**: **Extremely fragile**. The main binary and the plugin **must** be compiled with the exact same version of Go and all shared dependencies. (Generally not recommended for production).

---

## 🛠️ Design Pattern: The Interface Boundary

Regardless of the approach, you must define a clear interface.

```go
type PluginInterface interface {
    Name() string
    Execute(ctx context.Context, input []byte) ([]byte, error)
}
```

### Example with `hashicorp/go-plugin`:
1.  Define the interface.
2.  The Main app implements a "Dispenser" to request the plugin.
3.  The Plugin binary implements the interface and starts an RPC server.

---

## 🚀 Key Benefits for Platform Engineers
*   **Versioning**: Roll out new plugin versions without restarting the core platform.
*   **Isolation**: Third-party plugins with memory leaks won't bring down your API gateway.
*   **Diversity**: Allow users to write extensions in Rust, Python, or TypeScript (via gRPC/WASM).

---

## 🛠️ Real-World Use Cases
*   **Terraform**: Every provider (AWS, GCP, etc.) is a standalone Go plugin.
*   **Caddy**: Middleware and server modules are dynamically pluggable.
*   **Prometheus**: Exporters are essentially plugins for the scraping engine.
