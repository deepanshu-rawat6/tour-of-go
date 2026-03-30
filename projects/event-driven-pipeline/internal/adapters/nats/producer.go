// Package nats implements NATS JetStream producer and consumer.
package nats

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"tour_of_go/projects/event-driven-pipeline/internal/domain"
)

// Producer publishes events to a NATS JetStream subject.
type Producer struct {
	js      jetstream.JetStream
	subject string
}

func NewProducer(nc *nats.Conn, subject string) (*Producer, error) {
	js, err := jetstream.New(nc)
	if err != nil {
		return nil, fmt.Errorf("jetstream: %w", err)
	}
	return &Producer{js: js, subject: subject}, nil
}

func (p *Producer) Publish(ctx context.Context, event *domain.Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	// Inject OTel trace ID as NATS header for cross-service trace propagation
	msg := &nats.Msg{
		Subject: p.subject,
		Data:    data,
		Header:  nats.Header{},
	}
	if event.TraceID != "" {
		msg.Header.Set("traceparent", event.TraceID)
	}
	_, err = p.js.PublishMsg(ctx, msg)
	return err
}
