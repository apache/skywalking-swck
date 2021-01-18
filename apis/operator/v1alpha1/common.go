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
	"fmt"

	networkingv1 "k8s.io/api/networking/v1"
)

type Service struct {
	// ServiceTemplate defines the behavior of a service.
	// +kubebuilder:validation:Optional
	Template ServiceTemplate `json:"template,omitempty"`
	// Ingress defines the behavior of an ingress
	// +kubebuilder:validation:Optional
	Ingress Ingress `json:"ingress,omitempty"`
}

func (s *ServiceTemplate) Default() {
	if s.Type == "" {
		s.Type = ServiceTypeClusterIP
	}
}

func (s *ServiceTemplate) Validate() error {
	if s.Type == "" {
		return fmt.Errorf("failed to get service type:%v", *s)
	}
	return nil
}

type ServiceTemplate struct {

	// clusterIP is the IP address of the service and is usually assigned
	// randomly.
	// +kubebuilder:validation:Optional
	ClusterIP string `json:"clusterIP,omitempty"`
	// type determines how the Service is exposed.
	// +kubebuilder:validation:Optional
	Type ServiceType `json:"type,omitempty"`
	// externalIPs is a list of IP addresses for which nodes in the cluster
	// will also accept traffic for this service.
	// +kubebuilder:validation:Optional
	ExternalIPs []string `json:"externalIPs,omitempty"`
	// Only applies to Service Type: LoadBalancer
	// LoadBalancer will get created with the IP specified in this field.
	// +kubebuilder:validation:Optional
	LoadBalancerIP string `json:"loadBalancerIP,omitempty"`
	// If specified and supported by the platform, this will restrict traffic through the cloud-provider
	// load-balancer will be restricted to the specified client IPs.
	// +kubebuilder:validation:Optional
	LoadBalancerSourceRanges []string `json:"loadBalancerSourceRanges,omitempty"`
}

// Service Type string describes ingress methods for a service
type ServiceType string

const (
	// ServiceTypeClusterIP means a service will only be accessible inside the
	// cluster, via the cluster IP.
	ServiceTypeClusterIP ServiceType = "ClusterIP"

	// ServiceTypeNodePort means a service will be exposed on one port of
	// every node, in addition to 'ClusterIP' type.
	ServiceTypeNodePort ServiceType = "NodePort"

	// ServiceTypeLoadBalancer means a service will be exposed via an
	// external load balancer (if the cloud provider supports it), in addition
	// to 'NodePort' type.
	ServiceTypeLoadBalancer ServiceType = "LoadBalancer"
)

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
