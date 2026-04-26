//go:build linux

package namespace

import (
	"os"
	"os/exec"
	"syscall"
)

// ChildEnvKey is the env var that signals the re-execed child.
const ChildEnvKey = "_GOCKER_CHILD"

// Run forks a child process with all namespace flags set.
// The child re-execs /proc/self/exe with ChildEnvKey set so it knows
// to call RunChild instead of the normal CLI flow.
//
// mergedDir, hostname, and args are passed via environment variables
// to avoid needing a separate IPC channel.
func Run(mergedDir, hostname string, args []string) error {
	cmd := exec.Command("/proc/self/exe", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		ChildEnvKey+"=1",
		"_GOCKER_MERGED="+mergedDir,
		"_GOCKER_HOSTNAME="+hostname,
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS |
			syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWIPC |
			syscall.CLONE_NEWNET,
	}
	return cmd.Run()
}
