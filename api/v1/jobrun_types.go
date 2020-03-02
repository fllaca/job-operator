/*

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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// JobRunSpec defines the desired state of JobRun
type JobRunSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of JobRun. Edit JobRun_types.go to remove/update
	Foo             string         `json:"foo,omitempty"`
	JobTemplateName string         `json:"jobTemplate"`
	Overrides       corev1.PodSpec `json:"overrides,omitempty"`
}

// JobRunStatus defines the observed state of JobRun
type JobRunStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// Active is list of currently running jobs
	Active    []corev1.ObjectReference `json:"active,omitempty"`
	Completed []corev1.ObjectReference `json:"completed,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// JobRun is the Schema for the jobruns API
type JobRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JobRunSpec   `json:"spec,omitempty"`
	Status JobRunStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// JobRunList contains a list of JobRun
type JobRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JobRun `json:"items"`
}

func init() {
	SchemeBuilder.Register(&JobRun{}, &JobRunList{})
}
