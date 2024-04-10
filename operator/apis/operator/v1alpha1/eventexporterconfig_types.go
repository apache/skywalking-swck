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

// EventExporterConfigSpec defines the desired state of EventExporterConfig
type EventExporterConfigSpec struct {
	// Version of EventExporter.
	// +kubebuilder:validation:Required
	Version string `json:"version,omitempty"`
	// Data of filters and exporters
	// +kubebuilder:validation:Optional
	Data string `json:"data,omitempty"`
}

// EventExporterConfigStatus defines the observed state of EventExporterConfig
type EventExporterConfigStatus struct {
	// The number of eventexporters that need to be configured
	Desired int `json:"desired,omitempty"`
	// The number of eventexporters that configured successfully
	Ready int `json:"ready,omitempty"`
	// The time the EventExporterConfig was created.
	CreationTime metav1.Time `json:"creationTime,omitempty"`
	// The last time this condition was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Version",type="string",priority=1,JSONPath=".spec.version",description="The version"
// +kubebuilder:printcolumn:name="Instances",type="string",JSONPath=".status.desired",description="The number of expected instance"
// +kubebuilder:printcolumn:name="Running",type="string",JSONPath=".status.ready",description="The number of running"

// EventExporterConfig is the Schema for the eventexporterconfigs API
type EventExporterConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EventExporterConfigSpec   `json:"spec,omitempty"`
	Status EventExporterConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EventExporterConfigList contains a list of EventExporterConfig
type EventExporterConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EventExporterConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EventExporterConfig{}, &EventExporterConfigList{})
}
