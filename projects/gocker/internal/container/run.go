//go:build linux

package container

import (
	"fmt"
	"os"

	"github.com/tour-of-go/gocker/internal/cgroup"
	"github.com/tour-of-go/gocker/internal/image"
	"github.com/tour-of-go/gocker/internal/namespace"
	"github.com/tour-of-go/gocker/internal/overlay"
)

// RunOpts holds all parameters for a container run.
type RunOpts struct {
	Store    *image.Store
	Name     string
	Tag      string
	Hostname string
	Cmd      []string
	Limits   cgroup.Limits
}

// Run is the full container lifecycle: pull-if-missing → overlay → cgroup → namespace → cleanup.
func Run(opts RunOpts) error {
	// Pull image if not cached
	if !opts.Store.Exists(opts.Name, opts.Tag) {
		if err := image.Pull(opts.Store, opts.Name, opts.Tag); err != nil {
			return fmt.Errorf("pull: %w", err)
		}
	}

	id := generateID()
	if opts.Hostname == "" {
		opts.Hostname = id[:8] + "-gocker"
	}

	containerDir := opts.Store.ContainerDir(id)
	if err := os.MkdirAll(containerDir, 0755); err != nil {
		return err
	}
	defer os.RemoveAll(containerDir)

	// Mount OverlayFS
	mergedDir, err := overlay.Mount(opts.Store.RootfsDir(opts.Name, opts.Tag), containerDir)
	if err != nil {
		return fmt.Errorf("overlay: %w", err)
	}
	defer overlay.Unmount(mergedDir)

	// Create cgroup
	cgMgr, err := cgroup.New(id, opts.Limits)
	if err != nil {
		return fmt.Errorf("cgroup: %w", err)
	}
	defer cgMgr.Remove()

	// Fork child with namespaces; child re-execs /proc/self/exe
	return namespace.Run(mergedDir, opts.Hostname, opts.Cmd)
}

func generateID() string {
	f, _ := os.Open("/dev/urandom")
	defer f.Close()
	b := make([]byte, 8)
	f.Read(b)
	return fmt.Sprintf("%x", b)
}
