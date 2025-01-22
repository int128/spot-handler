/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// QueueSpec defines the desired state of Queue
type QueueSpec struct {
	// URL points to a queue of SQS.
	URL string `json:"url,omitempty"`

	// PodTermination defines the configuration for Pod termination.
	PodTermination QueuePodTerminationSpec `json:"podTermination,omitempty"`
}

// QueuePodTerminationSpec represents the configuration for Pod termination.
type QueuePodTerminationSpec struct {
	// Enabled indicates whether the Pod termination is enabled.
	// +optional
	Enabled bool `json:"enabled,omitempty"`
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
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   QueueSpec   `json:"spec,omitempty"`
	Status QueueStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// QueueList contains a list of Queue
type QueueList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Queue `json:"items"`
}

// QueueReference is a pointer to a Queue object.
type QueueReference struct {
	Name string `json:"name,omitempty"`
}

func init() {
	SchemeBuilder.Register(&Queue{}, &QueueList{})
}
