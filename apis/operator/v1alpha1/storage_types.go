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

// StorageSpec defines the desired state of Storage
type StorageSpec struct {
	// Type of storage.
	// +kubebuilder:validation:Required
	Type string `json:"type,omitempty"`
	// ConnectType is the way to connect storage(e.g. external,internal).
	// +kubebuilder:validation:Required
	ConnectType string `json:"connectType,omitempty"`
	// Address of external storage address.
	// +kubebuilder:validation:Optional
	ConnectAddress string `json:"address,omitempty"`
	// Version of storage.
	// +kubebuilder:validation:Required
	Version string `json:"version,omitempty"`
	// Image is the storage Docker image to deploy.
	// +kubebuilder:validation:Optional
	Image string `json:"image,omitempty"`
	// Instance is the number of storage.
	// +kubebuilder:validation:Optional
	Instances int32 `json:"instances,omitempty"`
	// Security relevant settings
	// +kubebuilder:validation:Optional
	Security SecuritySpec `json:"security,omitempty"`
	// ServiceName relevant settings
	ServiceName string `json:"servicename,omitempty"`
	// Config holds the Storage configuration.
	Config []corev1.EnvVar `json:"config,omitempty"`
}

// SecuritySpec defines the security setting of Storage
type SecuritySpec struct {
	// SSLConfig of  storage .
	// +kubebuilder:validation:Optional
	TLS bool `json:"tls,omitempty"`
	// UserConfig of storage .
	// +kubebuilder:validation:Optional
	User UserSpec `json:"user,omitempty"`
}

// UserSpec defines the user security setting of Storage
type UserSpec struct {
	// SecretName of storage user .
	// +kubebuilder:validation:Optional
	SecretName string `json:"secretName,omitempty"`
}

// StorageStatus defines the observed state of Storage
type StorageStatus struct {
	// readyReplicas is the number of Pods created by the StatefulSet
	// +kubebuilder:validation:Optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`
	// Represents the latest available observations of the underlying statefulset's current state.
	// +kubebuilder:validation:Optional
	Conditions []appsv1.StatefulSetCondition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Instances",type="string",JSONPath=".spec.instances",description="The number of expected instance"
// +kubebuilder:printcolumn:name="ReadyReplicas",type="string",JSONPath=".status.readyReplicas",description="The number of expected instance"
// +kubebuilder:printcolumn:name="Type",type="string",JSONPath=".spec.type",description="The type of strorage"
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=".spec.version",description="The version"
// +kubebuilder:printcolumn:name="ConnectType",type="string",JSONPath=".spec.connectType",description="the way to connect storage"

// Storage is the Schema for the storages API
type Storage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StorageSpec   `json:"spec,omitempty"`
	Status StorageStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// StorageList contains a list of Storage
type StorageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Storage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Storage{}, &StorageList{})
}
