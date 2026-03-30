// Package api implements the HTTP REST API using only net/http.
//
// Design decision (see docs/adr/009-stdlib-http-over-gin.md):
// Go 1.22's enhanced ServeMux supports method+path routing natively.
// No framework needed for this surface area.
package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"tour_of_go/projects/distributed-scheduler/internal/domain"
	"tour_of_go/projects/distributed-scheduler/internal/ports"
)

type Handler struct {
	jobRepo    ports.JobRepository
	triggerFn  func(ctx context.Context, jobName string) bool
	poolSnap   func() map[string]int64
	log        *slog.Logger
}

func NewHandler(jobRepo ports.JobRepository, triggerFn func(ctx context.Context, jobName string) bool, poolSnap func() map[string]int64) *Handler {
	return &Handler{jobRepo: jobRepo, triggerFn: triggerFn, poolSnap: poolSnap, log: slog.Default()}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /jobs", h.submitJob)
	mux.HandleFunc("POST /jobs/{id}/retry", h.retryJob)
	mux.HandleFunc("POST /jobs/{id}/cancel", h.cancelJob)
	mux.HandleFunc("GET /concurrency/keys", h.concurrencyKeys)
	mux.HandleFunc("GET /health", h.health)
}

type submitRequest struct {
	JobName            string            `json:"jobName"`
	Tenant             int               `json:"tenant"`
	Priority           int               `json:"priority"`
	Payload            map[string]any    `json:"payload"`
	ConcurrencyControl map[string]string `json:"concurrencyControl"`
}

func (h *Handler) submitJob(w http.ResponseWriter, r *http.Request) {
	var req submitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	job := &domain.Job{
		JobName:            req.JobName,
		Tenant:             req.Tenant,
		Priority:           req.Priority,
		Payload:            req.Payload,
		ConcurrencyControl: req.ConcurrencyControl,
		Status:             domain.StatusWaiting,
	}
	if err := h.jobRepo.Save(r.Context(), job); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.triggerFn(r.Context(), job.JobName)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"id": job.ID, "status": job.Status})
}

func (h *Handler) retryJob(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	job, err := h.jobRepo.FindByID(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err := job.Transition(domain.StatusRetry); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if err := h.jobRepo.Save(r.Context(), job); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.triggerFn(r.Context(), job.JobName)
	json.NewEncoder(w).Encode(map[string]any{"id": job.ID, "status": job.Status})
}

func (h *Handler) cancelJob(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	job, err := h.jobRepo.FindByID(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err := job.Transition(domain.StatusCancelled); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	h.jobRepo.Save(r.Context(), job)
	json.NewEncoder(w).Encode(map[string]any{"id": job.ID, "status": job.Status})
}

func (h *Handler) concurrencyKeys(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.poolSnap())
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
