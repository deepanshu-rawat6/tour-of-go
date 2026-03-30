package k8s

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
)

// SSEEvent is sent to browsers via Server-Sent Events.
type SSEEvent struct {
	Type     string   `json:"type"` // "ADDED" | "MODIFIED" | "DELETED"
	Greeting Greeting `json:"greeting"`
}

// Watcher watches Greeting CRs and sends SSE events to subscribers.
type Watcher struct {
	dynamic   dynamic.Interface
	namespace string
	mu        sync.RWMutex
	subs      map[chan SSEEvent]struct{}
	log       *slog.Logger
}

func NewWatcher(client *Client) *Watcher {
	return &Watcher{
		dynamic:   client.dynamic,
		namespace: client.namespace,
		subs:      make(map[chan SSEEvent]struct{}),
		log:       slog.Default(),
	}
}

// Subscribe returns a channel that receives SSE events.
func (w *Watcher) Subscribe() chan SSEEvent {
	ch := make(chan SSEEvent, 16)
	w.mu.Lock()
	w.subs[ch] = struct{}{}
	w.mu.Unlock()
	return ch
}

// Unsubscribe removes a subscriber channel.
func (w *Watcher) Unsubscribe(ch chan SSEEvent) {
	w.mu.Lock()
	delete(w.subs, ch)
	w.mu.Unlock()
	close(ch)
}

// Start begins watching Greeting resources. Stops when ctx is cancelled.
func (w *Watcher) Start(ctx context.Context) {
	go func() {
		watcher, err := w.dynamic.Resource(GreetingGVR).Namespace(w.namespace).Watch(ctx, metav1.ListOptions{})
		if err != nil {
			w.log.Error("watch failed", "error", err)
			return
		}
		defer watcher.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-watcher.ResultChan():
				if !ok {
					return
				}
				w.broadcast(event)
			}
		}
	}()
}

func (w *Watcher) broadcast(event watch.Event) {
	var g Greeting
	if u, ok := event.Object.(*unstructured.Unstructured); ok {
		g = toGreeting(*u)
	}
	sse := SSEEvent{Type: string(event.Type), Greeting: g}
	data, _ := json.Marshal(sse)
	_ = data

	w.mu.RLock()
	for ch := range w.subs {
		select {
		case ch <- sse:
		default:
		}
	}
	w.mu.RUnlock()
}
