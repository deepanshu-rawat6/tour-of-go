package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/tour-of-go/raft-kv-store/internal/kv"
	"github.com/tour-of-go/raft-kv-store/internal/raft"
)

// RaftNode is the interface the API uses to interact with the Raft layer.
type RaftNode interface {
	Propose(ctx context.Context, command []byte, transport raft.Transport) (string, error)
	State() (role raft.Role, term uint64, leaderID string, commitIndex uint64)
	ID() string
}

// Handler is the HTTP handler for the KV API.
type Handler struct {
	node      RaftNode
	store     *kv.Store
	transport raft.Transport
	peers     map[string]string // nodeID → HTTP address
}

func New(node RaftNode, store *kv.Store, transport raft.Transport, peers map[string]string) *Handler {
	return &Handler{node: node, store: store, transport: transport, peers: peers}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/kv/", h.handleKV)
	mux.HandleFunc("/status", h.handleStatus)
}

func (h *Handler) handleKV(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path[len("/kv/"):]
	if key == "" {
		http.Error(w, "key required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		val, ok := h.store.Get(key)
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		fmt.Fprint(w, val)

	case http.MethodPut:
		h.handleWrite(w, r, "put", key)

	case http.MethodDelete:
		h.handleWrite(w, r, "delete", key)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleWrite(w http.ResponseWriter, r *http.Request, op, key string) {
	role, _, leaderID, _ := h.node.State()
	if role != raft.Leader {
		// Redirect to leader
		leaderHTTP := h.peers[leaderID]
		if leaderHTTP == "" {
			http.Error(w, "no leader known", http.StatusServiceUnavailable)
			return
		}
		http.Redirect(w, r, "http://"+leaderHTTP+r.URL.Path, http.StatusTemporaryRedirect)
		return
	}

	var value string
	if op == "put" {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "reading body", http.StatusBadRequest)
			return
		}
		value = string(body)
	}

	cmd, _ := json.Marshal(kv.Command{Op: op, Key: key, Value: value})
	result, err := h.node.Propose(r.Context(), cmd, h.transport)
	if err != nil {
		var notLeader *raft.NotLeaderError
		if errors.As(err, &notLeader) {
			leaderHTTP := h.peers[notLeader.LeaderID]
			http.Redirect(w, r, "http://"+leaderHTTP+r.URL.Path, http.StatusTemporaryRedirect)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprint(w, result)
}

func (h *Handler) handleStatus(w http.ResponseWriter, r *http.Request) {
	role, term, leaderID, commitIndex := h.node.State()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"node_id":      h.node.ID(),
		"role":         role.String(),
		"term":         term,
		"leader_id":    leaderID,
		"commit_index": commitIndex,
	})
}
