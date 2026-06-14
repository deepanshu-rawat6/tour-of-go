package queue_test

import (
	"testing"
	"time"

	"tour_of_go/projects/from-scratch/06-message-queue/internal/queue"
)

func TestBroker_PubSub(t *testing.T) {
	b := queue.New()
	ch := b.Subscribe("events")

	b.Publish(queue.Message{Topic: "events", Payload: "hello"})

	select {
	case msg := <-ch:
		if msg.Payload != "hello" {
			t.Fatalf("want hello, got %s", msg.Payload)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for message")
	}
}

func TestBroker_MultipleSubscribers(t *testing.T) {
	b := queue.New()
	ch1 := b.Subscribe("t")
	ch2 := b.Subscribe("t")

	b.Publish(queue.Message{Topic: "t", Payload: "msg"})

	for _, ch := range []<-chan queue.Message{ch1, ch2} {
		select {
		case m := <-ch:
			if m.Payload != "msg" {
				t.Fatalf("want msg, got %s", m.Payload)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout")
		}
	}
}

func TestBroker_Unsubscribe(t *testing.T) {
	b := queue.New()
	ch := b.Subscribe("t")
	b.Unsubscribe("t", ch)
	// channel should be closed
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("expected closed channel")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("channel not closed after unsubscribe")
	}
}

func TestBroker_DifferentTopics(t *testing.T) {
	b := queue.New()
	ch := b.Subscribe("a")
	b.Publish(queue.Message{Topic: "b", Payload: "wrong"})
	select {
	case <-ch:
		t.Fatal("should not receive message for different topic")
	case <-time.After(50 * time.Millisecond):
		// correct — no message
	}
}
