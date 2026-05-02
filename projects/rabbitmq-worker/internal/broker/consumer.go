package broker

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Consumer receives messages from RabbitMQ with manual ack and QoS prefetch.
type Consumer struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

// NewConsumer connects, declares topology, and sets QoS prefetch.
func NewConsumer(url string, prefetch int) (*Consumer, error) {
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
	// QoS: process at most `prefetch` unacked messages at a time
	if err := ch.Qos(prefetch, 0, false); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("qos: %w", err)
	}
	return &Consumer{conn: conn, ch: ch}, nil
}

// Consume returns a channel of deliveries. autoAck=false for manual acknowledgement.
func (c *Consumer) Consume() (<-chan amqp.Delivery, error) {
	return c.ch.Consume(Queue, "", false, false, false, false, nil)
}

// Close releases the channel and connection.
func (c *Consumer) Close() {
	c.ch.Close()
	c.conn.Close()
}
