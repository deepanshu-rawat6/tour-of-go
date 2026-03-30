package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	natsclient "github.com/nats-io/nats.go"

	natsadapter "tour_of_go/projects/event-driven-pipeline/internal/adapters/nats"
	"tour_of_go/projects/event-driven-pipeline/internal/domain"
)

func main() {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = natsclient.DefaultURL
	}

	nc, err := natsclient.Connect(natsURL)
	if err != nil {
		slog.Error("nats connect failed", "error", err)
		os.Exit(1)
	}
	defer nc.Close()

	producer, err := natsadapter.NewProducer(nc, "events.orders")
	if err != nil {
		slog.Error("producer init failed", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	for i := 1; i <= 10; i++ {
		event := &domain.Event{
			ID:             fmt.Sprintf("evt-%d", i),
			Type:           "order.created",
			IdempotencyKey: fmt.Sprintf("order-%d", i),
			Payload:        map[string]any{"orderID": i, "amount": i * 100},
			Status:         domain.EventPending,
			CreatedAt:      time.Now(),
		}
		if err := producer.Publish(ctx, event); err != nil {
			slog.Error("publish failed", "eventID", event.ID, "error", err)
		} else {
			slog.Info("published", "eventID", event.ID)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Duplicate — same idempotency key, should be skipped by consumer
	dup := &domain.Event{
		ID: "evt-1-dup", Type: "order.created",
		IdempotencyKey: "order-1",
		Payload:        map[string]any{"orderID": 1},
		Status:         domain.EventPending, CreatedAt: time.Now(),
	}
	producer.Publish(ctx, dup)
	slog.Info("sent duplicate (will be deduplicated)", "eventID", dup.ID)
}
