// Command console serves the platform console — a K8s Greeting resource browser.
// Connects to the k8s-controller project via client-go.
package main

import (
	"context"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tour_of_go/projects/platform-console/internal/handlers"
	"tour_of_go/projects/platform-console/internal/k8s"
)

func main() {
	namespace := os.Getenv("NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":3001"
	}

	k8sClient, err := k8s.NewClient(namespace)
	if err != nil {
		slog.Error("k8s client init failed", "error", err)
		slog.Info("tip: ensure kubeconfig is set and k8s-controller CRD is installed")
		os.Exit(1)
	}

	watcher := k8s.NewWatcher(k8sClient)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	watcher.Start(ctx)

	tmpl := template.Must(template.ParseGlob("web/templates/*.html"))
	h := handlers.New(tmpl, k8sClient, watcher)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	srv := &http.Server{Addr: addr, Handler: mux, ReadTimeout: 15 * time.Second}

	go func() {
		slog.Info("platform console starting", "addr", addr, "namespace", namespace)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	<-sig
	cancel()
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutCancel()
	srv.Shutdown(shutCtx)
}
