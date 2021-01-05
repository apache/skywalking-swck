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
	networkingv1 "k8s.io/api/networking/v1"
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
	// +optional
	OAPServerAddress string `json:"OAPServerAddress,omitempty"`
	// Service relevant settings
	// +optional
	Service Service `json:"service,omitempty"`
}

type Service struct {
	// ServiceSpec defines the behavior of a service.
	// +optional
	ServiceSpec corev1.ServiceSpec `json:"serviceSpec,omitempty"`
	// Ingress defines the behavior of an ingress
	// +optional
	Ingress Ingress `json:"ingress,omitempty"`
}

type Ingress struct {
	// Annotations is an unstructured key value map stored with a resource that may be
	// set by external tools to store and retrieve arbitrary metadata. They are not
	// queryable and should be preserved when modifying objects.
	// +kubebuilder:validation:Optional
	Annotations map[string]string `json:"annotations,omitempty"`
	// Host is the fully qualified domain name of a network host, as defined by RFC 3986.
	// Note the following deviations from the "host" part of the
	// URI as defined in RFC 3986
	// +kubebuilder:validation:Optional
	Host string `json:"host,omitempty" protobuf:"bytes,1,opt,name=host"`
	// IngressClassName is the name of the IngressClass cluster resource. The
	// associated IngressClass defines which controller will implement the
	// resource. This replaces the deprecated `kubernetes.io/ingress.class`
	// annotation. For backwards compatibility, when that annotation is set, it
	// must be given precedence over this field. The controller may emit a
	// warning if the field and annotation have different values.
	// Implementations of this API should ignore Ingresses without a class
	// specified. An IngressClass resource may be marked as default, which can
	// be used to set a default value for this field. For more information,
	// refer to the IngressClass documentation.
	// +kubebuilder:validation:Optional
	IngressClassName *string `json:"ingressClassName,omitempty" protobuf:"bytes,4,opt,name=ingressClassName"`
	// TLS configuration. Currently the Ingress only supports a single TLS
	// port, 443. If multiple members of this list specify different hosts, they
	// will be multiplexed on the same port according to the hostname specified
	// through the SNI TLS extension, if the ingress controller fulfilling the
	// ingress supports SNI.
	// +kubebuilder:validation:Optional
	TLS []networkingv1.IngressTLS `json:"tls,omitempty" protobuf:"bytes,2,rep,name=tls"`
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
