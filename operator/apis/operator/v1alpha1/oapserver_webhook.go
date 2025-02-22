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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const annotationKeyIstioSetup = "istio-setup-command"

// log is for logging in this package.
var oapserverlog = logf.Log.WithName("oapserver-resource")

func (r *OAPServer) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithDefaulter(r).
		WithValidator(r).
		Complete()
}

// nolint: lll
// +kubebuilder:webhook:admissionReviewVersions=v1,sideEffects=None,path=/mutate-operator-skywalking-apache-org-v1alpha1-oapserver,mutating=true,failurePolicy=fail,groups=operator.skywalking.apache.org,resources=oapservers,verbs=create;update,versions=v1alpha1,name=moapserver.kb.io

var _ webhook.CustomDefaulter = &OAPServer{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (r *OAPServer) Default(_ context.Context, o runtime.Object) error {
	oapserver, ok := o.(*OAPServer)
	if !ok {
		return apierrors.NewBadRequest("object is not a OAPServer")
	}

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

var _ webhook.CustomValidator = &OAPServer{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (r *OAPServer) ValidateCreate(_ context.Context, o runtime.Object) (admission.Warnings, error) {
	oapserver, ok := o.(*OAPServer)
	if !ok {
		return nil, apierrors.NewBadRequest("object is not a OAPServer")
	}

	oapserverlog.Info("validate create", "name", oapserver.Name)
	return nil, oapserver.validate()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (r *OAPServer) ValidateUpdate(_ context.Context, o runtime.Object, _ runtime.Object) (admission.Warnings, error) {
	oapserver, ok := o.(*OAPServer)
	if !ok {
		return nil, apierrors.NewBadRequest("object is not a OAPServer")
	}

	oapserverlog.Info("validate update", "name", oapserver.Name)
	return nil, oapserver.validate()
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (r *OAPServer) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	oapserverlog.Info("validate delete", "name", r.Name)
	return nil, nil
}

func (r *OAPServer) validate() error {
	if r.Spec.Image == "" {
		return fmt.Errorf("image is absent")
	}
	return nil
}
