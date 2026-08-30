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

// UISpec defines the desired state of UI
type UISpec struct {
	// Kind selects which SkyWalking web UI to deploy. "horizon" -- the SkyWalking UI -- is
	// the only supported value and the default.
	//
	// The legacy Booster UI is gone: apache/skywalking removed its submodule in 11.0.0 and no
	// longer builds an image for it. A UI that still says kind: booster is rejected, with a
	// message pointing here.
	// +kubebuilder:validation:Enum=horizon
	// +kubebuilder:default=horizon
	// +kubebuilder:validation:Optional
	Kind string `json:"kind,omitempty"`
	// Version of UI.
	// +kubebuilder:validation:Required
	Version string `json:"version"`
	// Image is the UI Docker image to deploy.
	Image string `json:"image,omitempty"`
	// Count is the number of UI pods
	// +kubebuilder:validation:Required
	Instances int32 `json:"instances"`
	// Backend OAP server address. The operator passes it to Horizon as HORIZON_OAP_QUERY_URL.
	// +kubebuilder:validation:Optional
	OAPServerAddress string `json:"OAPServerAddress,omitempty"`
	// OAPServerAdminAddress is the OAP admin host (port 17128; runtime-rule, dsl-debug, inspect,
	// status). It is NOT derived from OAPServerAddress: the admin host arrived in OAP 11.x, and on
	// 10.x that port belongs to the AI-pipeline URI-recognition server, so guessing it pointed
	// Horizon at the wrong service. Left unset, oap.adminUrl is omitted from the generated
	// HORIZON_OAP_ADMIN_URL is left unset and Horizon reports the admin API unreachable. It also
	// decides the default template mode: live can only read OAP's template store over this host.
	// +kubebuilder:validation:Optional
	OAPServerAdminAddress string `json:"OAPServerAdminAddress,omitempty"`
	// OAPServerZipkinAddress is the OAP Zipkin REST host. It is NOT derived from OAPServerAddress:
	// the OAPServer this operator deploys exposes no Zipkin query port, so <OAPServerAddress>/zipkin
	// was a URL nothing served. Left unset, HORIZON_OAP_ZIPKIN_URL is simply not set.
	// +kubebuilder:validation:Optional
	OAPServerZipkinAddress string `json:"OAPServerZipkinAddress,omitempty"`
	// TemplatesMode selects where Horizon reads dashboard templates from.
	//
	// "live" seeds and reads them through OAP 11's /ui-management/templates* REST API, which Horizon
	// reaches over the OAP admin host. "readonly" renders them from the bundle inside the image and
	// never calls that API.
	//
	// OAP 10.x needs "readonly". It does have persistent template management, but over legacy
	// query-port GraphQL, and Horizon implements only the OAP 11 REST protocol -- so in "live"
	// mode the store is unreachable and Horizon blocks every layer-driven page, Traces most
	// visibly, rather than render a layer whose published template it cannot read. The operator
	// cannot infer this: a UI resource carries an OAP address, not an OAP version.
	// Left unset, the operator picks: live needs a reachable OAP admin host, so it is chosen only
	// when OAPServerAdminAddress is set, and readonly otherwise.
	// +kubebuilder:validation:Enum=live;readonly
	// +kubebuilder:validation:Optional
	TemplatesMode string `json:"templatesMode,omitempty"`
	// Env sets environment variables on the UI container.
	//
	// Horizon's image bakes a fully tokenised horizon.yaml, so every setting it has -- now or in a
	// release this operator predates -- is reachable as an environment variable. Which variables
	// exist, and what they mean, is Horizon's to document, not this operator's: see
	// https://skywalking.apache.org/docs/ for the current set.
	//
	// The operator sets only what it derives from the fields above, and an explicit value here
	// wins over the derived one.
	// +kubebuilder:validation:Optional
	Env []corev1.EnvVar `json:"env,omitempty"`
	// EnvFrom pulls environment variables from Secrets or ConfigMaps.
	//
	// Use it for anything that should not be written into this resource -- credentials, tokens,
	// signing keys -- since everything in Env is visible to anyone with read access on UI objects.
	// +kubebuilder:validation:Optional
	EnvFrom []corev1.EnvFromSource `json:"envFrom,omitempty"`
	// Config is a raw horizon.yaml. When set, it is mounted at /app/horizon.yaml and replaces the
	// tokenised one baked into the image -- which also disables every HORIZON_* variable that the
	// baked file would have carried, including any this operator sets. Prefer Env and EnvFrom;
	// reach for this only when a whole file is genuinely easier than the variables.
	// +kubebuilder:validation:Optional
	Config string `json:"config,omitempty"`
	// Service relevant settings
	// +kubebuilder:validation:Optional
	Service Service `json:"service,omitempty"`
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
