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

// JavaAgentSpec defines the desired state of JavaAgent
type JavaAgentSpec struct {
	// Pod is the name of injected Pod
	Pod string `json:"pod,omitempty"`
	// ServiceName is the name of service in the injected agent, which need to be printed
	ServiceName string `json:"serviceName,omitempty"`
	// backend_service is the backend service in the injected agent, which need to be printed
	BackendService string `json:"backendService,omitempty"`
	// AgentConfiguration is the injected agent's final configuration
	AgentConfiguration map[string]string `json:"agentConfiguration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Pod",type="string",JSONPath=".spec.pod",description="The name of injected Pod"
// +kubebuilder:printcolumn:name="ServiceName",type="string",JSONPath=".spec.serviceName",description="The name of service in the injected agent"
// +kubebuilder:printcolumn:name="BackendService",type="string",JSONPath=".spec.backendService",description="The backend service in the injected agent"

// JavaAgent is the Schema for the javaagents API
type JavaAgent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec JavaAgentSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// JavaAgentList contains a list of JavaAgent
type JavaAgentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JavaAgent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&JavaAgent{}, &JavaAgentList{})
}
