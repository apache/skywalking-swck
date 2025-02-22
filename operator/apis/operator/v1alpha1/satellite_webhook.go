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

// log is for logging in this package.
var satellitelog = logf.Log.WithName("satellite-resource")

func (r *Satellite) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithDefaulter(r).
		WithValidator(r).
		Complete()
}

//nolint: lll
//+kubebuilder:webhook:path=/mutate-operator-skywalking-apache-org-v1alpha1-satellite,mutating=true,failurePolicy=fail,sideEffects=None,groups=operator.skywalking.apache.org,resources=satellites,verbs=create;update,versions=v1alpha1,name=msatellite.kb.io,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &Satellite{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (r *Satellite) Default(_ context.Context, o runtime.Object) error {
	satellite, ok := o.(*Satellite)
	if !ok {
		return apierrors.NewBadRequest("object is not a Satellite")
	}

	satellitelog.Info("default", "name", satellite.Name)

	image := satellite.Spec.Image
	if image == "" {
		satellite.Spec.Image = fmt.Sprintf("apache/skywalking-satellite:%s", satellite.Spec.Version)
	}

	oapServerName := satellite.Spec.OAPServerName
	if oapServerName == "" {
		satellite.Spec.OAPServerName = satellite.Name
	}

	if satellite.ObjectMeta.Annotations[annotationKeyIstioSetup] == "" {
		satellite.Annotations[annotationKeyIstioSetup] = fmt.Sprintf("istioctl install --set profile=demo "+
			"--set meshConfig.defaultConfig.envoyAccessLogService.address=%s.%s:11800 "+
			"--set meshConfig.enableEnvoyAccessLogService=true", satellite.Name, satellite.Namespace)
	}

	return nil
}

//nolint: lll
//+kubebuilder:webhook:path=/validate-operator-skywalking-apache-org-v1alpha1-satellite,mutating=false,failurePolicy=fail,sideEffects=None,groups=operator.skywalking.apache.org,resources=satellites,verbs=create;update,versions=v1alpha1,name=vsatellite.kb.io,admissionReviewVersions=v1

var _ webhook.CustomValidator = &Satellite{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (r *Satellite) ValidateCreate(_ context.Context, o runtime.Object) (admission.Warnings, error) {
	satellite, ok := o.(*Satellite)
	if !ok {
		return nil, apierrors.NewBadRequest("object is not a Satellite")
	}

	satellitelog.Info("validate create", "name", satellite.Name)
	return nil, satellite.validate()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (r *Satellite) ValidateUpdate(_ context.Context, o runtime.Object, _ runtime.Object) (admission.Warnings, error) {
	satellite, ok := o.(*Satellite)
	if !ok {
		return nil, apierrors.NewBadRequest("object is not a Satellite")
	}

	satellitelog.Info("validate update", "name", satellite.Name)
	return nil, satellite.validate()
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (r *Satellite) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	satellitelog.Info("validate delete", "name", r.Name)
	return nil, r.validate()
}

func (r *Satellite) validate() error {
	if r.Spec.Image == "" {
		return fmt.Errorf("image is absent")
	}
	return nil
}
