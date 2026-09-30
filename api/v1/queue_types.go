package v1

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// QueueSpec defines the desired state of Queue
type QueueSpec struct {
	// URL points to a queue of SQS.
	URL string `json:"url,omitempty"`

	// SpotInterruption defines the configuration for a SpotInterruption event.
	SpotInterruption QueueSpotInterruptionSpec `json:"spotInterruption,omitempty"`
}

// QueueSpotInterruptionSpec represents the configuration for a SpotInterruption event.
type QueueSpotInterruptionSpec struct {
	// PodTermination defines the configuration for Pod termination.
	PodTermination QueuePodTerminationSpec `json:"podTermination,omitempty"`
}

// QueuePodTerminationSpec represents the configuration for Pod termination.
type QueuePodTerminationSpec struct {
	// Enabled indicates whether to terminate a Pod when the Node is interrupted.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// DelaySeconds is the delay before terminating the Pod.
	// The default is 0 (immediately).
	// +optional
	DelaySeconds int64 `json:"delaySeconds,omitempty"`

	// GracePeriodSeconds overrides the Pod terminationGracePeriodSeconds.
	// No override by default.
	// +optional
	GracePeriodSeconds *int64 `json:"gracePeriodSeconds,omitempty"`
}

// DelayDuration returns the time.Duration of DelaySeconds.
func (spec QueuePodTerminationSpec) DelayDuration() time.Duration {
	return time.Duration(spec.DelaySeconds) * time.Second
}

// QueueStatus defines the observed state of Queue
type QueueStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// Queue is the Schema for the queues API
type Queue struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of Queue
	// +required
	Spec QueueSpec `json:"spec"`

	// status defines the observed state of Queue
	// +optional
	Status QueueStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// QueueList contains a list of Queue
type QueueList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []Queue `json:"items"`
}

// QueueReference is a pointer to a Queue object.
type QueueReference struct {
	Name string `json:"name,omitempty"`
}

// QueueReferenceTo creates a QueueReference for the given Queue.
func QueueReferenceTo(queue Queue) QueueReference {
	return QueueReference{
		Name: queue.Name,
	}
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &Queue{}, &QueueList{})
		return nil
	})
}
