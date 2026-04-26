# Design Decisions — xdp-firewall

## 1. XDP over iptables/nftables

**Decision:** Use eBPF/XDP to drop packets, not iptables rules.

**Why:** iptables processes packets after the kernel networking stack has allocated socket buffers and run through the full protocol stack. XDP runs at the NIC driver level — the packet is dropped before any kernel memory is allocated for it. At 10Gbps line rate, this is the difference between the kernel being overwhelmed and the attack being invisible to the OS. XDP can process 24+ million packets/second on a single core; iptables saturates at ~1-2 million.

---

## 2. BPF_MAP_TYPE_LPM_TRIE over BPF_MAP_TYPE_HASH

**Decision:** Use an LPM (Longest Prefix Match) trie for the blacklist, not a hash map.

**Why:** A hash map can only match exact IPs. Blocking `10.0.0.0/8` to cover all 16 million addresses in that range would require 16 million hash map entries. An LPM trie stores one entry per CIDR and the kernel evaluates prefix matches in O(prefix_length) time — exactly how BGP routing tables and AWS Security Groups work. This is the technically correct data structure for a firewall.

---

## 3. BPF_MAP_TYPE_PERCPU_ARRAY for counters

**Decision:** Use per-CPU arrays for packet/drop counters, not a shared atomic integer.

**Why:** Modern NICs process packets across multiple CPU cores simultaneously. A shared atomic counter would cause cache line bouncing — every increment requires a cross-CPU cache invalidation. PERCPU_ARRAY gives each CPU its own counter copy with zero contention. The Go daemon aggregates them by summing across all CPUs on each poll cycle. This is the standard pattern used by Cilium, Cloudflare's XDP firewall, and the Linux kernel's own network statistics.

---

## 4. cilium/ebpf over libbpfgo

**Decision:** Use `github.com/cilium/ebpf` (pure Go) instead of libbpfgo (CGo wrapper).

**Why:** CGo breaks cross-compilation. With libbpfgo, building for `linux/arm64` from a Mac requires a full C cross-compilation toolchain. `cilium/ebpf` is pure Go — it reads the compiled BPF ELF file directly using Go's `encoding/binary`. The same binary can be built on macOS and deployed to any Linux architecture. It also ships `bpf2go`, which generates strongly-typed Go structs from the C source — bringing Go's type safety to kernel memory interactions.

---

## 5. Hexagonal architecture (ports & adapters)

**Decision:** The core `ThreatEngine` depends only on `BPFMapPort` and `RuleStorePort` interfaces, never on concrete implementations.

**Why:** This makes the entire domain logic testable without root, without Linux, without clang. The 22 unit tests run on macOS in CI with mock adapters. It also means the storage backend (file today, Redis tomorrow) or the kernel interface (XDP today, tc/BPF tomorrow) can be swapped without touching the domain. The HTTP adapter, file store adapter, and BPF adapter are all independently replaceable.

---

## 6. Atomic file writes (write-to-tmp + rename)

**Decision:** `Save()` writes to `rules.json.tmp` then calls `os.Rename()`.

**Why:** `os.Rename()` is atomic on Linux and macOS (it's a single `rename(2)` syscall). If the process crashes mid-write, the original `rules.json` is untouched. Without this, a crash during a write could leave a partially-written JSON file that fails to parse on the next startup — causing the daemon to start with an empty blacklist and leaving the kernel map in an unknown state.

---

## 7. Startup reconciliation (file is source of truth)

**Decision:** On startup, diff `rules.json` against the kernel LPM trie and sync both directions.

**Why:** The XDP program lives in the kernel independently of the Go daemon. If the daemon crashes and restarts, the kernel map may have rules that were added before the crash but not yet written to disk (or vice versa). Reconciliation ensures the kernel state always matches the intended state in `rules.json`. File is the source of truth because it's durable; the kernel map is ephemeral (lost on reboot).

---

## 8. noopBPF adapter for non-Linux builds

**Decision:** The `cmd/xdp-fw/main.go` uses a `noopBPF` struct when the real BPF adapter isn't available, rather than failing to compile on macOS.

**Why:** The entire project — domain logic, HTTP API, file persistence, CIDR validation — is testable and runnable on macOS without any Linux kernel or clang dependency. The `//go:build linux` tag on `adapter.go` means the real BPF code only compiles on Linux. This makes local development and CI fast, while the production deployment on Linux gets the real kernel integration.

---

## 9. CIDR validation in the core domain, not the HTTP adapter

**Decision:** `validateCIDR()` lives in `internal/core/engine.go`, not in the HTTP handler.

**Why:** Validation is a domain concern — the rule "a CIDR must be a valid IPv4 prefix" is true regardless of whether the request came from HTTP, a CLI flag, or a file import. Putting it in the HTTP adapter would mean the batch and import paths need their own validation. In the core, it's validated once for all entry points.

---

## 10. Delta-based Prometheus counter updates

**Decision:** The metrics poller tracks previous counter values and adds the delta to Prometheus counters, rather than using `Set()`.

**Why:** Prometheus counters must be monotonically increasing — you can only `Add()` to them, never `Set()`. The per-CPU eBPF counters are also monotonically increasing (they never reset). The poller reads the current total, subtracts the previous total, and adds the delta. This correctly handles the case where the daemon restarts (the eBPF counters keep counting) — the Prometheus counter picks up from where it left off rather than resetting to zero.
