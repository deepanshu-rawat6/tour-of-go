package informer

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/tour-of-go/k8s-event-sink/internal/core"
)

// Processor is the interface the informer calls for each event.
type Processor interface {
	Process(ctx context.Context, event core.Event)
}

// Watcher manages K8s informers for one or more namespaces.
type Watcher struct {
	client    kubernetes.Interface
	processor Processor
	stopCh    chan struct{}
}

// New creates a Watcher using the given kubeconfig path (empty = in-cluster).
func New(kubeconfigPath string, processor Processor) (*Watcher, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfigPath != "" {
		rules.ExplicitPath = kubeconfigPath
	}
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, nil).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("building kubeconfig: %w", err)
	}
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("creating k8s client: %w", err)
	}
	return &Watcher{client: client, processor: processor, stopCh: make(chan struct{})}, nil
}

// Start launches informers for the given namespaces.
// Pass ["*"] for cluster-wide watching.
func (w *Watcher) Start(ctx context.Context, namespaces []string) {
	if len(namespaces) == 1 && namespaces[0] == "*" {
		w.startInformer(ctx, "")
		return
	}
	for _, ns := range namespaces {
		w.startInformer(ctx, ns)
	}
}

func (w *Watcher) startInformer(ctx context.Context, namespace string) {
	factory := informers.NewSharedInformerFactoryWithOptions(
		w.client, 30*time.Second,
		informers.WithNamespace(namespace),
	)
	eventInformer := factory.Core().V1().Events().Informer()
	eventInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if e, ok := obj.(*corev1.Event); ok {
				w.processor.Process(ctx, toEvent(e))
			}
		},
		UpdateFunc: func(_, newObj interface{}) {
			if e, ok := newObj.(*corev1.Event); ok {
				w.processor.Process(ctx, toEvent(e))
			}
		},
	})
	factory.Start(w.stopCh)
}

// Stop shuts down all informers.
func (w *Watcher) Stop() { close(w.stopCh) }

// toEvent converts a Kubernetes v1.Event to the core domain Event.
func toEvent(e *corev1.Event) core.Event {
	firstSeen := e.FirstTimestamp.Time
	lastSeen := e.LastTimestamp.Time
	if firstSeen.IsZero() {
		firstSeen = e.CreationTimestamp.Time
	}
	if lastSeen.IsZero() {
		lastSeen = firstSeen
	}
	return core.Event{
		ID:        eventID(e),
		Namespace: e.Namespace,
		Pod:       e.InvolvedObject.Name,
		Reason:    e.Reason,
		Message:   e.Message,
		Type:      e.Type,
		Count:     int(e.Count),
		FirstSeen: firstSeen,
		LastSeen:  lastSeen,
	}
}

// eventID generates a stable ID from the event's key fields.
func eventID(e *corev1.Event) string {
	key := fmt.Sprintf("%s/%s/%s/%s", e.Namespace, e.InvolvedObject.Name, e.Reason, e.FirstTimestamp.String())
	return fmt.Sprintf("%x", sha256.Sum256([]byte(key)))[:16]
}
