package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// SpotInterruptedPodSpec represents a Pod affected by SpotInterruption
type SpotInterruptedPodSpec struct {
	// Pod refers to the Pod affected by SpotInterruption
	Pod corev1.LocalObjectReference `json:"pod,omitempty"`

	// Node refers to the Node affected by SpotInterruption
	Node corev1.LocalObjectReference `json:"node,omitempty"`

	// SpotInterruption refers to the SpotInterruption event that caused the Pod to be interrupted.
	SpotInterruption SpotInterruptionReference `json:"spotInterruption,omitempty"`
}

// SpotInterruptedPodStatus defines the observed state of SpotInterruptedPod
type SpotInterruptedPodStatus struct {
	// Timestamp at which the SpotInterruptedPod was reconciled successfully.
	// +optional
	ReconciledAt metav1.Time `json:"reconciledAt,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// SpotInterruptedPod is the Schema for the spotinterruptedpods API
type SpotInterruptedPod struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of SpotInterruptedPod
	// +required
	Spec SpotInterruptedPodSpec `json:"spec"`

	// status defines the observed state of SpotInterruptedPod
	// +optional
	Status SpotInterruptedPodStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// SpotInterruptedPodList contains a list of SpotInterruptedPod
type SpotInterruptedPodList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []SpotInterruptedPod `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &SpotInterruptedPod{}, &SpotInterruptedPodList{})
		return nil
	})
}
