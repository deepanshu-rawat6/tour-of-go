// FS-17: Chat with Memory
// Builds on FS-16 by maintaining conversation history across turns.
// Run: export OPENAI_API_KEY=sk-... && go run .
package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

const (
	maxContextTokens = 4000  // stop adding old messages when history exceeds this
	model            = openai.GPT4oMini
	avgCharsPerToken = 4     // rough estimate: 1 token ≈ 4 chars
)

func main() {
	client := openai.NewClient(mustGetenv("OPENAI_API_KEY"))

	// Conversation history — grows with each turn
	history := []openai.ChatCompletionMessage{
		{
			Role: openai.ChatMessageRoleSystem,
			Content: "You are a helpful assistant. " +
				"Answer concisely. If you don't know something, say so.",
		},
	}

	fmt.Println("Chat with Memory (FS-17)")
	fmt.Println("Commands: /clear (reset), /system <text> (change persona), /history, /tokens")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("You: ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// Handle slash commands
		switch {
		case input == "/clear":
			system := history[0] // keep system prompt
			history = []openai.ChatCompletionMessage{system}
			fmt.Println("[History cleared]")
			continue

		case strings.HasPrefix(input, "/system "):
			newSystem := strings.TrimPrefix(input, "/system ")
			history[0] = openai.ChatCompletionMessage{
				Role: openai.ChatMessageRoleSystem, Content: newSystem,
			}
			fmt.Println("[System prompt updated]")
			continue

		case input == "/history":
			for i, msg := range history {
				fmt.Printf("[%d] %s: %s\n", i, msg.Role, truncate(msg.Content, 80))
			}
			continue

		case input == "/tokens":
			fmt.Printf("[Estimated tokens in context: ~%d / %d]\n",
				estimateTokens(history), maxContextTokens)
			continue
		}

		// Add user message
		history = append(history, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: input,
		})

		// Trim history if approaching context limit
		history = trimHistory(history, maxContextTokens)

		// Stream response
		fmt.Print("Assistant: ")
		reply, err := streamWithHistory(client, history)
		fmt.Println()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			// Remove the user message we just added so history stays clean
			history = history[:len(history)-1]
			continue
		}

		// Append assistant reply to history — THIS is what creates "memory"
		history = append(history, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: reply,
		})

		fmt.Printf("[~%d tokens in context]\n\n", estimateTokens(history))
	}
}

// streamWithHistory sends the full history and streams the response.
// Returns the complete assistant reply text.
func streamWithHistory(client *openai.Client, history []openai.ChatCompletionMessage) (string, error) {
	stream, err := client.CreateChatCompletionStream(context.Background(),
		openai.ChatCompletionRequest{
			Model:       model,
			Messages:    history, // ← the entire conversation every time
			Temperature: 0.7,
			MaxTokens:   500,
			Stream:      true,
		})
	if err != nil {
		return "", err
	}
	defer stream.Close()

	var sb strings.Builder
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return sb.String(), err
		}
		token := resp.Choices[0].Delta.Content
		fmt.Print(token)
		sb.WriteString(token)
	}
	return sb.String(), nil
}

// trimHistory removes the oldest non-system messages when context is too large.
// Always preserves: system prompt (index 0) + last N messages.
func trimHistory(history []openai.ChatCompletionMessage, maxTokens int) []openai.ChatCompletionMessage {
	for estimateTokens(history) > maxTokens && len(history) > 3 {
		// Remove message at index 1 (oldest non-system message)
		history = append(history[:1], history[2:]...)
	}
	return history
}

// estimateTokens gives a rough token count for the whole history.
func estimateTokens(history []openai.ChatCompletionMessage) int {
	total := 0
	for _, msg := range history {
		total += len(msg.Content) / avgCharsPerToken
		total += 4 // per-message overhead (role tokens)
	}
	return total
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func mustGetenv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		fmt.Fprintf(os.Stderr, "%s not set\n", key)
		os.Exit(1)
	}
	return v
}
