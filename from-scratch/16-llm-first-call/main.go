// FS-16: Your First LLM Call
// Run: export OPENAI_API_KEY=sk-... && go run .
// Starts an HTTP server and an interactive CLI.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	openai "github.com/sashabaranov/go-openai"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "OPENAI_API_KEY not set")
		os.Exit(1)
	}
	client := openai.NewClient(apiKey)

	// Choose mode: CLI or HTTP server
	if len(os.Args) > 1 && os.Args[1] == "server" {
		runServer(client)
	} else {
		runCLI(client)
	}
}

// ── CLI mode ──────────────────────────────────────────────────────────────

func runCLI(client *openai.Client) {
	fmt.Println("First LLM Call — type a question, Ctrl+C to exit")
	fmt.Println("Model: gpt-4o-mini  Temperature: 0  Streaming: on")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("You: ")
		if !scanner.Scan() {
			break
		}
		prompt := scanner.Text()
		if prompt == "" {
			continue
		}

		fmt.Print("Assistant: ")
		usage, err := streamResponse(client, prompt)
		fmt.Println()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			continue
		}

		// Show token usage + cost after each response
		inputCost := float64(usage.PromptTokens) * 0.000000150     // $0.150 per 1M
		outputCost := float64(usage.CompletionTokens) * 0.000000600 // $0.600 per 1M
		fmt.Printf("[tokens: %d in + %d out = %d total | cost: $%.6f]\n\n",
			usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens,
			inputCost+outputCost)
	}
}

// streamResponse sends a single prompt and prints tokens as they arrive.
// Returns token usage from the final chunk.
func streamResponse(client *openai.Client, prompt string) (openai.Usage, error) {
	stream, err := client.CreateChatCompletionStream(context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4oMini,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem,
					Content: "You are a helpful, concise assistant."},
				{Role: openai.ChatMessageRoleUser,
					Content: prompt},
			},
			Temperature: 0,
			MaxTokens:   500,
			Stream:      true,
		})
	if err != nil {
		return openai.Usage{}, err
	}
	defer stream.Close()

	var usage openai.Usage
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return usage, err
		}
		// Print each token immediately as it arrives
		fmt.Print(resp.Choices[0].Delta.Content)
		// Capture usage from last chunk (OpenAI sends it there)
		if resp.Usage != nil {
			usage = *resp.Usage
		}
	}
	return usage, nil
}

// ── HTTP server mode ──────────────────────────────────────────────────────

// POST /ask  body: {"prompt":"..."} [returns SSE stream]
// GET  /ask?q=...               [returns SSE stream]
func runServer(client *openai.Client) {
	http.HandleFunc("/ask", func(w http.ResponseWriter, r *http.Request) {
		var prompt string
		if r.Method == http.MethodPost {
			var body struct {
				Prompt string `json:"prompt"`
			}
			json.NewDecoder(r.Body).Decode(&body)
			prompt = body.Prompt
		} else {
			prompt = r.URL.Query().Get("q")
		}
		if prompt == "" {
			http.Error(w, "missing prompt", 400)
			return
		}

		// SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher := w.(http.Flusher)

		stream, err := client.CreateChatCompletionStream(r.Context(),
			openai.ChatCompletionRequest{
				Model: openai.GPT4oMini,
				Messages: []openai.ChatCompletionMessage{
					{Role: openai.ChatMessageRoleSystem, Content: "You are a helpful assistant."},
					{Role: openai.ChatMessageRoleUser, Content: prompt},
				},
				Temperature: 0,
				Stream:      true,
			})
		if err != nil {
			fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
			return
		}
		defer stream.Close()

		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
				return
			}
			token := resp.Choices[0].Delta.Content
			if token == "" {
				continue
			}
			// Each SSE event: data: "token"\n\n
			b, _ := json.Marshal(token)
			fmt.Fprintf(w, "data: %s\n\n", b)
			flusher.Flush()
		}
		fmt.Fprint(w, "event: done\ndata: {}\n\n")
		flusher.Flush()
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	fmt.Println("Server listening on :8080")
	fmt.Println("  POST /ask  body={\"prompt\":\"...\"}")
	fmt.Println("  GET  /ask?q=your+question")
	http.ListenAndServe(":8080", nil)
}
