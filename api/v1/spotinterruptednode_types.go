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

// SpotInterruptedNodeSpec defines the desired state of SpotInterruptedNode
type SpotInterruptedNodeSpec struct {
	// Node refers to the Node affected by SpotInterruption
	Node corev1.LocalObjectReference `json:"node,omitempty"`

	// InstanceID refers to the instance ID of the Node affected by SpotInterruption
	InstanceID string `json:"instanceID,omitempty"`

	// PodTermination is propagated from the owner object.
	PodTermination PodTerminationSpec `json:"podTermination,omitempty"`
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
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SpotInterruptedNodeSpec   `json:"spec,omitempty"`
	Status SpotInterruptedNodeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SpotInterruptedNodeList contains a list of SpotInterruptedNode
type SpotInterruptedNodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SpotInterruptedNode `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SpotInterruptedNode{}, &SpotInterruptedNodeList{})
}
