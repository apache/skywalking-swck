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

// BanyanDBSpec defines the desired state of BanyanDB
type BanyanDBSpec struct {
	// Version of BanyanDB.
	// +kubebuilder:validation:Required
	Version string `json:"version"`

	// Number of replicas
	// +kubebuilder:validation:Required
	Counts int `json:"counts"`

	// Pod template of each BanyanDB instance
	// +kubebuilder:validation:Required
	Image string `json:"image"`

	// Pod affinity
	// +kubebuilder:validation:Optional
	Affinity corev1.Affinity `json:"affinity"`

	// BanyanDB startup parameters
	// +kubebuilder:validation:Optional
	Config []string `json:"config"`

	// TODO support TLS settings
	// BanyanDB Service
	// +kubebuilder:validation:Optional
	Service Service `json:"service,omitempty"`

	// BanyanDB Storage
	// +kubebuilder:validation:Optional
	Storages []corev1.PersistentVolumeClaim `json:"storages,omitempty"`
}

// BanyanDBStatus defines the observed state of BanyanDB
type BanyanDBStatus struct {
	AvailableReplicas int32 `json:"available_pods,omitempty"`
	// Represents the latest available observations of the underlying statefulset's current state.
	// +kubebuilder:validation:Optional
	Conditions []appsv1.DeploymentCondition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// BanyanDB is the Schema for the banyandbs API
type BanyanDB struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BanyanDBSpec   `json:"spec,omitempty"`
	Status BanyanDBStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BanyanDBList contains a list of BanyanDB
type BanyanDBList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BanyanDB `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BanyanDB{}, &BanyanDBList{})
}
