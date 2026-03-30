package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/nats-io/nats.go/jetstream"

	"tour_of_go/projects/event-driven-pipeline/internal/domain"
	"tour_of_go/projects/event-driven-pipeline/internal/pipeline"
)

// Consumer pulls events from a NATS JetStream consumer and submits them to the processor.
type Consumer struct {
	consumer  jetstream.Consumer
	processor *pipeline.Processor
	log       *slog.Logger
}

func NewConsumer(js jetstream.JetStream, stream, consumerName string, processor *pipeline.Processor) (*Consumer, error) {
	c, err := js.Consumer(context.Background(), stream, consumerName)
	if err != nil {
		return nil, fmt.Errorf("get consumer: %w", err)
	}
	return &Consumer{consumer: c, processor: processor, log: slog.Default()}, nil
}

// Start begins consuming messages. Stops when ctx is cancelled.
func (c *Consumer) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				c.log.Info("consumer stopped")
				return
			default:
				msgs, err := c.consumer.Fetch(10)
				if err != nil {
					continue
				}
				for msg := range msgs.Messages() {
					var event domain.Event
					if err := json.Unmarshal(msg.Data(), &event); err != nil {
						c.log.Error("unmarshal failed", "error", err)
						msg.Nak()
						continue
					}
					// Extract OTel trace ID from NATS header
					if tp := msg.Headers().Get("traceparent"); tp != "" {
						event.TraceID = tp
					}
					if err := c.processor.Submit(&event); err != nil {
						// Buffer full — nack so NATS redelivers (backpressure)
						c.log.Warn("processor buffer full, nacking", "eventID", event.ID)
						msg.Nak()
					} else {
						msg.Ack()
					}
				}
			}
		}
	}()
}
