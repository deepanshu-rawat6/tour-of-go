package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/tour-of-go/k8s-event-sink/internal/core"
)

var severityEmoji = map[string]string{
	"critical": "🔴",
	"warning":  "🟡",
}

// Alerter implements core.AlerterPort for Slack webhooks.
type Alerter struct {
	webhookURL string
}

func New(webhookURL string) *Alerter { return &Alerter{webhookURL: webhookURL} }

func (a *Alerter) Notify(ctx context.Context, event core.Event) error {
	emoji := severityEmoji[event.Severity]
	if emoji == "" {
		emoji = "⚪"
	}
	countStr := ""
	if event.Count > 1 {
		countStr = fmt.Sprintf(" _(seen %d times)_", event.Count)
	}
	text := fmt.Sprintf("%s *%s* | `%s/%s`%s\n>%s",
		emoji, event.Reason, event.Namespace, event.Pod, countStr, event.Message)

	payload := map[string]string{"text": text}
	data, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.webhookURL, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("slack webhook returned %s", resp.Status)
	}
	return nil
}
