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
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ExporterConfig map[string]interface{}

// EventExporterSpec defines the desired state of EventExporter
type EventExporterSpec struct {
	// Version of EventExporter.
	// +kubebuilder:validation:Required
	Version string `json:"version,omitempty"`
	// Image is the event exporter Docker image to deploy.
	Image string `json:"image,omitempty"`
	// Instances is the number of event exporter pods
	// +kubebuilder:validation:Required
	Instances int32 `json:"instances,omitempty"`
	// Data of filters and exporters
	// +kubebuilder:validation:Optional
	Data string `json:"data,omitempty"`
}

// Important: Run "make" to regenerate code after modifying this file

// EventExporterStatus defines the observed state of EventExporter
type EventExporterStatus struct {
	// Total number of available pods targeted by this deployment.
	// +kubebuilder:validation:Optional
	AvailableReplicas int32 `json:"availableReplicas,omitempty"`
	// Represents the latest available observations of the underlying deployment's current state.
	// +kubebuilder:validation:Optional
	Conditions []appsv1.DeploymentCondition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Version",type="string",priority=1,JSONPath=".spec.version",description="The version"
// +kubebuilder:printcolumn:name="Image",type="string",priority=1,JSONPath=".spec.image"
// +kubebuilder:printcolumn:name="Instances",type="string",JSONPath=".spec.instances",description="The number of expected instances"

// EventExporter is the Schema for the eventexporters API
type EventExporter struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EventExporterSpec   `json:"spec,omitempty"`
	Status EventExporterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// EventExporterList contains a list of EventExporter
type EventExporterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EventExporter `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EventExporter{}, &EventExporterList{})
}
