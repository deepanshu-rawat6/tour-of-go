package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"

	bleveadapter "github.com/tour-of-go/k8s-event-sink/internal/adapters/bleve"
	"github.com/tour-of-go/k8s-event-sink/internal/adapters/informer"
	slackadapter "github.com/tour-of-go/k8s-event-sink/internal/adapters/slack"
	stdoutadapter "github.com/tour-of-go/k8s-event-sink/internal/adapters/stdout"
	sqliteadapter "github.com/tour-of-go/k8s-event-sink/internal/adapters/sqlite"
	"github.com/tour-of-go/k8s-event-sink/internal/config"
	"github.com/tour-of-go/k8s-event-sink/internal/core"
	"github.com/tour-of-go/k8s-event-sink/internal/metrics"
)

func main() {
	var cfgFile string
	root := &cobra.Command{
		Use:   "event-sink",
		Short: "k8s-event-sink — stream, deduplicate, and persist Kubernetes events",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cfgFile)
		},
	}
	root.Flags().StringVarP(&cfgFile, "config", "c", "config.yaml", "path to config file")
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

	metrics.Register()

	// Storage adapters.
	sqliteStore, err := sqliteadapter.New(cfg.Storage.SQLitePath)
	if err != nil {
		return fmt.Errorf("sqlite: %w", err)
	}
	defer sqliteStore.Close()

	bleveIdx, err := bleveadapter.New(cfg.Storage.BlevePath)
	if err != nil {
		return fmt.Errorf("bleve: %w", err)
	}
	defer bleveIdx.Close()

	// Alerters.
	var alerters []core.AlerterPort
	if cfg.Alerts.Slack.WebhookURL != "" {
		alerters = append(alerters, slackadapter.New(cfg.Alerts.Slack.WebhookURL))
	}
	if cfg.Alerts.Stdout || len(alerters) == 0 {
		alerters = append(alerters, stdoutadapter.New())
	}
	alerter := core.NewMultiAlerter(alerters...)

	// Core domain: filter + dedup + processor.
	filter := core.NewFilter(cfg)
	// processor is created after dedup so the callback can reference it.
	var processor *core.EventProcessor
	dedup := core.NewDedupEngine(cfg.DedupWindow(), func(ctx context.Context, event core.Event) {
		if processor != nil {
			processor.OnForward(ctx, event)
		}
	})
	processor = core.NewEventProcessor(filter, dedup, sqliteStore, bleveIdx, alerter)

	// K8s informer.
	watcher, err := informer.New(cfg.KubeConfig, processor)
	if err != nil {
		return fmt.Errorf("k8s client: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	watcher.Start(ctx, cfg.Namespaces)
	defer watcher.Stop()

	// HTTP: Prometheus + search + query.
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		if q == "" {
			http.Error(w, "missing q", http.StatusBadRequest)
			return
		}
		results, err := bleveIdx.Search(q)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, results)
	})
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		f := core.QueryFilter{
			Namespace: r.URL.Query().Get("namespace"),
			Severity:  r.URL.Query().Get("severity"),
			Limit:     100,
		}
		results, err := sqliteStore.Query(r.Context(), f)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, results)
	})

	srv := &http.Server{Addr: cfg.MetricsAddr, Handler: mux}
	go func() {
		fmt.Printf("event-sink started — namespaces: %v, addr: %s\n", cfg.Namespaces, cfg.MetricsAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "HTTP error: %v\n", err)
		}
	}()

	<-ctx.Done()
	fmt.Println("Shutting down — flushing dedup buckets...")
	dedup.Flush(context.Background())
	srv.Shutdown(context.Background()) //nolint:errcheck
	return nil
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
