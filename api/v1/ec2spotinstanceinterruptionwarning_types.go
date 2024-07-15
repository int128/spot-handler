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

// EC2SpotInstanceInterruptionWarningSpec represents an EC2SpotInstanceInterruptionWarning event.
type EC2SpotInstanceInterruptionWarningSpec struct {
	EventTime        metav1.Time `json:"eventTime,omitempty"`
	InstanceID       string      `json:"instanceID,omitempty"`
	AvailabilityZone string      `json:"availabilityZone,omitempty"`
}

// EC2SpotInstanceInterruptionWarningStatus defines the observed state of EC2SpotInstanceInterruptionWarning
type EC2SpotInstanceInterruptionWarningStatus struct {
	ProcessedTime metav1.Time `json:"processedTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster

// EC2SpotInstanceInterruptionWarning is the Schema for the ec2spotinstanceinterruptionwarnings API
type EC2SpotInstanceInterruptionWarning struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EC2SpotInstanceInterruptionWarningSpec   `json:"spec,omitempty"`
	Status EC2SpotInstanceInterruptionWarningStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EC2SpotInstanceInterruptionWarningList contains a list of EC2SpotInstanceInterruptionWarning
type EC2SpotInstanceInterruptionWarningList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EC2SpotInstanceInterruptionWarning `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EC2SpotInstanceInterruptionWarning{}, &EC2SpotInstanceInterruptionWarningList{})
}
