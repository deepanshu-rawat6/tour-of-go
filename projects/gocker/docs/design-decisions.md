# Design Decisions — gocker

Why the implementation is structured the way it is.

---

## 1. Re-exec trick instead of `fork()` directly

**Decision:** The parent process re-execs `/proc/self/exe` (itself) with a hidden env var `_GOCKER_CHILD=1` to create the child inside the new namespaces. The child detects this env var in `main()` and runs container setup instead of the CLI.

**Why:** Go's runtime starts multiple OS threads before `main()` runs (for the garbage collector, goroutine scheduler, etc.). Linux's `clone()` syscall with namespace flags only isolates the calling thread — the other threads remain in the parent's namespaces. This makes direct `fork()`-based namespace creation unreliable in Go. Re-execing the binary as a fresh process ensures the child starts with a single thread in the correct namespaces. This is the exact same pattern used by `runc` (the OCI runtime Docker uses) and `libcontainer`.

---

## 2. OverlayFS over bind mounts or full copy

**Decision:** Use OverlayFS (`mount -t overlay`) with a read-only lower layer (image rootfs) and a writable upper layer (per-container directory).

**Why:** Three alternatives were considered:
- **Full copy of rootfs per container:** Simple but slow (Alpine is ~7MB; copying on every `run` adds latency) and wastes disk space.
- **Bind mount of image rootfs:** Fast but the container can modify the image, breaking isolation between runs.
- **OverlayFS:** The image is never touched. Each container gets a fresh writable layer. Writes are copy-on-write — only modified files land in `upper/`. This is exactly how Docker's `overlay2` storage driver works, making gocker a faithful implementation of the real thing.

---

## 3. cgroups v1/v2 auto-detection at runtime

**Decision:** Check for `/sys/fs/cgroup/cgroup.controllers` at runtime to determine which cgroup version is active, then use the appropriate file paths.

**Why:** cgroups v2 is the default on Ubuntu 22.04+, Fedora 31+, and Arch. cgroups v1 is still common on older distros and some cloud VM images. Hardcoding either version would make gocker fail silently on half of Linux systems. Runtime detection with a single `os.Stat` call costs nothing and makes the tool work correctly everywhere.

---

## 4. `//go:build linux` on all syscall-using files

**Decision:** Every file that uses `syscall.Mount`, `syscall.Chroot`, `syscall.CLONE_*`, or cgroup file paths has a `//go:build linux` build tag. The image store and pull packages have no build tag.

**Why:** This allows `go build ./...` and `go test ./internal/image/...` to run on macOS for development and CI, while the Linux-specific packages are only compiled when targeting Linux. Without build tags, the project would fail to compile on macOS entirely, making local development impossible. The image store and OCI pull code is pure Go with no OS-specific syscalls, so it runs and tests correctly on any platform.

---

## 5. `syscall.Exec` in the child, not `exec.Command`

**Decision:** The child process calls `syscall.Exec` (which replaces the current process image) rather than `exec.Command` (which forks a new child).

**Why:** After `chroot` and namespace setup, we want the user's command to be PID 1 inside the container — the first and only process. Using `exec.Command` would make the gocker child process PID 1 and the user's command PID 2. `syscall.Exec` replaces the current process in-place, so the user's command becomes PID 1 directly. This matches Docker's behavior and means signals (SIGTERM, SIGINT) go directly to the user's process.

---

## 6. Narrow OCI client — no registry library dependency

**Decision:** Implement the Docker Hub pull flow directly using `net/http` and `encoding/json`. No third-party OCI registry client library.

**Why:** The full OCI pull flow for public Docker Hub images requires exactly three HTTP calls: token fetch, manifest fetch, and blob download per layer. A third-party library (like `google/go-containerregistry`) would add significant dependency weight for functionality that fits in ~100 lines. Implementing it directly also makes the code readable as a learning artifact — you can see exactly what `docker pull` does over the wire.

---

## 7. `defer` for all cleanup

**Decision:** OverlayFS unmount, cgroup removal, and container directory deletion are all registered with `defer` immediately after creation.

**Why:** Container setup has multiple failure points (overlay mount fails, cgroup creation fails, namespace clone fails). Without `defer`, each failure path would need explicit cleanup code, and it's easy to miss a case. `defer` guarantees cleanup runs regardless of which step fails or panics. The order matters: overlay must be unmounted before the container directory is removed, and `defer` executes in LIFO order — which is exactly right since the directory is created first and unmounted last.

---

## 8. Image store at `~/.gocker/` with `name:tag` directory names

**Decision:** Cache images at `~/.gocker/images/<name>:<tag>/rootfs/` and container working dirs at `~/.gocker/containers/<uuid>/`.

**Why:** Using the home directory means the tool works without root for the image cache itself (only the `run` command needs root for syscalls). The `name:tag` directory naming is human-readable and directly maps to the image reference — no separate metadata database needed. Container directories use random UUIDs to avoid collisions when multiple containers run concurrently.

---

## 9. Network namespace isolated but no veth wiring

**Decision:** `CLONE_NEWNET` is included in the clone flags (container gets an isolated network namespace) but no virtual ethernet pair or NAT is configured.

**Why:** Full container networking requires creating a `veth` pair, moving one end into the container's network namespace, assigning IP addresses, and setting up NAT rules via `iptables`. This is a significant amount of code that goes beyond the core learning objectives (namespaces, OverlayFS, cgroups). The network namespace is still created — the container is isolated from the host network — but it has no internet access. This is equivalent to `docker run --network none` and is clearly documented as a known limitation.

---

## 10. Single merged rootfs on pull (no layer deduplication)

**Decision:** All OCI image layers are extracted sequentially into a single `rootfs/` directory on `gocker pull`. There is no per-layer storage or deduplication across images.

**Why:** True layer deduplication (like Docker's content-addressable layer store) requires a separate metadata database, layer reference counting, and a more complex storage layout. For a learning project, the complexity cost outweighs the benefit. The current approach is correct and simple: layers are applied in order (later layers overwrite earlier ones, whiteout files handle deletions), producing a correct merged rootfs. The tradeoff — disk space is not shared between images with common base layers — is acceptable at this scale.
