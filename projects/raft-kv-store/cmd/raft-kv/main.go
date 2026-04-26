package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/tour-of-go/raft-kv-store/internal/api"
	"github.com/tour-of-go/raft-kv-store/internal/config"
	"github.com/tour-of-go/raft-kv-store/internal/kv"
	"github.com/tour-of-go/raft-kv-store/internal/raft"
	"github.com/tour-of-go/raft-kv-store/internal/transport"
	"github.com/tour-of-go/raft-kv-store/internal/wal"
)

func main() {
	var cfgFile string
	root := &cobra.Command{
		Use:   "raft-kv",
		Short: "Distributed key-value store using Raft consensus",
	}
	root.PersistentFlags().StringVarP(&cfgFile, "config", "c", "config.yaml", "path to config file")

	start := &cobra.Command{
		Use:   "start",
		Short: "Start a Raft KV node",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cfgFile)
		},
	}
	root.AddCommand(start)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(cfgFile string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	// Open WAL
	w, err := wal.Open(cfg.WALDir)
	if err != nil {
		return fmt.Errorf("WAL: %w", err)
	}
	defer w.Close()

	// KV state machine
	store := kv.New()

	// Peer IDs
	peerIDs := make([]string, len(cfg.Peers))
	peerGRPC := make(map[string]string)
	peerHTTP := make(map[string]string)
	for i, p := range cfg.Peers {
		peerIDs[i] = p.ID
		peerGRPC[p.ID] = p.GRPCAddr
		peerHTTP[p.ID] = p.HTTPAddr
	}

	// Raft node
	node, err := raft.NewNode(cfg.NodeID, peerIDs, w, store)
	if err != nil {
		return fmt.Errorf("raft node: %w", err)
	}

	// gRPC transport
	grpcClient := transport.NewClient(peerGRPC)
	defer grpcClient.Close()

	grpcServer := transport.NewServer(node)
	go func() {
		if err := grpcServer.Serve(cfg.GRPCAddr); err != nil {
			fmt.Fprintf(os.Stderr, "gRPC server error: %v\n", err)
		}
	}()
	defer grpcServer.Stop()

	// HTTP API
	mux := http.NewServeMux()
	apiHandler := api.New(node, store, grpcClient, peerHTTP)
	apiHandler.Register(mux)
	httpServer := &http.Server{Addr: cfg.HTTPAddr, Handler: mux}
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "HTTP server error: %v\n", err)
		}
	}()

	// Start Raft main loop
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	fmt.Printf("Node %s started — gRPC %s, HTTP %s\n", cfg.NodeID, cfg.GRPCAddr, cfg.HTTPAddr)
	node.Run(ctx, grpcClient, cfg.ElectionTimeoutMinMs, cfg.ElectionTimeoutMaxMs, cfg.HeartbeatMs)

	httpServer.Shutdown(context.Background()) //nolint:errcheck
	fmt.Printf("Node %s stopped.\n", cfg.NodeID)
	return nil
}
