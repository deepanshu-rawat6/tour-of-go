// Command producer publishes 10 sample tasks to RabbitMQ.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"tour_of_go/projects/rabbitmq-worker/internal/broker"
	"tour_of_go/projects/rabbitmq-worker/internal/domain"
)

func main() {
	url := envOr("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
	p, err := broker.NewProducer(url)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer p.Close()

	types := []string{"email", "sms", "webhook"}
	ctx := context.Background()
	for i := 1; i <= 10; i++ {
		task := domain.Task{
			ID:      fmt.Sprintf("task-%03d", i),
			Type:    types[i%len(types)],
			Payload: fmt.Sprintf(`{"seq":%d}`, i),
		}
		if err := p.Publish(ctx, task); err != nil {
			log.Fatalf("publish task %s: %v", task.ID, err)
		}
		log.Printf("published %s (type=%s)", task.ID, task.Type)
	}
	log.Println("done — check http://localhost:15672 (guest/guest)")
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
