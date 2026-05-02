// Package broker provides RabbitMQ producer and consumer implementations.
package broker

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"tour_of_go/projects/rabbitmq-worker/internal/domain"
)

const (
	Exchange    = "tasks"
	Queue       = "tasks.queue"
	DLXExchange = "tasks.dlx"
	DLQueue     = "tasks.dlq"
	RoutingKey  = "task"
)

// Producer publishes tasks to a durable RabbitMQ exchange.
type Producer struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

// NewProducer connects to RabbitMQ, declares the exchange/queue topology, and returns a Producer.
func NewProducer(url string) (*Producer, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("channel: %w", err)
	}
	if err := declareTopology(ch); err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}
	return &Producer{conn: conn, ch: ch}, nil
}

// Publish serialises a Task and sends it as a persistent message.
func (p *Producer) Publish(ctx context.Context, task domain.Task) error {
	body, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("marshal task: %w", err)
	}
	return p.ch.PublishWithContext(ctx, Exchange, RoutingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent, // survives broker restart
		Body:         body,
	})
}

// Close releases the channel and connection.
func (p *Producer) Close() {
	p.ch.Close()
	p.conn.Close()
}

// declareTopology declares the full exchange/queue/DLX topology.
// Idempotent — safe to call on every startup.
func declareTopology(ch *amqp.Channel) error {
	// Main exchange
	if err := ch.ExchangeDeclare(Exchange, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare exchange: %w", err)
	}
	// Dead-letter exchange
	if err := ch.ExchangeDeclare(DLXExchange, "fanout", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare DLX: %w", err)
	}
	// Main queue — messages that exhaust retries are routed to DLX
	_, err := ch.QueueDeclare(Queue, true, false, false, false, amqp.Table{
		"x-dead-letter-exchange": DLXExchange,
	})
	if err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}
	if err := ch.QueueBind(Queue, RoutingKey, Exchange, false, nil); err != nil {
		return fmt.Errorf("bind queue: %w", err)
	}
	// Dead-letter queue
	if _, err := ch.QueueDeclare(DLQueue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare DLQ: %w", err)
	}
	if err := ch.QueueBind(DLQueue, "", DLXExchange, false, nil); err != nil {
		return fmt.Errorf("bind DLQ: %w", err)
	}
	return nil
}
