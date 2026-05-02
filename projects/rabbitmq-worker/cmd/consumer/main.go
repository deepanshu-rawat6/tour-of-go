// Command consumer starts a worker pool that processes tasks from RabbitMQ.
// Graceful shutdown: SIGTERM/SIGINT stops consuming, drains in-flight messages, then exits.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"tour_of_go/projects/rabbitmq-worker/internal/broker"
	"tour_of_go/projects/rabbitmq-worker/internal/worker"
)

func main() {
	url := envOr("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	prefetch := envInt("PREFETCH", 5)
	workers := envInt("WORKERS", 3)

	consumer, err := broker.NewConsumer(url, prefetch)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}

	deliveries, err := consumer.Consume()
	if err != nil {
		log.Fatalf("consume: %v", err)
	}

	pool := worker.New(workers)
	pool.Run(deliveries)
	log.Printf("consumer started (workers=%d, prefetch=%d)", workers, prefetch)

	// Graceful shutdown on SIGTERM / SIGINT
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	<-ctx.Done()

	log.Println("shutdown signal received — draining in-flight messages...")
	consumer.Close() // closing the channel causes deliveries to be closed → workers exit
	pool.Wait()      // wait for all in-flight messages to be acked/nacked
	log.Println("shutdown complete")
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
