package alert

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tour-of-go/tf-drift-detector/internal/diff"
	"github.com/tour-of-go/tf-drift-detector/internal/poller"
	"github.com/tour-of-go/tf-drift-detector/internal/state"
)

func driftResult(id string) poller.DriftResult {
	return poller.DriftResult{
		Resource: state.ManagedResource{Type: "aws_instance", ID: id},
		Drifted:  true,
		Fields:   []diff.DriftField{{Path: "instance_type", Expected: "t3.micro", Actual: "t3.large"}},
	}
}

func TestStdoutAlerter(t *testing.T) {
	// Capture stdout via a pipe isn't straightforward; test the format function directly
	text := formatSlackText([]poller.DriftResult{driftResult("i-1")}, nil)
	if !strings.Contains(text, "aws_instance") {
		t.Errorf("expected resource type in output: %s", text)
	}
	if !strings.Contains(text, "instance_type") {
		t.Errorf("expected field name in output: %s", text)
	}
}

func TestSlackAlerter_PostsCorrectPayload(t *testing.T) {
	var received []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		received = buf.Bytes()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a := SlackAlerter{WebhookURL: srv.URL}
	err := a.Send(context.Background(), []poller.DriftResult{driftResult("i-1")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]string
	json.Unmarshal(received, &payload)
	if !strings.Contains(payload["text"], "aws_instance") {
		t.Errorf("expected resource type in Slack payload: %s", payload["text"])
	}
}

func TestDiscordAlerter_PostsCorrectPayload(t *testing.T) {
	var received []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		received = buf.Bytes()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a := DiscordAlerter{WebhookURL: srv.URL}
	err := a.Send(context.Background(), []poller.DriftResult{driftResult("i-1")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]string
	json.Unmarshal(received, &payload)
	if _, ok := payload["content"]; !ok {
		t.Error("expected 'content' field in Discord payload")
	}
}

func TestMultiAlerter_NoSendOnEmpty(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a := NewMulti(SlackAlerter{WebhookURL: srv.URL})
	a.Send(context.Background(), nil, nil) // empty — should not POST
	if called {
		t.Error("expected no HTTP call for empty drift")
	}
}
