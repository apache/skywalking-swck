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
	"context"
	"fmt"

	"github.com/Masterminds/semver/v3"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const annotationKeyIstioSetup = "istio-setup-command"

// The oldest OAP this operator is tested and documented against, matching apache/skywalking-helm.
// Below it, the Horizon UI -- the only UI this operator deploys -- is not a supported pairing:
// Horizon 1.0.0 reaches OAP over an admin host that arrived in 11.x, and its template store is an
// 11.x addition.
//
// This is a warning rather than a rejection. An OAPServer already running 9.x keeps working; the
// operator just says so on the next apply instead of failing it.
const minimumSupportedOAPVersion = "10.4.0"

// log is for logging in this package.
var oapserverlog = logf.Log.WithName("oapserver-resource")

func (r *OAPServer) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, r).
		WithDefaulter(r).
		WithValidator(r).
		Complete()
}

// nolint: lll
// +kubebuilder:webhook:admissionReviewVersions=v1,sideEffects=None,path=/mutate-operator-skywalking-apache-org-v1alpha1-oapserver,mutating=true,failurePolicy=fail,groups=operator.skywalking.apache.org,resources=oapservers,verbs=create;update,versions=v1alpha1,name=moapserver.kb.io

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (r *OAPServer) Default(_ context.Context, oapserver *OAPServer) error {
	oapserverlog.Info("default", "name", oapserver.Name)

	image := oapserver.Spec.Image
	if image == "" {
		oapserver.Spec.Image = fmt.Sprintf("apache/skywalking-oap-server:%s", oapserver.Spec.Version)
	}
	for _, envVar := range oapserver.Spec.Config {
		if envVar.Name == "SW_ENVOY_METRIC_ALS_HTTP_ANALYSIS" &&
			oapserver.ObjectMeta.Annotations[annotationKeyIstioSetup] == "" {
			oapserver.Annotations[annotationKeyIstioSetup] = fmt.Sprintf("istioctl install --set profile=demo "+
				"--set meshConfig.defaultConfig.envoyAccessLogService.address=%s.%s:11800 "+
				"--set meshConfig.enableEnvoyAccessLogService=true", oapserver.Name, oapserver.Namespace)
		}
	}

	return nil
}

// nolint: lll
// +kubebuilder:webhook:admissionReviewVersions=v1,sideEffects=None,verbs=create;update,path=/validate-operator-skywalking-apache-org-v1alpha1-oapserver,mutating=false,failurePolicy=fail,groups=operator.skywalking.apache.org,resources=oapservers,versions=v1alpha1,name=voapserver.kb.io

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (r *OAPServer) ValidateCreate(_ context.Context, oapserver *OAPServer) (admission.Warnings, error) {
	oapserverlog.Info("validate create", "name", oapserver.Name)
	return oapserver.versionWarnings(), oapserver.validate()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (r *OAPServer) ValidateUpdate(_ context.Context, oapserver *OAPServer, _ *OAPServer) (admission.Warnings, error) {
	oapserverlog.Info("validate update", "name", oapserver.Name)
	return oapserver.versionWarnings(), oapserver.validate()
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (r *OAPServer) ValidateDelete(_ context.Context, oapserver *OAPServer) (admission.Warnings, error) {
	oapserverlog.Info("validate delete", "name", oapserver.Name)
	return nil, nil
}

// versionWarnings reports an OAP older than the operator supports. An unparseable version -- a
// nightly tag, a fork's scheme -- is left alone rather than guessed at.
func (r *OAPServer) versionWarnings() admission.Warnings {
	current, err := semver.NewVersion(r.Spec.Version)
	if err != nil {
		return nil
	}
	minimum := semver.MustParse(minimumSupportedOAPVersion)
	if !current.LessThan(minimum) {
		return nil
	}
	return admission.Warnings{fmt.Sprintf(
		"OAP %s is older than the oldest supported version %s. SkyWalking Cloud on Kubernetes "+
			"deploys the Horizon UI, which needs the OAP admin host added in 10.x; 11.0.0 is "+
			"recommended. See docs/en/setup/getting-started.md.",
		r.Spec.Version, minimumSupportedOAPVersion)}
}

func (r *OAPServer) validate() error {
	if r.Spec.Image == "" {
		return fmt.Errorf("image is absent")
	}
	return r.validateStorage()
}

// validateStorage refuses an OAPServer that has nowhere to write.
//
// SkyWalking removed H2 permanently in 10.2.0, so no version this operator supports has an embedded
// storage and there is nothing to fall back to. An OAPServer with no storage is not degraded, it is
// broken: it starts, reads the default selector, dials a BanyanDB on 127.0.0.1:17912 and never
// becomes ready. Failing at admission says that in one line instead of leaving a pod crash-looping
// for someone to diagnose.
//
// Setting the storage environment directly in spec.config is still allowed -- it was the only way
// to reach BanyanDB before the operator learned to configure it -- so an explicit SW_STORAGE
// counts as having made the choice.
func (r *OAPServer) validateStorage() error {
	if r.Spec.StorageConfig != nil && r.Spec.StorageConfig.Name != "" {
		return nil
	}
	for _, e := range r.Spec.Config {
		// A name on its own is not a choice of storage: SW_STORAGE with no value and no source
		// reaches the OAP as an empty string, which selects the default and leaves it dialling
		// 127.0.0.1 -- exactly the state this check exists to prevent.
		if e.Name == "SW_STORAGE" && (e.Value != "" || e.ValueFrom != nil) {
			return nil
		}
	}
	return fmt.Errorf("no storage is configured: set spec.storage.name to a Storage resource, " +
		"or set SW_STORAGE in spec.config. SkyWalking removed the embedded H2 in 10.2.0, so there " +
		"is no fallback and an OAPServer without storage never becomes ready")
}
