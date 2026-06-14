// Package handler provides the HTTP API for the task scheduler.
package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"tour_of_go/projects/from-scratch/09-task-scheduler/internal/scheduler"
)

func Register(mux *http.ServeMux, s *scheduler.Scheduler) {
	mux.HandleFunc("/tasks", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			listTasks(w, s)
		case http.MethodPost:
			addTask(w, r, s)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/tasks/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/tasks/")
		s.Remove(id)
		w.WriteHeader(http.StatusNoContent)
	})
}

func listTasks(w http.ResponseWriter, s *scheduler.Scheduler) {
	tasks := s.List()
	type row struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Expr    string `json:"expr"`
		LastRun string `json:"last_run,omitempty"`
	}
	rows := make([]row, len(tasks))
	for i, t := range tasks {
		r := row{ID: t.ID, Name: t.Name, Expr: t.Expr}
		if !t.LastRun.IsZero() {
			r.LastRun = t.LastRun.Format("15:04:05")
		}
		rows[i] = r
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rows)
}

func addTask(w http.ResponseWriter, r *http.Request, s *scheduler.Scheduler) {
	var body struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Expr string `json:"expr"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if err := s.Add(body.ID, body.Name, body.Expr, func() {
		log.Printf("[scheduler] task %s (%s) fired", body.ID, body.Name)
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusCreated)
}
