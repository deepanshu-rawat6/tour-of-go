# eBPF with Go: The Modern SRE Superpower

eBPF (Extended Berkeley Packet Filter) allows you to run sandboxed programs directly within the Linux Kernel without changing the kernel's source code or loading new modules. For Go developers, this enables building incredibly high-performance observability, networking, and security tools.

---

## 🏗️ How eBPF Works with Go

1.  **Write C Code**: You write your "probe" or "filter" in a restricted subset of C.
2.  **Compile to BPF Bytecode**: Using `clang` and `llvm`, you compile the C code into `.o` bytecode.
3.  **The Go Host**: You write a Go program that loads the bytecode into the kernel using the `cilium/ebpf` or `aquasecurity/libbpfgo` libraries.
4.  **Verification**: The kernel "Verifier" checks the code for safety (no infinite loops, no invalid memory access).
5.  **JIT Compilation**: The bytecode is JIT-compiled into native machine code for the CPU.

---

## 🛠️ The Architecture: Go as the "User-Space Host"

Go is responsible for the "Control Plane" of your eBPF program:
*   **Loading**: Pulling the BPF program into the kernel.
*   **Maps**: Using shared data structures (BPF Maps) to communicate between the Kernel (C) and User-space (Go).
*   **Event Handling**: Reading high-speed events from the kernel via "Perf Buffers" or "Ring Buffers".

---

## 🏎️ Why eBPF?

*   **Zero-Copy Observability**: Observe every syscall, network packet, or function call without the overhead of context-switching between User-space and Kernel-space.
*   **Security Enforcement**: Block malicious connections or unauthorized file access directly at the kernel level (e.g., Tetragon).
*   **High-Speed Networking**: Implement Load Balancers (like Cilium) that process packets before they even reach the standard Linux networking stack.

---

## 🛠️ Example Scenario: Tracking `execve` Syscalls

1.  **BPF Probe (C)**: Intercepts the `execve` syscall and sends the process name to a BPF Map.
2.  **Go Program**: Listens to the Map and logs whenever a new process starts on the system.

```go
// Using cilium/ebpf (Pseudo-code)
objs := bpfObjects{}
if err := loadBpfObjects(&objs, nil); err != nil { ... }
defer objs.Close()

// Attach to the 'execve' tracepoint
tp, _ := link.Tracepoint("syscalls", "sys_enter_execve", objs.ExecveProbe, nil)
```

---

## 🚀 Key Benefits for Platform Engineers
*   **Deep Visibility**: Inspect encrypted traffic (TLS) before it gets encrypted or after it's decrypted in the kernel.
*   **Resource Efficiency**: Far less CPU overhead than traditional polling-based monitoring tools.
*   **Kernel Safety**: Run code in the kernel without the risk of a Kernel Panic (thanks to the Verifier).

---

## 🛠️ Popular Go + eBPF Projects
*   **Cilium**: Cloud-native networking and security (uses eBPF for the data plane).
*   **Tetragon**: Security observability and runtime enforcement.
*   **Pixie**: Auto-telemetry for Kubernetes using eBPF.
