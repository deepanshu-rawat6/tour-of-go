//go:build linux

package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/tour-of-go/gocker/internal/cgroup"
	"github.com/tour-of-go/gocker/internal/container"
	"github.com/tour-of-go/gocker/internal/image"
	"github.com/tour-of-go/gocker/internal/namespace"
)

func main() {
	// Re-exec detection: if we're the child process inside the new namespaces,
	// run the container setup instead of the normal CLI.
	if os.Getenv(namespace.ChildEnvKey) == "1" {
		mergedDir := os.Getenv("_GOCKER_MERGED")
		hostname := os.Getenv("_GOCKER_HOSTNAME")
		// args[0] is the binary path; actual command starts at args[1]
		args := os.Args[1:]
		if err := namespace.RunChild(mergedDir, hostname, args); err != nil {
			fmt.Fprintln(os.Stderr, "container error:", err)
			os.Exit(1)
		}
		return
	}

	if err := rootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "gocker",
		Short: "gocker — mini container runtime",
	}
	root.AddCommand(pullCmd(), imagesCmd(), runCmd(), rmiCmd())
	return root
}

// --- pull ---

func pullCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pull <image:tag>",
		Short: "Pull an image from Docker Hub",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := image.NewStore()
			if err != nil {
				return err
			}
			name, tag := image.ParseRef(args[0])
			return image.Pull(store, name, tag)
		},
	}
}

// --- images ---

func imagesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "images",
		Short: "List locally cached images",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := image.NewStore()
			if err != nil {
				return err
			}
			list, err := store.List()
			if err != nil {
				return err
			}
			tw := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			fmt.Fprintln(tw, "IMAGE\tTAG\tSIZE")
			for _, ref := range list {
				name, tag := image.ParseRef(ref)
				size, _ := image.DirSize(store.RootfsDir(name, tag))
				fmt.Fprintf(tw, "%s\t%s\t%s\n", name, tag, image.FormatSize(size))
			}
			tw.Flush()
			return nil
		},
	}
}

// --- run ---

func runCmd() *cobra.Command {
	var memStr string
	var cpus float64
	var pidsLimit int64
	var hostname string

	cmd := &cobra.Command{
		Use:   "run [flags] <image:tag> <command> [args...]",
		Short: "Run a command in an isolated container",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := image.NewStore()
			if err != nil {
				return err
			}
			name, tag := image.ParseRef(args[0])

			memBytes, err := parseMemory(memStr)
			if err != nil {
				return fmt.Errorf("--memory: %w", err)
			}

			return container.Run(container.RunOpts{
				Store:    store,
				Name:     name,
				Tag:      tag,
				Hostname: hostname,
				Cmd:      args[1:],
				Limits: cgroup.Limits{
					MemoryBytes: memBytes,
					CPUQuota:    cpus,
					PidsMax:     pidsLimit,
				},
			})
		},
	}
	cmd.Flags().StringVar(&memStr, "memory", "", "Memory limit (e.g. 128m, 1g)")
	cmd.Flags().Float64Var(&cpus, "cpus", 0, "CPU quota (e.g. 0.5 = 50% of one core)")
	cmd.Flags().Int64Var(&pidsLimit, "pids-limit", 0, "Max processes inside the container")
	cmd.Flags().StringVar(&hostname, "hostname", "", "Container hostname (default: random ID)")
	return cmd
}

// --- rmi ---

func rmiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rmi <image:tag>",
		Short: "Delete a cached image",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := image.NewStore()
			if err != nil {
				return err
			}
			name, tag := image.ParseRef(args[0])
			dir := store.ImageDir(name, tag)
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				return fmt.Errorf("image %s not found", args[0])
			}
			if err := os.RemoveAll(dir); err != nil {
				return err
			}
			fmt.Printf("Deleted %s\n", args[0])
			return nil
		},
	}
}

// parseMemory converts "128m", "1g", "512k" to bytes. Empty string → 0 (unlimited).
func parseMemory(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}
	s = strings.ToLower(strings.TrimSpace(s))
	multipliers := map[byte]int64{'k': 1024, 'm': 1024 * 1024, 'g': 1024 * 1024 * 1024}
	last := s[len(s)-1]
	if mul, ok := multipliers[last]; ok {
		n, err := strconv.ParseInt(s[:len(s)-1], 10, 64)
		if err != nil {
			return 0, err
		}
		return n * mul, nil
	}
	return strconv.ParseInt(s, 10, 64)
}
