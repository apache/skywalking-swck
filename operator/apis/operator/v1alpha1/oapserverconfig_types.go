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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FileConfig contains the static file configuration's name, path and data
// Static files refer to all files in the oap-server's configuration directory
// (/skywalking/config)
type FileConfig struct {
	// Name of static file
	Name string `json:"name,omitempty"`
	// Path of static file
	Path string `json:"path,omitempty"`
	// Data of static file
	Data string `json:"data,omitempty"`
}

// OAPServerConfigSpec defines the desired state of OAPServerConfig
type OAPServerConfigSpec struct {
	// Version of OAP.
	//+kubebuilder:validation:Required
	Version string `json:"version,omitempty"`
	// Env holds the OAP server environment configuration.
	// +kubebuilder:validation:Optional
	Env []corev1.EnvVar `json:"env,omitempty"`
	// File holds the OAP server's static file configuration
	// +kubebuilder:validation:Optional
	File []FileConfig `json:"file,omitempty"`
}

// OAPServerConfigStatus defines the observed state of OAPServerConfig
type OAPServerConfigStatus struct {
	// The number of oapserver that need to be configured
	Desired int `json:"desired,omitempty"`
	// The number of oapserver that configured successfully
	Ready int `json:"ready,omitempty"`
	// The time the OAPServerConfig was created.
	CreationTime metav1.Time `json:"creationTime,omitempty"`
	// The last time this condition was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Version",type="string",priority=1,JSONPath=".spec.version",description="The version"
// +kubebuilder:printcolumn:name="Instances",type="string",JSONPath=".status.desired",description="The number of expected instance"
// +kubebuilder:printcolumn:name="Running",type="string",JSONPath=".status.ready",description="The number of running"

// OAPServerConfig is the Schema for the oapserverconfigs API
type OAPServerConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OAPServerConfigSpec   `json:"spec,omitempty"`
	Status OAPServerConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OAPServerConfigList contains a list of OAPServerConfig
type OAPServerConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OAPServerConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OAPServerConfig{}, &OAPServerConfigList{})
}
