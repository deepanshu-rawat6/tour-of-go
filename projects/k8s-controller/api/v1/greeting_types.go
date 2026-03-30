// Package v1 contains API types for the greeting.example.com group.
package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GreetingSpec defines the desired state of a Greeting resource.
type GreetingSpec struct {
	// Message is the greeting text to store in the ConfigMap.
	// +kubebuilder:validation:MinLength=1
	Message string `json:"message"`
}

// GreetingStatus defines the observed state of a Greeting resource.
type GreetingStatus struct {
	// ConfigMapName is the name of the ConfigMap created by this Greeting.
	ConfigMapName string `json:"configMapName,omitempty"`
	// Ready indicates whether the ConfigMap has been successfully created.
	Ready bool `json:"ready,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.spec.message`
// +kubebuilder:printcolumn:name="Ready",type=boolean,JSONPath=`.status.ready`

// Greeting is the Schema for the greetings API.
type Greeting struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GreetingSpec   `json:"spec,omitempty"`
	Status GreetingStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GreetingList contains a list of Greeting resources.
type GreetingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Greeting `json:"items"`
}
