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

// FetcherSpec defines the desired state of Fetcher
type FetcherSpec struct {
	// Fetcher is the type of how to fetch metrics from target.
	// +kubebuilder:validation:Required
	Type []FetcherType `json:"type,omitempty"`
	// OAPServerAddress is the address of backend OAPServers
	// +kubebuilder:validation:Required
	OAPServerAddress string `json:"OAPServerAddress,omitempty"`
	// ClusterName
	// +kubebuilder:validation:Optional
	ClusterName string `json:"clusterName,omitempty"`
}

// Service Type string describes ingress methods for a service
type FetcherType string

const (
	// ServiceTypeClusterIP means a service will only be accessible inside the
	// cluster, via the cluster IP.
	FetcherTypePrometheus = "prometheus"
)

// FetcherStatus defines the observed state of Fetcher
type FetcherStatus struct {
	// Replicas is currently not being set and might be removed in the next version.
	// +kubebuilder:validation:Optional
	Replicas int32 `json:"replicas,omitempty"`
}

// +kubebuilder:object:root=true

// Fetcher is the Schema for the fetchers API
type Fetcher struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FetcherSpec   `json:"spec,omitempty"`
	Status FetcherStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FetcherList contains a list of Fetcher
type FetcherList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Fetcher `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Fetcher{}, &FetcherList{})
}
