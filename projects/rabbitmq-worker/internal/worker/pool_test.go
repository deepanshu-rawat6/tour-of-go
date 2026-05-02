package worker_test

import (
	"encoding/json"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"tour_of_go/projects/rabbitmq-worker/internal/domain"
	"tour_of_go/projects/rabbitmq-worker/internal/worker"
)

// mockDelivery builds an amqp.Delivery with the given task and x-death count.
func mockDelivery(t *testing.T, task domain.Task, deathCount int64) amqp.Delivery {
	t.Helper()
	body, _ := json.Marshal(task)
	d := amqp.Delivery{Body: body}
	if deathCount > 0 {
		d.Headers = amqp.Table{
			"x-death": []any{
				amqp.Table{"count": deathCount},
			},
		}
	}
	return d
}

func TestPool_ProcessesDeliveries(t *testing.T) {
	ch := make(chan amqp.Delivery, 5)
	task := domain.Task{ID: "t1", Type: "email", Payload: `{}`}

	for i := 0; i < 5; i++ {
		ch <- mockDelivery(t, task, 0)
	}
	close(ch)

	p := worker.New(2)
	p.Run(ch)
	p.Wait() // must not hang
}

func TestPool_HandlesInvalidJSON(t *testing.T) {
	ch := make(chan amqp.Delivery, 1)
	ch <- amqp.Delivery{Body: []byte("not-json")}
	close(ch)

	p := worker.New(1)
	p.Run(ch)
	p.Wait() // must not panic
}

func TestDeathCount_ZeroWhenNoHeader(t *testing.T) {
	d := mockDelivery(t, domain.Task{ID: "x"}, 0)
	// Verify the delivery has no x-death header
	if _, ok := d.Headers["x-death"]; ok {
		t.Fatal("expected no x-death header")
	}
}

func TestDeathCount_ReadsFromHeader(t *testing.T) {
	d := mockDelivery(t, domain.Task{ID: "x"}, 3)
	deaths, ok := d.Headers["x-death"].([]any)
	if !ok || len(deaths) == 0 {
		t.Fatal("expected x-death header")
	}
	m := deaths[0].(amqp.Table)
	if m["count"].(int64) != 3 {
		t.Fatalf("want 3, got %v", m["count"])
	}
}
