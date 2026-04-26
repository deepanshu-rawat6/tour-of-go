package httpadapter

import (
	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	appmetrics "github.com/tour-of-go/xdp-firewall/internal/metrics"
)

// Engine is the interface the HTTP adapter uses to interact with the core.
type Engine interface {
	BlockCIDR(cidr string) error
	UnblockCIDR(cidr string) error
	BatchBlock(cidrs []string) error
	ImportList(cidrs []string) error
	ListRules() []string
}

// CounterSource provides live stats for the /stats endpoint.
type CounterSource interface {
	PacketsTotal() float64
	DropsTotal() float64
}

// Handler is the HTTP inbound adapter.
type Handler struct {
	engine Engine
}

func New(engine Engine) *Handler {
	return &Handler{engine: engine}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/blacklist", h.handleBlacklist)
	mux.HandleFunc("/api/blacklist/batch", h.handleBatch)
	mux.HandleFunc("/api/blacklist/import", h.handleImport)
	mux.HandleFunc("/stats", h.handleStats)
	mux.Handle("/metrics", promhttp.Handler())
}

// POST /api/blacklist  {"cidr": "10.0.0.0/8"}
// DELETE /api/blacklist {"cidr": "10.0.0.0/8"}
// GET /api/blacklist
func (h *Handler) handleBlacklist(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rules := h.engine.ListRules()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"rules": rules, "count": len(rules)})

	case http.MethodPost:
		var req struct {
			CIDR string `json:"cidr"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.CIDR == "" {
			http.Error(w, "body must be {\"cidr\": \"...\"}", http.StatusBadRequest)
			return
		}
		if err := h.engine.BlockCIDR(req.CIDR); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)

	case http.MethodDelete:
		var req struct {
			CIDR string `json:"cidr"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.CIDR == "" {
			http.Error(w, "body must be {\"cidr\": \"...\"}", http.StatusBadRequest)
			return
		}
		if err := h.engine.UnblockCIDR(req.CIDR); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// POST /api/blacklist/batch  {"cidrs": ["10.0.0.0/8", "192.168.0.0/16"]}
func (h *Handler) handleBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		CIDRs []string `json:"cidrs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.CIDRs) == 0 {
		http.Error(w, "body must be {\"cidrs\": [...]}", http.StatusBadRequest)
		return
	}
	if err := h.engine.BatchBlock(req.CIDRs); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// POST /api/blacklist/import
// Accepts: multipart file upload (field "file") OR JSON {"cidrs": [...]}
// File format: one CIDR per line (plain text or JSON array)
func (h *Handler) handleImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var cidrs []string

	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart/form-data") {
		r.ParseMultipartForm(10 << 20) // 10 MB
		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "missing file field", http.StatusBadRequest)
			return
		}
		defer file.Close()
		cidrs = parseCIDRLines(file)
	} else {
		var req struct {
			CIDRs []string `json:"cidrs"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		cidrs = req.CIDRs
	}

	if len(cidrs) == 0 {
		http.Error(w, "no CIDRs provided", http.StatusBadRequest)
		return
	}
	if err := h.engine.ImportList(cidrs); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]int{"imported": len(cidrs)})
}

// GET /stats
func (h *Handler) handleStats(w http.ResponseWriter, r *http.Request) {
	rules := h.engine.ListRules()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"blacklist_size": len(rules),
		"rules":          rules,
		// Live counter values come from Prometheus gatherer
		"packets_total": appmetrics.PacketsTotal,
		"drops_total":   appmetrics.DropsTotal,
	})
}

// parseCIDRLines reads one CIDR per line from r, skipping blanks and comments.
func parseCIDRLines(r io.Reader) []string {
	var cidrs []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		cidrs = append(cidrs, line)
	}
	return cidrs
}
