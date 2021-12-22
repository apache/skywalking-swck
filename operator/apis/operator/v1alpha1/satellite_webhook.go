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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var satellitelog = logf.Log.WithName("satellite-resource")

func (r *Satellite) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//nolint: lll
//+kubebuilder:webhook:path=/mutate-operator-skywalking-apache-org-v1alpha1-satellite,mutating=true,failurePolicy=fail,sideEffects=None,groups=operator.skywalking.apache.org,resources=satellites,verbs=create;update,versions=v1alpha1,name=msatellite.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Satellite{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Satellite) Default() {
	satellitelog.Info("default", "name", r.Name)

	image := r.Spec.Image
	if image == "" {
		r.Spec.Image = fmt.Sprintf("apache/skywalking-satellite:%s", r.Spec.Version)
	}

	oapServerName := r.Spec.OAPServerName
	if oapServerName == "" {
		r.Spec.OAPServerName = r.Name
	}

	if r.ObjectMeta.Annotations[annotationKeyIstioSetup] == "" {
		r.Annotations[annotationKeyIstioSetup] = fmt.Sprintf("istioctl install --set profile=demo "+
			"--set meshConfig.defaultConfig.envoyAccessLogService.address=%s.%s:11800 "+
			"--set meshConfig.enableEnvoyAccessLogService=true", r.Name, r.Namespace)
	}
}

//nolint: lll
//+kubebuilder:webhook:path=/validate-operator-skywalking-apache-org-v1alpha1-satellite,mutating=false,failurePolicy=fail,sideEffects=None,groups=operator.skywalking.apache.org,resources=satellites,verbs=create;update,versions=v1alpha1,name=vsatellite.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Satellite{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Satellite) ValidateCreate() error {
	satellitelog.Info("validate create", "name", r.Name)
	return r.validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Satellite) ValidateUpdate(old runtime.Object) error {
	satellitelog.Info("validate update", "name", r.Name)
	return r.validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Satellite) ValidateDelete() error {
	satellitelog.Info("validate delete", "name", r.Name)
	return r.validate()
}

func (r *Satellite) validate() error {
	if r.Spec.Image == "" {
		return fmt.Errorf("image is absent")
	}
	return nil
}
