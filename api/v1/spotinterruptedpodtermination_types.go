/*
Copyright 2025.

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SpotInterruptedPodTerminationSpec defines the desired state of SpotInterruptedPodTermination.
type SpotInterruptedPodTerminationSpec struct {
	// Pod refers to the Pod affected by SpotInterruption
	Pod corev1.LocalObjectReference `json:"pod,omitempty"`

	// Node refers to the Node affected by SpotInterruption
	Node corev1.LocalObjectReference `json:"node,omitempty"`

	// InstanceID refers to the instance ID of the Node affected by SpotInterruption
	InstanceID string `json:"instanceID,omitempty"`

	// PodTermination is propagated from the owner object.
	PodTermination PodTerminationSpec `json:"podTermination,omitempty"`
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
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SpotInterruptedPodTerminationSpec   `json:"spec,omitempty"`
	Status SpotInterruptedPodTerminationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SpotInterruptedPodTerminationList contains a list of SpotInterruptedPodTermination.
type SpotInterruptedPodTerminationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SpotInterruptedPodTermination `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SpotInterruptedPodTermination{}, &SpotInterruptedPodTerminationList{})
}
