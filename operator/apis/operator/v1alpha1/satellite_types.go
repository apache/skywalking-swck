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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SatelliteSpec defines the desired state of Satellite
type SatelliteSpec struct {
	// Version of Satellite.
	// +kubebuilder:validation:Required
	Version string `json:"version"`
	// Image is the Satellite Docker image to deploy.
	Image string `json:"image,omitempty"`
	// Count is the number of Satellite servers
	// +kubebuilder:validation:Required
	Instances int32 `json:"instances"`
	// Config holds the Satellite server configuration.
	Config []corev1.EnvVar `json:"config,omitempty"`
	// Service relevant settings
	// +kubebuilder:validation:Optional
	Service Service `json:"service,omitempty"`
	// Backend OAP server name
	// +kubebuilder:validation:Optional
	OAPServerName string `json:"OAPServerName,omitempty"`
}

// SatelliteStatus defines the observed state of Satellite
type SatelliteStatus struct {
	// Total number of available pods (ready for at least minReadySeconds) targeted by this deployment.
	// +kubebuilder:validation:Optional
	AvailableReplicas int32 `json:"availableReplicas,omitempty"`
	// Address indicates the entry of Satellite server which ingresses data
	// +kubebuilder:validation:Optional
	Address string `json:"address,omitempty"`
	// Represents the latest available observations of the underlying deployment's current state.
	// +kubebuilder:validation:Optional
	Conditions []appsv1.DeploymentCondition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Version",type="string",priority=1,JSONPath=".spec.version",description="The version"
//+kubebuilder:printcolumn:name="Instances",type="string",JSONPath=".spec.instances",description="The number of expected instance"
//+kubebuilder:printcolumn:name="Running",type="string",JSONPath=".status.availableReplicas",description="The number of running"
//+kubebuilder:printcolumn:name="Address",type="string",JSONPath=".status.address",description="The address of Satellite server"
//+kubebuilder:printcolumn:name="Image",type="string",priority=1,JSONPath=".spec.image"

// Satellite is the Schema for the satellites API
type Satellite struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SatelliteSpec   `json:"spec,omitempty"`
	Status SatelliteStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SatelliteList contains a list of Satellite
type SatelliteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Satellite `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Satellite{}, &SatelliteList{})
}
