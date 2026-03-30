// Command dashboard serves the real-time job scheduler dashboard.
// It connects to the distributed-scheduler REST API and pushes live
// concurrency updates to all connected browsers via WebSocket.
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

	"tour_of_go/projects/realtime-dashboard/internal/handlers"
	"tour_of_go/projects/realtime-dashboard/internal/scheduler"
	"tour_of_go/projects/realtime-dashboard/internal/ws"
)

func main() {
	schedulerURL := os.Getenv("SCHEDULER_URL")
	if schedulerURL == "" {
		schedulerURL = "http://localhost:8080"
	}
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":3000"
	}

	// Parse all templates (layout + pages + partials)
	tmpl := template.Must(template.ParseGlob("web/templates/*.html"))
	tmpl = template.Must(tmpl.ParseGlob("web/templates/partials/*.html"))

	hub := ws.NewHub()
	go hub.Run()

	schedClient := scheduler.NewClient(schedulerURL)
	poller := scheduler.NewPoller(schedClient, hub, 1*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	poller.Start(ctx)

	h := handlers.New(tmpl, schedClient, hub)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	srv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		slog.Info("dashboard starting", "addr", addr, "scheduler", schedulerURL)
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
