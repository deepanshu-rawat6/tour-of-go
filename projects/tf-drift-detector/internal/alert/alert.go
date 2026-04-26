package alert

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/tour-of-go/tf-drift-detector/internal/poller"
)

// Alerter sends drift notifications.
type Alerter interface {
	Send(ctx context.Context, newDrift, resolved []poller.DriftResult) error
}

// MultiAlerter fans out to all configured alerters.
type MultiAlerter struct{ alerters []Alerter }

func NewMulti(alerters ...Alerter) *MultiAlerter { return &MultiAlerter{alerters: alerters} }

func (m *MultiAlerter) Send(ctx context.Context, newDrift, resolved []poller.DriftResult) error {
	for _, a := range m.alerters {
		if err := a.Send(ctx, newDrift, resolved); err != nil {
			fmt.Fprintf(os.Stderr, "alert error: %v\n", err)
		}
	}
	return nil
}

// --- Stdout ---

// StdoutAlerter prints drift as JSON to stdout.
type StdoutAlerter struct{}

func (StdoutAlerter) Send(_ context.Context, newDrift, resolved []poller.DriftResult) error {
	payload := map[string]interface{}{
		"new_drift": newDrift,
		"resolved":  resolved,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

// --- Slack ---

// SlackAlerter sends drift to a Slack incoming webhook.
type SlackAlerter struct{ WebhookURL string }

func (s SlackAlerter) Send(ctx context.Context, newDrift, resolved []poller.DriftResult) error {
	if len(newDrift) == 0 && len(resolved) == 0 {
		return nil
	}
	text := formatSlackText(newDrift, resolved)
	payload := map[string]string{"text": text}
	return postJSON(ctx, s.WebhookURL, payload)
}

func formatSlackText(newDrift, resolved []poller.DriftResult) string {
	var sb strings.Builder
	if len(newDrift) > 0 {
		sb.WriteString(fmt.Sprintf("🚨 *Terraform Drift Detected* — %d resource(s)\n", len(newDrift)))
		for _, r := range newDrift {
			sb.WriteString(fmt.Sprintf("• `%s` / `%s`\n", r.Resource.Type, r.Resource.ID))
			for _, f := range r.Fields {
				sb.WriteString(fmt.Sprintf("  - `%s`: `%v` → `%v`\n", f.Path, f.Expected, f.Actual))
			}
		}
	}
	if len(resolved) > 0 {
		sb.WriteString(fmt.Sprintf("✅ *Drift Resolved* — %d resource(s)\n", len(resolved)))
		for _, r := range resolved {
			sb.WriteString(fmt.Sprintf("• `%s` / `%s`\n", r.Resource.Type, r.Resource.ID))
		}
	}
	return sb.String()
}

// --- Discord ---

// DiscordAlerter sends drift to a Discord webhook.
type DiscordAlerter struct{ WebhookURL string }

func (d DiscordAlerter) Send(ctx context.Context, newDrift, resolved []poller.DriftResult) error {
	if len(newDrift) == 0 && len(resolved) == 0 {
		return nil
	}
	// Discord uses "content" field for plain text webhooks
	payload := map[string]string{"content": formatSlackText(newDrift, resolved)}
	return postJSON(ctx, d.WebhookURL, payload)
}

// --- HTTP helper ---

func postJSON(ctx context.Context, url string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
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
		return fmt.Errorf("webhook returned %s", resp.Status)
	}
	return nil
}
