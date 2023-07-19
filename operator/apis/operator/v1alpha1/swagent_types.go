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

// SwAgentSpec defines the desired state of SwAgent
type SwAgentSpec struct {
	// Selector is the selector label of injected Object
	// +kubebuilder:validation:Optional
	Selector map[string]string `json:"selector,omitempty"`

	// ContainerMatcher is the regular expression to match pods which is to be injected.
	// +kubebuilder:validation:Optional
	ContainerMatcher string `json:"containerMatcher,omitempty"`

	// JavaSidecar defines Java agent special configs.
	// +kubebuilder:validation:Optional
	JavaSidecar JavaSidecar `json:"javaSidecar,omitempty"`

	// SharedVolume is the name of an empty volume which shared by initContainer and target containers.
	// +kubebuilder:validation:Optional
	SharedVolumeName string `json:"sharedVolumeName,omitempty"`

	// Select the optional plugin which needs to be moved to the directory(/plugins). Such as`trace`,`webflux`,`cloud-gateway-2.1.x`.
	// +kubebuilder:validation:Optional
	OptionalPlugins []string `json:"optionalPlugins,omitempty"`

	// Select the optional reporter plugin which needs to be moved to the directory(/plugins). such as `kafka`.
	// +kubebuilder:validation:Optional
	OptionalReporterPlugins []string `json:"optionalReporterPlugins,omitempty"`
}

// Java defines Java agent special configs.
type JavaSidecar struct {
	// Name is the name for initContainer.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="inject-skywalking-agent"
	Name string `json:"name,omitempty"`

	// Image is the image for initContainer, which commonly contains SkyWalking java agent SDK.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="apache/skywalking-java-agent:8.16.0-java8"
	Image string `json:"image,omitempty"`

	// Command is the command for initContainer.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default={"mkdir -p /sky/agent && cp -r /skywalking/agent/* /sky/agent"}
	Command []string `json:"command,omitempty"`

	// Args is the args for initContainer.
	// +kubebuilder:validation:Optional
	Args []string `json:"args,omitempty"`

	// Resources is the resources for initContainer pod resources
	// +kubebuilder:validation:Optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Env defines java specific env vars.
	// +kubebuilder:validation:Optional
	Env []corev1.EnvVar `json:"env,omitempty"`
}

// SwAgentStatus defines the observed state of SwAgent
type SwAgentStatus struct {
	// The time the SwAgent was created.
	CreationTime metav1.Time `json:"creationTime,omitempty"`
	// The last time this condition was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
}

type SharedVolume struct {
	// The name of shared volume
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="sky-agent"
	Name string `json:"name,omitempty"`
	// The mountPath of shared volume
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="/sky/agent"
	MountPath string `json:"mountPath,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// SwAgent is the Schema for the swagents API
type SwAgent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SwAgentSpec   `json:"spec,omitempty"`
	Status SwAgentStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SwAgentList contains a list of SwAgent
type SwAgentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SwAgent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SwAgent{}, &SwAgentList{})
}
