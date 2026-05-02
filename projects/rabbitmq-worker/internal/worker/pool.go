// Package worker provides a fixed-size goroutine pool for processing tasks.
package worker

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
	"tour_of_go/projects/rabbitmq-worker/internal/domain"
)

const maxRetries = 3

// Pool processes deliveries from a channel using a fixed number of goroutines.
type Pool struct {
	size int
	wg   sync.WaitGroup
}

func New(size int) *Pool { return &Pool{size: size} }

// Run starts `size` workers consuming from deliveries until the channel is closed.
func (p *Pool) Run(deliveries <-chan amqp.Delivery) {
	for i := 0; i < p.size; i++ {
		p.wg.Add(1)
		go func(id int) {
			defer p.wg.Done()
			for d := range deliveries {
				p.handle(id, d)
			}
		}(i)
	}
}

// Wait blocks until all in-flight messages are processed.
func (p *Pool) Wait() { p.wg.Wait() }

func (p *Pool) handle(workerID int, d amqp.Delivery) {
	var task domain.Task
	if err := json.Unmarshal(d.Body, &task); err != nil {
		log.Printf("[worker %d] bad message body: %v — nack no-requeue", workerID, err)
		d.Nack(false, false) // malformed → DLX immediately
		return
	}

	retries := deathCount(d)
	log.Printf("[worker %d] processing %s (type=%s, retries=%d)", workerID, task.ID, task.Type, retries)

	if err := process(task); err != nil {
		if retries >= maxRetries {
			log.Printf("[worker %d] %s exhausted retries → DLX", workerID, task.ID)
			d.Nack(false, false) // no-requeue → routed to DLX
		} else {
			log.Printf("[worker %d] %s failed, requeue (retry %d/%d)", workerID, task.ID, retries+1, maxRetries)
			d.Nack(false, true) // requeue for retry
		}
		return
	}
	log.Printf("[worker %d] %s done ✓", workerID, task.ID)
	d.Ack(false)
}

// process simulates work with a ~10% random failure rate.
func process(task domain.Task) error {
	if rand.Intn(10) == 0 {
		return fmt.Errorf("simulated failure for %s", task.ID)
	}
	return nil
}

// deathCount reads the x-death header to determine how many times a message has been dead-lettered.
func deathCount(d amqp.Delivery) int {
	deaths, ok := d.Headers["x-death"].([]any)
	if !ok {
		return 0
	}
	total := 0
	for _, item := range deaths {
		if m, ok := item.(amqp.Table); ok {
			if count, ok := m["count"].(int64); ok {
				total += int(count)
			}
		}
	}
	return total
}
