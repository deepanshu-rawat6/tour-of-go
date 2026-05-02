// Package domain defines the core task type for the rabbitmq-worker.
package domain

// Task represents a unit of work published to RabbitMQ.
type Task struct {
	ID         string `json:"id"`
	Type       string `json:"type"`    // "email", "sms", "webhook"
	Payload    string `json:"payload"` // JSON-encoded task-specific data
	RetryCount int    `json:"retry_count"`
}
