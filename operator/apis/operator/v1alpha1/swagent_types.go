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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SwAgentSpec defines the desired state of SwAgent
type SwAgentSpec struct {
	// Selector is the selector label of injected Object
	// +kubebuilder:validation:Optional
	Selector map[string]string `json:"selector,omitempty"`

	// ContainerMatcher is the regular expression to match pods which is to be injected.
	// +kubebuilder:validation:Optional
	ContainerMatcher string `json:"containerMatcher,omitempty"`

	// Java defines Java agent special configs.
	// +kubebuilder:validation:Optional
	JavaSidecar JavaSidecar `json:"javaSidecar,omitempty"`

	// SharedVolume is the shared volume by initContainer and target containers.
	SharedVolume SharedVolume `json:"sharedVolume,omitempty"`

	// SwConfigMap defines the configmap which contains agent.config
	SwConfigMap SwConfigMap `json:"swConfigmap,omitempty"`

	// Select the optional plugin which needs to be moved to the directory(/plugins). Such as`trace`,`webflux`,`cloud-gateway-2.1.x`.
	OptionalPlugins []string `json:"optionalPlugins,omitempty"`

	// Select the optional reporter plugin which needs to be moved to the directory(/plugins). such as `kafka`.
	OptionalReporterPlugin []string `json:"optionalReporterPlugin,omitempty"`
}

// Java defines Java agent special configs.
type JavaSidecar struct {
	// Name is the name for initContainer.
	// +optional
	Name string `json:"name,omitempty" default:"inject-skywalking-agent"`

	// Image is the image for initContainer, which commonly contains SkyWalking java agent SDK.
	// +optional
	Image string `json:"image,omitempty" default:"apache/skywalking-java-agent:8.8.0-java8"`

	// Command is the command for initContainer.
	// +optional
	Command []string `json:"command,omitempty" default:"mkdir -p /sky/agent && cp -r /skywalking/agent/* /sky/agent"`

	// Args is the args for initContainer.
	Args []string `json:"args,omitempty"`

	// Resources is the resources for initContainer pod resources
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Env defines java specific env vars.
	// +optional
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
	Name string `json:"name,omitempty" default:"sky-agent"`
	// The mountPath of shared volume
	MountPath string `json:"mountPath,omitempty" default:"/sky/agent"`
}

type SwConfigMap struct {
	// The name pf configmap used in the injected container as agent.config
	Name string `json:"name,omitempty" default:"java-agent-configmap-volume"`
	// The name of configmap volume.
	VolumeName string `json:"volumeName,omitempty" default:"skywalking-swck-java-agent-configmap"`
	// Mount path of the configmap in the injected container
	VolumeMountPath string `json:"volumeMountPath,omitempty" default:"/sky/agent/config"`
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
