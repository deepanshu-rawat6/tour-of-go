// Package handlers implements HTTP handlers for the platform console.
// Uses html/template directly (templ requires a code generation step;
// we use stdlib templates here so the project builds without tooling).
package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"

	"tour_of_go/projects/platform-console/internal/k8s"
)

// Handler holds all dependencies.
type Handler struct {
	tmpl    *template.Template
	k8s     *k8s.Client
	watcher *k8s.Watcher
	log     *slog.Logger
}

func New(tmpl *template.Template, client *k8s.Client, watcher *k8s.Watcher) *Handler {
	return &Handler{tmpl: tmpl, k8s: client, watcher: watcher, log: slog.Default()}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /", h.listGreetings)
	mux.HandleFunc("GET /greetings/new", h.newGreetingForm)
	mux.HandleFunc("POST /greetings", h.createGreeting)
	mux.HandleFunc("POST /greetings/{name}/delete", h.deleteGreeting)
	mux.HandleFunc("GET /events", h.sseStream)
}

func (h *Handler) listGreetings(w http.ResponseWriter, r *http.Request) {
	greetings, err := h.k8s.List(r.Context())
	if err != nil {
		h.log.Error("list greetings failed", "error", err)
		greetings = []k8s.Greeting{}
	}
	h.tmpl.ExecuteTemplate(w, "index.html", map[string]any{"Greetings": greetings})
}

func (h *Handler) newGreetingForm(w http.ResponseWriter, r *http.Request) {
	h.tmpl.ExecuteTemplate(w, "new.html", nil)
}

func (h *Handler) createGreeting(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	name := r.FormValue("name")
	message := r.FormValue("message")
	if err := h.k8s.Create(r.Context(), name, message); err != nil {
		http.Error(w, "Create failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handler) deleteGreeting(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := h.k8s.Delete(r.Context(), name); err != nil {
		http.Error(w, "Delete failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// sseStream streams Greeting watch events as Server-Sent Events.
// The browser uses EventSource to receive these without polling.
func (h *Handler) sseStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	ch := h.watcher.Subscribe()
	defer h.watcher.Unsubscribe(ch)

	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}
