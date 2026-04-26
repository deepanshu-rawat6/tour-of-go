# How gocker Works — Deep Dive

A technical walkthrough of every kernel mechanism gocker uses, and how they map to what Docker does internally.

---

## 1. Linux Namespaces

Namespaces are the core isolation primitive. Each namespace type wraps a specific global resource so that processes inside the namespace see their own isolated copy.

### What gocker uses

```go
cmd.SysProcAttr = &syscall.SysProcAttr{
    Cloneflags: syscall.CLONE_NEWPID |  // isolated process tree
                syscall.CLONE_NEWNS  |  // isolated mount table
                syscall.CLONE_NEWUTS |  // isolated hostname
                syscall.CLONE_NEWIPC |  // isolated IPC resources
                syscall.CLONE_NEWNET,   // isolated network stack
}
```

### PID namespace (`CLONE_NEWPID`)

The first process in the new PID namespace gets PID 1. From inside the container:

```
/ # ps
PID   COMMAND
  1   /bin/sh     ← this is actually PID 38291 on the host
  4   ps
```

The host can still see the real PID. The container cannot see host processes.

### Mount namespace (`CLONE_NEWNS`)

After cloning, the child has a copy of the parent's mount table. gocker then:
1. Calls `syscall.Chroot(mergedDir)` to change the root
2. Mounts `/proc` fresh inside the new root so `ps` works correctly
3. Any mounts made inside the container don't leak to the host

### UTS namespace (`CLONE_NEWUTS`)

Allows setting an independent hostname:
```go
syscall.Sethostname([]byte(containerID))
```

### Network namespace (`CLONE_NEWNET`)

Creates an isolated network stack. The container has a loopback interface only — no `eth0`, no internet. This is equivalent to `docker run --network none`.

---

## 2. OverlayFS

OverlayFS is a union filesystem that presents a merged view of two directories: a read-only lower layer and a writable upper layer.

### Mount command equivalent

```bash
mount -t overlay overlay \
  -o lowerdir=/path/to/image/rootfs,\
     upperdir=/path/to/container/upper,\
     workdir=/path/to/container/work \
  /path/to/container/merged
```

### What happens on writes

| Operation | Result |
|-----------|--------|
| Read a file | Served from `lower/` (image) |
| Write a file | Copy-on-write: file copied to `upper/`, modified there |
| Delete a file | A "whiteout" file created in `upper/` hides the lower file |
| Create a file | Written directly to `upper/` |

The image `rootfs/` (lower layer) is **never modified**. Multiple containers can share the same image simultaneously.

### In Go

```go
opts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", lower, upper, work)
syscall.Mount("overlay", merged, "overlay", 0, opts)
```

---

## 3. cgroups (Control Groups)

cgroups limit and account for resource usage of process groups.

### v1 vs v2

**cgroups v1** (older): separate filesystem hierarchies per controller
```
/sys/fs/cgroup/memory/gocker/<id>/memory.limit_in_bytes
/sys/fs/cgroup/cpu/gocker/<id>/cpu.cfs_quota_us
/sys/fs/cgroup/pids/gocker/<id>/pids.max
```

**cgroups v2** (modern, unified): single hierarchy
```
/sys/fs/cgroup/gocker/<id>/memory.max
/sys/fs/cgroup/gocker/<id>/cpu.max
/sys/fs/cgroup/gocker/<id>/pids.max
```

gocker detects which version is active by checking for `/sys/fs/cgroup/cgroup.controllers`.

### Applying limits

```go
// Memory: write bytes as string
os.WriteFile(".../memory.max", []byte("134217728"), 0700)  // 128 MiB

// CPU: quota/period format (v2) — 50000/100000 = 50% of one core
os.WriteFile(".../cpu.max", []byte("50000 100000"), 0700)

// PIDs
os.WriteFile(".../pids.max", []byte("20"), 0700)

// Add the container PID to the cgroup
os.WriteFile(".../cgroup.procs", []byte(strconv.Itoa(pid)), 0700)
```

---

## 4. chroot

`chroot` changes the root directory for a process. After `chroot(mergedDir)`, the process sees the OverlayFS merged directory as `/`.

```go
syscall.Chroot(mergedDir)
syscall.Chdir("/")
```

Combined with `CLONE_NEWNS`, this is fully isolated — the process cannot `chroot` back out.

---

## 5. OCI Image Pull

gocker pulls images from Docker Hub using the OCI Distribution Specification.

### Flow

```
1. GET /token?service=registry.docker.io&scope=repository:library/alpine:pull
   → Bearer token for anonymous pull

2. GET /v2/library/alpine/manifests/3.19
   Accept: application/vnd.docker.distribution.manifest.v2+json
   → JSON manifest listing layer digests

3. For each layer digest:
   GET /v2/library/alpine/blobs/sha256:<digest>
   → Download gzipped tar
   → Extract into rootfs/ in order (later layers overwrite earlier ones)
```

Layers already on disk (matched by digest) are skipped — content-addressed caching.

---

## 6. The Re-exec Trick

Go's runtime starts multiple threads before `main()` runs, which breaks `fork()`-based namespace creation. The standard solution is to re-exec the binary itself as the child process with a hidden flag:

```
Parent: exec.Cmd{Path: "/proc/self/exe", Args: ["gocker", "__child", ...]}
         └─ clone with namespace flags
Child:  detects "__child" arg → runs container setup (chroot, mount /proc, exec cmd)
```

This is the same pattern used by `runc` (the OCI runtime Docker uses under the hood).

---

## 7. Cleanup

gocker uses `defer` to guarantee cleanup even on error or panic:

```go
mergedDir, err := overlay.Mount(imageRootfs, containerDir)
defer overlay.Unmount(mergedDir)        // always unmount

cgroupMgr, err := cgroup.New(id, limits)
defer cgroupMgr.Remove()               // always delete cgroup

// ... run container ...
// container dir (upper/, work/, merged/) removed after unmount
```

After the container exits: OverlayFS is unmounted, the cgroup is deleted, and the container directory (including the writable `upper/` layer) is removed. The image `rootfs/` remains intact for the next run.

---

## How gocker Compares to Docker

| Feature | gocker | Docker |
|---------|--------|--------|
| PID isolation | `CLONE_NEWPID` | `CLONE_NEWPID` via runc |
| Filesystem | OverlayFS (chroot) | overlay2 driver (pivot_root) |
| Resource limits | cgroups v1/v2 | cgroups v1/v2 |
| Image format | OCI manifest + layer blobs | OCI / Docker image spec |
| Networking | Isolated (no veth) | Bridge network with veth pairs |
| Image layers | Merged into single rootfs | Deduplicated layer cache |
| Security | Root only | rootless mode available |
| Production use | Learning/research | Yes |
