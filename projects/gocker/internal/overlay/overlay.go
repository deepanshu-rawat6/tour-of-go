//go:build linux

package overlay

import (
	"fmt"
	"os"
	"syscall"
)

// Mount creates an OverlayFS with imageRootfs as the read-only lower layer.
// Returns the merged directory path (the chroot target).
func Mount(imageRootfs, containerDir string) (string, error) {
	upper := containerDir + "/upper"
	work := containerDir + "/work"
	merged := containerDir + "/merged"

	for _, d := range []string{upper, work, merged} {
		if err := os.MkdirAll(d, 0755); err != nil {
			return "", err
		}
	}

	opts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", imageRootfs, upper, work)
	if err := syscall.Mount("overlay", merged, "overlay", 0, opts); err != nil {
		return "", fmt.Errorf("overlayfs mount: %w", err)
	}
	return merged, nil
}

// Unmount unmounts the OverlayFS merged directory.
func Unmount(merged string) error {
	return syscall.Unmount(merged, 0)
}
