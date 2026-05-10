package diagnostic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
}

func NewClient(baseURL string) *Client {
	return &Client{httpClient: &http.Client{}, baseURL: baseURL}
}

type buildStatus struct {
	Status     string `json:"status"`
	BuildNumber int   `json:"build_number"`
	DurationMs  int   `json:"duration_ms"`
}

func (c *Client) CheckBuildStatus(ctx context.Context, userID string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/builds/%s/latest", c.baseURL, userID), nil)
	if err != nil {
		return "", err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("diagnostic API: status %d", resp.StatusCode)
	}
	var bs buildStatus
	if err := json.NewDecoder(resp.Body).Decode(&bs); err != nil {
		return "", err
	}
	return fmt.Sprintf("build #%d: %s (duration: %dms)", bs.BuildNumber, bs.Status, bs.DurationMs), nil
}
