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

// SQSSpec defines the desired state of SQS
type SQSSpec struct {
	// QueueURL refers to the queue for events.
	QueueURL string `json:"queueURL,omitempty"`
}

// SQSStatus defines the observed state of SQS
type SQSStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// SQS is the Schema for the sqs API
type SQS struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SQSSpec   `json:"spec,omitempty"`
	Status SQSStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SQSList contains a list of SQS
type SQSList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SQS `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SQS{}, &SQSList{})
}
