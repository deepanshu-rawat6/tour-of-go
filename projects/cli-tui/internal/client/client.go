// Package client provides an HTTP client for the distributed-scheduler API.
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Job is a simplified view of a scheduler job.
type Job struct {
	ID      int64  `json:"id"`
	JobName string `json:"jobName"`
	Tenant  int    `json:"tenant"`
	Status  string `json:"status"`
}

// Client calls the distributed-scheduler REST API.
type Client struct {
	base string
	http *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		base: strings.TrimRight(baseURL, "/"),
		http: &http.Client{Timeout: 3 * time.Second},
	}
}

// GetConcurrencyKeys returns the current concurrency pool snapshot.
func (c *Client) GetConcurrencyKeys(ctx context.Context) (map[string]int64, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", c.base+"/concurrency/keys", nil)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result map[string]int64
	return result, json.NewDecoder(resp.Body).Decode(&result)
}

// SubmitJob posts a new job to the scheduler.
func (c *Client) SubmitJob(ctx context.Context, jobName string, tenant, priority int) error {
	body := fmt.Sprintf(`{"jobName":%q,"tenant":%d,"priority":%d,"payload":{}}`, jobName, tenant, priority)
	req, _ := http.NewRequestWithContext(ctx, "POST", c.base+"/jobs", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("scheduler returned %d", resp.StatusCode)
	}
	return nil
}
