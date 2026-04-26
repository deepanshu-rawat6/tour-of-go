package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tour-of-go/k8s-event-sink/internal/core"
)

func TestSlackAlerter_PostsPayload(t *testing.T) {
	var received []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		received = buf.Bytes()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a := New(srv.URL)
	event := core.Event{
		ID: "e1", Namespace: "default", Pod: "api-pod",
		Reason: "OOMKilled", Message: "container killed", Severity: "critical",
		Count: 5, LastSeen: time.Now(),
	}
	if err := a.Notify(context.Background(), event); err != nil {
		t.Fatal(err)
	}

	var payload map[string]string
	json.Unmarshal(received, &payload)
	if !strings.Contains(payload["text"], "OOMKilled") {
		t.Errorf("expected OOMKilled in payload: %s", payload["text"])
	}
	if !strings.Contains(payload["text"], "🔴") {
		t.Errorf("expected critical emoji in payload: %s", payload["text"])
	}
	if !strings.Contains(payload["text"], "seen 5 times") {
		t.Errorf("expected count in payload: %s", payload["text"])
	}
}
