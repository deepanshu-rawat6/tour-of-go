// Package k8s wraps client-go to manage Greeting custom resources
// from the k8s-controller project.
package k8s

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// GreetingGVR is the GroupVersionResource for the Greeting CRD.
var GreetingGVR = schema.GroupVersionResource{
	Group:    "greeting.example.com",
	Version:  "v1",
	Resource: "greetings",
}

// Greeting is a simplified view of the Greeting CR.
type Greeting struct {
	Name      string
	Namespace string
	Message   string
	Ready     bool
}

// Client wraps the dynamic Kubernetes client for Greeting operations.
type Client struct {
	dynamic   dynamic.Interface
	namespace string
}

// NewClient creates a Client using the current kubeconfig.
func NewClient(namespace string) (*Client, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		// Fall back to kubeconfig (local dev)
		cfg, err = clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
		if err != nil {
			return nil, fmt.Errorf("kubeconfig: %w", err)
		}
	}
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &Client{dynamic: dyn, namespace: namespace}, nil
}

// List returns all Greeting CRs in the namespace.
func (c *Client) List(ctx context.Context) ([]Greeting, error) {
	list, err := c.dynamic.Resource(GreetingGVR).Namespace(c.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	greetings := make([]Greeting, 0, len(list.Items))
	for _, item := range list.Items {
		greetings = append(greetings, toGreeting(item))
	}
	return greetings, nil
}

// Create creates a new Greeting CR.
func (c *Client) Create(ctx context.Context, name, message string) error {
	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "greeting.example.com/v1",
			"kind":       "Greeting",
			"metadata":   map[string]any{"name": name, "namespace": c.namespace},
			"spec":       map[string]any{"message": message},
		},
	}
	_, err := c.dynamic.Resource(GreetingGVR).Namespace(c.namespace).Create(ctx, obj, metav1.CreateOptions{})
	return err
}

// Delete deletes a Greeting CR by name.
func (c *Client) Delete(ctx context.Context, name string) error {
	return c.dynamic.Resource(GreetingGVR).Namespace(c.namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

func toGreeting(u unstructured.Unstructured) Greeting {
	msg, _, _ := unstructured.NestedString(u.Object, "spec", "message")
	ready, _, _ := unstructured.NestedBool(u.Object, "status", "ready")
	return Greeting{
		Name:      u.GetName(),
		Namespace: u.GetNamespace(),
		Message:   msg,
		Ready:     ready,
	}
}
