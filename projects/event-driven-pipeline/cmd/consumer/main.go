// Command consumer processes events from NATS JetStream with exactly-once semantics.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/redis/go-redis/v9"

	natsadapter "tour_of_go/projects/event-driven-pipeline/internal/adapters/nats"
	redisadapter "tour_of_go/projects/event-driven-pipeline/internal/adapters/redis"
	"tour_of_go/projects/event-driven-pipeline/internal/domain"
	"tour_of_go/projects/event-driven-pipeline/internal/pipeline"
)

// loggingHandler is a simple EventHandler that logs the event.
type loggingHandler struct{}

func (h *loggingHandler) Handle(_ context.Context, event *domain.Event) error {
	slog.Info("processing event",
		"eventID", event.ID,
		"type", event.Type,
		"payload", fmt.Sprintf("%v", event.Payload),
	)
	return nil
}

// loggingDLQ logs events that end up in the dead letter queue.
type loggingDLQ struct{}

func (d *loggingDLQ) Publish(_ context.Context, event *domain.Event) error {
	slog.Warn("event in DLQ", "eventID", event.ID, "retries", event.RetryCount)
	return nil
}

func main() {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	nc, err := nats.Connect(natsURL)
	if err != nil {
		slog.Error("nats connect failed", "error", err)
		os.Exit(1)
	}
	defer nc.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: redisAddr})
	idempotency := redisadapter.NewIdempotencyStore(redisClient)

	processor := pipeline.NewProcessor(&loggingHandler{}, idempotency, &loggingDLQ{}, 100)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	processor.Start(ctx, 4) // 4 worker goroutines

	js, _ := jetstream.New(nc)
	// Ensure stream exists
	js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     "EVENTS",
		Subjects: []string{"events.>"},
	})

	consumer, err := natsadapter.NewConsumer(js, "EVENTS", "order-processor", processor)
	if err != nil {
		slog.Error("consumer init failed", "error", err)
		os.Exit(1)
	}
	consumer.Start(ctx)

	slog.Info("consumer started, waiting for events...")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	<-sig

	slog.Info("shutting down...")
	cancel()
	processor.Wait()
}
