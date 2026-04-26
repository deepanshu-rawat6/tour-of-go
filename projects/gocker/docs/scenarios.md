# Use Cases & Scenarios

Real-world and learning scenarios where gocker is the right tool.

---

## 1. Learning How Docker Actually Works

**The scenario:** You've used Docker for years but don't know what happens when you type `docker run`. You want to understand it at the kernel level.

**How gocker helps:** Every line of gocker maps directly to a Docker internal:

| gocker code | Docker equivalent |
|-------------|------------------|
| `CLONE_NEWPID` | PID namespace in `runc` |
| `syscall.Chroot(mergedDir)` | `pivot_root` in production runtimes |
| OverlayFS mount | Docker's storage driver (overlay2) |
| cgroup write | Docker's `--memory`, `--cpus` flags |
| OCI manifest fetch | `docker pull` internals |

After building gocker, `docker run` is no longer magic — it's just these syscalls with more error handling.

```bash
# Prove PID isolation
sudo ./gocker run alpine:3.19 ps
# PID   COMMAND
#   1   ps        ← only sees itself, not 200+ host processes
```

---

## 2. Security Research — Namespace Escape Testing

**The scenario:** You're a security engineer studying container escape vulnerabilities. You need a minimal runtime where you can observe exactly what isolation is and isn't in place.

**How gocker helps:** Because gocker is ~500 lines of Go with no abstraction layers, you can:
- Remove a namespace flag and observe what leaks (e.g., remove `CLONE_NEWPID` and see host PIDs)
- Test what happens without `CLONE_NEWNET` (container can see host network interfaces)
- Experiment with cgroup limit bypass attempts

```bash
# What does the container see without PID namespace?
# Edit namespace/ns.go, remove CLONE_NEWPID, rebuild
sudo ./gocker run alpine:3.19 ps aux
# Now shows all host processes — demonstrates why PID ns matters
```

---

## 3. Resource Limit Enforcement Testing

**The scenario:** You're a platform engineer writing admission policies for a Kubernetes cluster. You want to understand what `resources.limits.memory` actually enforces at the kernel level before writing OPA policies.

**How gocker helps:** You can directly observe cgroup enforcement:

```bash
# Limit to 32MB memory
sudo ./gocker run --memory 32m alpine:3.19 /bin/sh

# Inside the container — try to allocate more than the limit
/ # dd if=/dev/zero of=/dev/null bs=1M count=100
# Process gets OOM-killed by the kernel — cgroup enforcement in action

# Verify the limit was written
cat /sys/fs/cgroup/gocker/<id>/memory.max
# 33554432
```

---

## 4. CI/CD Sandboxing (Conceptual)

**The scenario:** You want to understand how CI systems like GitHub Actions or Jenkins agents run untrusted build scripts in isolation.

**How gocker helps:** gocker demonstrates the exact mechanism:
- Each job gets a fresh OverlayFS upper layer (writes don't persist between jobs)
- PID namespace means a runaway build process can't kill the host agent
- cgroup limits prevent a build from consuming all host memory/CPU
- Network namespace can isolate builds that shouldn't reach the internet

```bash
# Simulate a CI job: run a build script in isolation with resource limits
sudo ./gocker run --memory 512m --cpus 1.0 --pids-limit 50 alpine:3.19 \
  /bin/sh -c "apk add --no-cache go && go build ./..."
# Container exits → upper layer deleted → host is clean
```

---

## 5. Filesystem Isolation Demonstration

**The scenario:** You're teaching a workshop on containers and want to show concretely that a container's filesystem is isolated from the host.

**How gocker helps:** OverlayFS makes this tangible:

```bash
# Run a container and create a file
sudo ./gocker run alpine:3.19 /bin/sh -c "echo hello > /proof.txt && cat /proof.txt"
# hello

# After the container exits, check the host
ls /proof.txt
# No such file — it was in the container's upper/ layer, now deleted

# The image rootfs is also untouched
ls ~/.gocker/images/alpine:3.19/rootfs/proof.txt
# No such file
```

---

## 6. Understanding OCI Image Format

**The scenario:** You're building a custom image registry or working with container image tooling (Skopeo, Crane, Buildah) and need to understand the OCI Distribution Spec.

**How gocker helps:** The pull implementation is a minimal, readable OCI client:
- Token auth flow for Docker Hub
- Manifest v2 parsing (layer digests, media types)
- Layer blob download and extraction
- Content-addressed caching by digest

Reading `internal/image/pull.go` gives you a complete picture of what `docker pull` does over the wire, without the complexity of the full Docker client.

```bash
# Pull and inspect what was downloaded
sudo ./gocker pull alpine:3.19
ls -la ~/.gocker/images/alpine:3.19/rootfs/
# bin  dev  etc  home  lib  media  mnt  opt  proc  root  run  sbin  srv  sys  tmp  usr  var
# A complete Alpine Linux root filesystem
```

---

## Summary

| Scenario | Primary concept learned | Audience |
|----------|------------------------|----------|
| Learning Docker internals | Namespaces, OverlayFS, cgroups | Developers, SREs |
| Security research | Namespace isolation boundaries | Security engineers |
| Resource limit testing | cgroup v1/v2 enforcement | Platform engineers |
| CI/CD sandboxing | Ephemeral container model | DevOps engineers |
| Filesystem isolation demo | OverlayFS copy-on-write | Workshop instructors |
| OCI image format | OCI Distribution Spec | Tooling developers |
