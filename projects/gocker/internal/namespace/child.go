//go:build linux

package namespace

import (
	"fmt"
	"os"
	"syscall"
)

// RunChild is called in the re-execed child process after namespace creation.
// It sets up the container environment and execs the target command.
func RunChild(mergedDir, hostname string, args []string) error {
	// chroot into the OverlayFS merged directory
	if err := syscall.Chroot(mergedDir); err != nil {
		return fmt.Errorf("chroot: %w", err)
	}
	if err := syscall.Chdir("/"); err != nil {
		return fmt.Errorf("chdir: %w", err)
	}

	// Set container hostname
	if hostname != "" {
		if err := syscall.Sethostname([]byte(hostname)); err != nil {
			return fmt.Errorf("sethostname: %w", err)
		}
	}

	// Mount /proc so ps, top, etc. work correctly inside the container
	if err := os.MkdirAll("/proc", 0755); err != nil {
		return fmt.Errorf("mkdir /proc: %w", err)
	}
	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		return fmt.Errorf("mount /proc: %w", err)
	}

	// exec replaces this process — no return on success
	path, err := lookPath(args[0])
	if err != nil {
		return fmt.Errorf("command not found: %s", args[0])
	}
	return syscall.Exec(path, args, os.Environ())
}

// lookPath searches for a binary in common container PATH locations.
func lookPath(cmd string) (string, error) {
	if len(cmd) > 0 && cmd[0] == '/' {
		return cmd, nil
	}
	for _, dir := range []string{"/usr/local/sbin", "/usr/local/bin", "/usr/sbin", "/usr/bin", "/sbin", "/bin"} {
		p := dir + "/" + cmd
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("not found: %s", cmd)
}
