package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/tour-of-go/xdp-firewall/internal/adapters/filestore"
	httpadapter "github.com/tour-of-go/xdp-firewall/internal/adapters/http"
	"github.com/tour-of-go/xdp-firewall/internal/config"
	"github.com/tour-of-go/xdp-firewall/internal/core"
	appmetrics "github.com/tour-of-go/xdp-firewall/internal/metrics"
)

func main() {
	var cfgFile string
	var ifaceOverride string

	root := &cobra.Command{
		Use:   "xdp-fw",
		Short: "XDP Firewall — kernel-level packet filter with CIDR blacklist",
		Long: `Loads a compiled eBPF/XDP program into the Linux kernel and manages
a CIDR blacklist via an HTTP admin API. Packets from blacklisted IPs/subnets
are dropped at the NIC driver level before the OS networking stack sees them.

Requires root (CAP_NET_ADMIN) and Linux kernel 5.8+.`,
	}
	root.PersistentFlags().StringVarP(&cfgFile, "config", "c", "config.yaml", "path to config file")
	root.PersistentFlags().StringVar(&ifaceOverride, "interface", "", "override network interface from config")

	start := &cobra.Command{
		Use:   "start",
		Short: "Load XDP program and start the admin API",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cfgFile, ifaceOverride)
		},
	}
	root.AddCommand(start)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(cfgFile, ifaceOverride string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if ifaceOverride != "" {
		cfg.Interface = ifaceOverride
	}

	pollInterval, err := cfg.PollInterval()
	if err != nil {
		return fmt.Errorf("invalid poll interval: %w", err)
	}

	// Register Prometheus metrics.
	appmetrics.Register()

	// File store adapter.
	store := filestore.New(cfg.RulesFile)

	// NOTE: On Linux with root, replace this stub with the real BPFAdapter:
	//   bpfAdapter, err := bpfmap.Load(cfg.Interface, "bpf/xdp_firewall.o")
	//   if err != nil { return fmt.Errorf("loading BPF: %w", err) }
	//   defer bpfAdapter.Close()
	//
	// For portability (macOS dev, CI), we use a no-op BPF adapter here.
	// The real adapter is in internal/adapters/bpfmap/ (linux build tag).
	bpfAdapter := &noopBPF{}

	// Core domain.
	engine := core.NewThreatEngine(bpfAdapter, store)

	// Reconcile file ↔ kernel state on startup.
	if err := engine.Reconcile(); err != nil {
		fmt.Fprintf(os.Stderr, "reconcile warning: %v\n", err)
	}

	// Metrics poller.
	poller := appmetrics.NewPoller(bpfAdapter, engine, pollInterval)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	go poller.Run(ctx)

	// HTTP server.
	mux := http.NewServeMux()
	httpadapter.New(engine).Register(mux)
	srv := &http.Server{Addr: cfg.HTTPAddr, Handler: mux}

	go func() {
		fmt.Printf("xdp-fw started — interface: %s, HTTP: %s\n", cfg.Interface, cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "HTTP server error: %v\n", err)
		}
	}()

	<-ctx.Done()
	fmt.Println("Shutting down...")
	srv.Shutdown(context.Background()) //nolint:errcheck
	return nil
}

// noopBPF is a no-op BPFMapPort + CounterReader for non-Linux builds and testing.
type noopBPF struct {
	rules map[string]struct{}
}

func (n *noopBPF) Insert(cidr string) error {
	if n.rules == nil {
		n.rules = make(map[string]struct{})
	}
	n.rules[cidr] = struct{}{}
	return nil
}
func (n *noopBPF) Delete(cidr string) error {
	delete(n.rules, cidr)
	return nil
}
func (n *noopBPF) List() ([]string, error) {
	out := make([]string, 0, len(n.rules))
	for k := range n.rules {
		out = append(out, k)
	}
	return out, nil
}
func (n *noopBPF) ReadCounters() (core.Counters, error) {
	return core.Counters{}, nil
}
