package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// SpotInterruptedPodTerminationSpec defines the desired state of SpotInterruptedPodTermination.
type SpotInterruptedPodTerminationSpec struct {
	// TerminationTimestamp is the timestamp at which the Pod will be terminated.
	TerminationTimestamp metav1.Time `json:"terminationTimestamp,omitempty"`

	// GracePeriodSeconds overrides the Pod terminationGracePeriodSeconds.
	// +optional
	GracePeriodSeconds *int64 `json:"gracePeriodSeconds,omitempty"`

	// Pod refers to the Pod affected by SpotInterruption
	Pod corev1.LocalObjectReference `json:"pod,omitempty"`

	// Node refers to the Node affected by SpotInterruption
	Node corev1.LocalObjectReference `json:"node,omitempty"`

	// InstanceID represents the instance affected by the event.
	InstanceID string `json:"instanceID,omitempty"`
}

// SpotInterruptedPodTerminationStatus defines the observed state of SpotInterruptedPodTermination.
type SpotInterruptedPodTerminationStatus struct {
	// Timestamp at which the SpotInterruptedPodTermination was reconciled successfully.
	// +optional
	ReconciledAt metav1.Time `json:"reconciledAt,omitempty"`

	// RequestedAt indicates the time at which the termination was requested.
	// +optional
	RequestedAt metav1.Time `json:"requestedAt,omitempty"`

	// RequestError indicates the error message when the termination request failed.
	// +optional
	RequestError string `json:"requestError,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// SpotInterruptedPodTermination is the Schema for the spotinterruptedpodterminations API.
type SpotInterruptedPodTermination struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of SpotInterruptedPodTermination
	// +required
	Spec SpotInterruptedPodTerminationSpec `json:"spec"`

	// status defines the observed state of SpotInterruptedPodTermination
	// +optional
	Status SpotInterruptedPodTerminationStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// SpotInterruptedPodTerminationList contains a list of SpotInterruptedPodTermination.
type SpotInterruptedPodTerminationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []SpotInterruptedPodTermination `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &SpotInterruptedPodTermination{}, &SpotInterruptedPodTerminationList{})
		return nil
	})
}
