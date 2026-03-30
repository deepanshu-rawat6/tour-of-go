// Package handlers implements the HTTP handlers for the dashboard.
package handlers

import (
	"html/template"
	"log/slog"
	"net/http"
	"strconv"

	"tour_of_go/projects/realtime-dashboard/internal/scheduler"
	"tour_of_go/projects/realtime-dashboard/internal/ws"
)

// Handler holds all dependencies for the HTTP handlers.
type Handler struct {
	tmpl      *template.Template
	scheduler *scheduler.Client
	hub       *ws.Hub
	log       *slog.Logger
}

func New(tmpl *template.Template, sched *scheduler.Client, hub *ws.Hub) *Handler {
	return &Handler{tmpl: tmpl, scheduler: sched, hub: hub, log: slog.Default()}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /", h.index)
	mux.HandleFunc("GET /partials/concurrency", h.concurrencyPartial)
	mux.HandleFunc("POST /jobs", h.submitJob)
	mux.HandleFunc("GET /ws", h.websocket)
}

func (h *Handler) index(w http.ResponseWriter, r *http.Request) {
	keys, _ := h.scheduler.GetConcurrencyKeys(r.Context())
	h.tmpl.ExecuteTemplate(w, "index.html", map[string]any{
		"ConcurrencyKeys": keys,
		"SchedulerURL":    h.scheduler,
	})
}

func (h *Handler) concurrencyPartial(w http.ResponseWriter, r *http.Request) {
	keys, err := h.scheduler.GetConcurrencyKeys(r.Context())
	if err != nil {
		keys = map[string]int64{"error": -1}
	}
	h.tmpl.ExecuteTemplate(w, "concurrency.html", keys)
}

func (h *Handler) submitJob(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	jobName := r.FormValue("jobName")
	tenant, _ := strconv.Atoi(r.FormValue("tenant"))
	priority, _ := strconv.Atoi(r.FormValue("priority"))
	if priority == 0 {
		priority = 5
	}
	if err := h.scheduler.SubmitJob(r.Context(), jobName, tenant, priority); err != nil {
		h.log.Error("submit job failed", "error", err)
		http.Error(w, "Failed to submit job: "+err.Error(), http.StatusBadGateway)
		return
	}
	// HTMX: return updated concurrency partial after submit
	h.concurrencyPartial(w, r)
}

func (h *Handler) websocket(w http.ResponseWriter, r *http.Request) {
	ws.ServeWS(h.hub, w, r)
}
