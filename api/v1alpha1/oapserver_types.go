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

// OAPServerSpec defines the desired state of OAPServer
type OAPServerSpec struct {
	// Version of OAP.
	Version string `json:"version"`
	// Image is the OAP Server Docker image to deploy.
	Image string `json:"image,omitempty"`
	// Count is the number of OAP servers
	Instances int32 `json:"instances,imitempty"`
	// Config holds the OAP server configuration.
	Config []corev1.EnvVar `json:"config,omitempty"`
}

// OAPServerPhase is the phase OAP server is in from the controller point of view.
type OAPServerPhase string

const (
	// OAPServerReadyPhase is operating at the desired spec.
	OAPServerReadyPhase OAPServerPhase = "Ready"
	// OAPServerChangingPhase controller is working towards a desired state, cluster can be unavailable.
	OAPServerChangingPhase OAPServerPhase = "Changing"
	// OAPServerResourceInvalid is marking a resource as invalid, should never happen if admission control is installed correctly.
	OAPServerResourceInvalid OAPServerPhase = "Invalid"
)

// OAPServerStatus defines the observed state of OAPServer
type OAPServerStatus struct {
	// The phase OAP servers is in.
	// +kubebuilder:validation:Optional
	Phase OAPServerPhase `json:"phase,omitempty"`
	// A human readable message indicating details about why the OAP servers is in this phase.
	// +kubebuilder:validation:Optional
	Message string `json:"message,omitempty"`
	// A brief CamelCase message indicating details about why the OAP servers is in this state.
	// +kubebuilder:validation:Optional
	Reason string `json:"reason,omitempty"`
	// The last time the phase is transitioned
	// +kubebuilder:validation:Optional
	LastTime metav1.Time `json:"lastTime,omitempty"`
	// Total number of available pods (ready for at least minReadySeconds) targeted by this deployment.
	// +kubebuilder:validation:Optional
	AvailableReplicas int32 `json:"availableReplicas,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Version",type="string",priority=1,JSONPath=".spec.version",description="The version"
// +kubebuilder:printcolumn:name="Instances",type="string",JSONPath=".spec.instances",description="The number of expected instance"
// +kubebuilder:printcolumn:name="Running",type="string",JSONPath=".status.availableReplicas",description="The number of running"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="The phase of deployment"
// +kubebuilder:printcolumn:name="Image",type="string",priority=1,JSONPath=".spec.image"

// OAPServer is the Schema for the oapservers API
type OAPServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OAPServerSpec   `json:"spec,omitempty"`
	Status OAPServerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OAPServerList contains a list of OAPServer
type OAPServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OAPServer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OAPServer{}, &OAPServerList{})
}