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

// PodTerminationSpec represents the configuration for Pod termination.
type PodTerminationSpec struct {
	// Enabled indicates whether to terminate a Pod when the Node is interrupted.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// GracePeriodSeconds overrides the Pod terminationGracePeriodSeconds.
	// +optional
	GracePeriodSeconds *int64 `json:"gracePeriodSeconds,omitempty"`
}
