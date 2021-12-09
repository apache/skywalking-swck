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

// UISpec defines the desired state of UI
type UISpec struct {
	// Version of UI.
	// +kubebuilder:validation:Required
	Version string `json:"version"`
	// Image is the UI Docker image to deploy.
	Image string `json:"image,omitempty"`
	// Count is the number of UI pods
	// +kubebuilder:validation:Required
	Instances int32 `json:"instances"`
	// Backend OAP server address
	// +kubebuilder:validation:Optional
	OAPServerAddress string `json:"OAPServerAddress,omitempty"`
	// Service relevant settings
	// +kubebuilder:validation:Optional
	Service Service `json:"service,omitempty"`
}

// UIStatus defines the observed state of UI
type UIStatus struct {
	// Total number of available pods (ready for at least minReadySeconds) targeted by this deployment.
	// +kubebuilder:validation:Optional
	AvailableReplicas int32 `json:"availableReplicas,omitempty"`
	// externalIPs is a list of IP addresses for which nodes in the cluster
	// will also accept traffic for this service.
	// +kubebuilder:validation:Optional
	ExternalIPs []string `json:"externalIPs,omitempty"`
	// Ports that will be exposed by this service.
	// +kubebuilder:validation:Optional
	Ports []int32 `json:"ports"`
	// +kubebuilder:validation:Optional
	InternalAddress string `json:"internalAddress,omitempty"`
	// Represents the latest available observations of the underlying deployment's current state.
	// +kubebuilder:validation:Optional
	Conditions []appsv1.DeploymentCondition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Version",type="string",priority=1,JSONPath=".spec.version",description="The version"
// +kubebuilder:printcolumn:name="Instances",type="string",JSONPath=".spec.instances",description="The number of expected instance"
// +kubebuilder:printcolumn:name="Running",type="string",JSONPath=".status.availableReplicas",description="The number of running"
// +kubebuilder:printcolumn:name="InternalAddress",type="string",JSONPath=".status.internalAddress",description="The address of OAP server"
// +kubebuilder:printcolumn:name="ExternalIPs",type="string",JSONPath=".status.externalIPs",description="The address of OAP server"
// +kubebuilder:printcolumn:name="Ports",type="string",JSONPath=".status.ports",description="The address of OAP server"
// +kubebuilder:printcolumn:name="Image",type="string",priority=1,JSONPath=".spec.image"

// UI is the Schema for the uis API
type UI struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UISpec   `json:"spec,omitempty"`
	Status UIStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// UIList contains a list of UI
type UIList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []UI `json:"items"`
}

func init() {
	SchemeBuilder.Register(&UI{}, &UIList{})
}
