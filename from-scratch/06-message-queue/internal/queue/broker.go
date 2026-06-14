// Package queue implements an in-memory pub/sub message broker.
package queue

import (
	"sync"
)

// Message is a published message.
type Message struct {
	Topic   string
	Payload string
}

// Broker manages topics and subscriber channels.
type Broker struct {
	mu   sync.RWMutex
	subs map[string][]chan Message
}

func New() *Broker {
	return &Broker{subs: make(map[string][]chan Message)}
}

// Subscribe returns a channel that receives messages published to topic.
func (b *Broker) Subscribe(topic string) <-chan Message {
	ch := make(chan Message, 64)
	b.mu.Lock()
	b.subs[topic] = append(b.subs[topic], ch)
	b.mu.Unlock()
	return ch
}

// Unsubscribe removes a subscriber channel from a topic.
func (b *Broker) Unsubscribe(topic string, ch <-chan Message) {
	b.mu.Lock()
	defer b.mu.Unlock()
	subs := b.subs[topic]
	for i, s := range subs {
		if s == ch {
			b.subs[topic] = append(subs[:i], subs[i+1:]...)
			close(s)
			return
		}
	}
}

// Publish sends msg to all subscribers of msg.Topic.
func (b *Broker) Publish(msg Message) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.subs[msg.Topic] {
		select {
		case ch <- msg:
		default: // slow subscriber — drop
		}
	}
}
