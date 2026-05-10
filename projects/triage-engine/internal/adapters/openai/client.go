package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"tour_of_go/projects/triage-engine/internal/domain"
)

type Client struct {
	httpClient     *http.Client
	apiKey         string
	model          string
	embeddingModel string
	baseURL        string
}

func NewClient(apiKey, model, embeddingModel string) *Client {
	return &Client{
		httpClient:     &http.Client{},
		apiKey:         apiKey,
		model:          model,
		embeddingModel: embeddingModel,
		baseURL:        "https://api.openai.com",
	}
}

// SetBaseURL overrides the API base URL — used in tests to point at an httptest server.
func (c *Client) SetBaseURL(url string) { c.baseURL = url }

// chatRequest / chatResponse are minimal OpenAI chat completion shapes.
type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

func (c *Client) chat(ctx context.Context, system, user string) (string, error) {
	body, _ := json.Marshal(chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("openai chat: status %d", resp.StatusCode)
	}
	var cr chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return "", err
	}
	if len(cr.Choices) == 0 {
		return "", fmt.Errorf("openai chat: no choices returned")
	}
	return strings.TrimSpace(cr.Choices[0].Message.Content), nil
}

func (c *Client) Categorize(ctx context.Context, ticket domain.TicketData) (string, error) {
	system := "Categorize the support ticket into exactly one of: build_failure, deployment_issue, access_request, performance, unknown. Reply with only the category."
	user := ticket.Summary + "\n" + ticket.Description
	return c.chat(ctx, system, user)
}

func (c *Client) DraftResponse(ctx context.Context, ticket domain.TicketData, runbook []string, diagnostic string) (string, error) {
	system := fmt.Sprintf(
		"You are a support engineer. Use the runbook context and diagnostic result to draft a helpful response.\n\nRunbook:\n%s\n\nDiagnostic:\n%s",
		strings.Join(runbook, "\n---\n"), diagnostic,
	)
	user := fmt.Sprintf("Ticket: %s\n%s", ticket.Summary, ticket.Description)
	return c.chat(ctx, system, user)
}

// embedRequest / embedResponse are minimal OpenAI embeddings shapes.
type embedRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}
type embedResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

func (c *Client) Embed(ctx context.Context, text string) ([]float32, error) {
	body, _ := json.Marshal(embedRequest{Model: c.embeddingModel, Input: text})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai embed: status %d", resp.StatusCode)
	}
	var er embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&er); err != nil {
		return nil, err
	}
	if len(er.Data) == 0 {
		return nil, fmt.Errorf("openai embed: no data returned")
	}
	return er.Data[0].Embedding, nil
}
