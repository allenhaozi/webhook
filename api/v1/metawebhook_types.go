/*
Copyright 2022.

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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MetaWebHookSpec defines the desired state of MetaWebHook
type MetaWebHookSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	ServiceType    string `json:"serviceType,omitempty"`
	Database       string `json:"database,omitempty"`
	DatabaseSchema string `json:"databaseSchema,omitempty"`
	TableFQN       string `json:"tableFQN,omitempty"`
	TableId        string `json:"tableId,omitempty"`
}

// MetaWebHookStatus defines the observed state of MetaWebHook
type MetaWebHookStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Status string `json:"status,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// MetaWebHook is the Schema for the metawebhooks API
type MetaWebHook struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MetaWebHookSpec   `json:"spec,omitempty"`
	Status MetaWebHookStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MetaWebHookList contains a list of MetaWebHook
type MetaWebHookList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MetaWebHook `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MetaWebHook{}, &MetaWebHookList{})
}
