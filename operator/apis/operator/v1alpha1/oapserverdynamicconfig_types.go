// Licensed to Apache Software Foundation (ASF) under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Apache Software Foundation (ASF) licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Config contains the dynamic configuration's key and value
type Config struct {
	// configuration's key
	Name string `json:"name,omitempty"`
	// configuration's value
	Value string `json:"value,omitempty"`
}

// OAPServerDynamicConfigSpec defines the desired state of OAPServerDynamicConfig
type OAPServerDynamicConfigSpec struct {
	// Version of OAP.
	//+kubebuilder:validation:Required
	Version string `json:"version,omitempty"`
	// Locate specific configmap
	// +kubebuilder:validation:Optional
	LabelSelector string `json:"labelSelector,omitempty"`
	// All configurations' key and value
	// +kubebuilder:validation:Optional
	Data []Config `json:"data,omitempty"`
}

// OAPServerDynamicConfigStatus defines the observed state of OAPServerDynamicConfig
type OAPServerDynamicConfigStatus struct {
	// The state of dynamic configuration
	State string `json:"state,omitempty"`
	// The time the OAPServerDynamicConfig was created.
	CreationTime metav1.Time `json:"creationTime,omitempty"`
	// The last time this condition was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
}

//+kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Version",type="string",priority=1,JSONPath=".spec.version",description="The version"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state",description="The state of dynamic configuration"

// OAPServerDynamicConfig is the Schema for the oapserverdynamicconfigs API
type OAPServerDynamicConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OAPServerDynamicConfigSpec   `json:"spec,omitempty"`
	Status OAPServerDynamicConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OAPServerDynamicConfigList contains a list of OAPServerDynamicConfig
type OAPServerDynamicConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OAPServerDynamicConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OAPServerDynamicConfig{}, &OAPServerDynamicConfigList{})
}
