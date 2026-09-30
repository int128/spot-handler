package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// SpotInterruptedNodeSpec defines the desired state of SpotInterruptedNode
type SpotInterruptedNodeSpec struct {
	// Node refers to the Node affected by SpotInterruption
	Node corev1.LocalObjectReference `json:"node,omitempty"`

	// SpotInterruption refers to the SpotInterruption event that caused the Node to be interrupted.
	SpotInterruption SpotInterruptionReference `json:"spotInterruption,omitempty"`
}

// SpotInterruptedNodeStatus defines the observed state of SpotInterruptedNode
type SpotInterruptedNodeStatus struct {
	// Timestamp at which the SpotInterruptedNode was reconciled successfully.
	// +optional
	ReconciledAt metav1.Time `json:"reconciledAt,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// SpotInterruptedNode is the Schema for the spotinterruptednodes API
type SpotInterruptedNode struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of SpotInterruptedNode
	// +required
	Spec SpotInterruptedNodeSpec `json:"spec"`

	// status defines the observed state of SpotInterruptedNode
	// +optional
	Status SpotInterruptedNodeStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// SpotInterruptedNodeList contains a list of SpotInterruptedNode
type SpotInterruptedNodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []SpotInterruptedNode `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &SpotInterruptedNode{}, &SpotInterruptedNodeList{})
		return nil
	})
}
