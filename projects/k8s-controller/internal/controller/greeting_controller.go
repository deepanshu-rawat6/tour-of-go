// Package controller implements the Greeting controller.
package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	greetingv1 "tour_of_go/projects/k8s-controller/api/v1"
)

// GreetingReconciler reconciles Greeting objects.
// The Reconcile loop is the heart of every K8s controller:
//   - It is called whenever a Greeting resource is created, updated, or deleted.
//   - It is idempotent: running it multiple times produces the same result.
//   - It drives the cluster toward the "desired state" defined in the spec.
type GreetingReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=greeting.example.com,resources=greetings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=greeting.example.com,resources=greetings/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile is the main reconciliation loop.
func (r *GreetingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling Greeting", "name", req.Name, "namespace", req.Namespace)

	// 1. Fetch the Greeting resource that triggered this reconciliation
	greeting := &greetingv1.Greeting{}
	if err := r.Get(ctx, req.NamespacedName, greeting); err != nil {
		if errors.IsNotFound(err) {
			// Resource was deleted — nothing to do (K8s garbage collects owned resources)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("failed to get Greeting: %w", err)
	}

	// 2. Define the desired ConfigMap
	cmName := "greeting-" + greeting.Name
	desired := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: greeting.Namespace,
			// OwnerReference: when the Greeting is deleted, K8s auto-deletes this ConfigMap
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(greeting, greetingv1.GroupVersion.WithKind("Greeting")),
			},
		},
		Data: map[string]string{
			"message": greeting.Spec.Message,
		},
	}

	// 3. Check if the ConfigMap already exists
	existing := &corev1.ConfigMap{}
	err := r.Get(ctx, client.ObjectKey{Name: cmName, Namespace: greeting.Namespace}, existing)

	if errors.IsNotFound(err) {
		// 4a. Create the ConfigMap
		logger.Info("Creating ConfigMap", "configmap", cmName)
		if err := r.Create(ctx, desired); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create ConfigMap: %w", err)
		}
	} else if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get ConfigMap: %w", err)
	} else {
		// 4b. Update if the message changed
		if existing.Data["message"] != greeting.Spec.Message {
			logger.Info("Updating ConfigMap", "configmap", cmName)
			existing.Data["message"] = greeting.Spec.Message
			if err := r.Update(ctx, existing); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to update ConfigMap: %w", err)
			}
		}
	}

	// 5. Update the Greeting status to reflect what we did
	greeting.Status.ConfigMapName = cmName
	greeting.Status.Ready = true
	if err := r.Status().Update(ctx, greeting); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update status: %w", err)
	}

	logger.Info("Reconciliation complete", "configmap", cmName)
	return ctrl.Result{}, nil
}

// SetupWithManager registers the controller with the manager.
// This tells controller-runtime: "watch Greeting resources and call Reconcile for each change."
func (r *GreetingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&greetingv1.Greeting{}).
		Owns(&corev1.ConfigMap{}). // also reconcile when owned ConfigMaps change
		Complete(r)
}
