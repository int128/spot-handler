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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SpotInterruptedPodSpec   `json:"spec,omitempty"`
	Status SpotInterruptedPodStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SpotInterruptedPodList contains a list of SpotInterruptedPod
type SpotInterruptedPodList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SpotInterruptedPod `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SpotInterruptedPod{}, &SpotInterruptedPodList{})
}
