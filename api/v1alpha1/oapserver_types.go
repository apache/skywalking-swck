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
	Instances int `json:"instances,imitempty"`
	// Config holds the OAP server configuration.
	Config map[string]string `json:"config,omitempty"`
	// PodTemplate provides customisation options (labels, annotations, affinity rules, resource requests, and so on) for the Pods belonging to this OAP server.
	// +kubebuilder:validation:Optional
	PodTemplate corev1.PodTemplateSpec `json:"podTemplate,omitempty"`
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
	Phase OAPServerPhase `json:"phase,omitempty"`
	// A human readable message indicating details about why the OAP servers is in this phase.
	// +kubebuilder:validation:Optional
	Message string `json:"message,omitempty"`
	// A brief CamelCase message indicating details about why the OAP servers is in this state.
	// +kubebuilder:validation:Optional
	Reason string `json:"reason,omitempty"`
}

// +kubebuilder:object:root=true

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
