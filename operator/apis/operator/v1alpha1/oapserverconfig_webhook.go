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
var oapserverconfiglog = logf.Log.WithName("oapserverconfig-resource")

func (r *OAPServerConfig) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithDefaulter(r).
		WithValidator(r).
		Complete()
}

// nolint: lll
//+kubebuilder:webhook:path=/mutate-operator-skywalking-apache-org-v1alpha1-oapserverconfig,mutating=true,failurePolicy=fail,sideEffects=None,groups=operator.skywalking.apache.org,resources=oapserverconfigs,verbs=create;update,versions=v1alpha1,name=moapserverconfig.kb.io,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &OAPServerConfig{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (r *OAPServerConfig) Default(_ context.Context, o runtime.Object) error {
	oapserverconfig, ok := o.(*OAPServerConfig)
	if !ok {
		return apierrors.NewBadRequest("object is not a OAPServerConfig")
	}

	oapserverconfiglog.Info("default", "name", oapserverconfig.Name)

	// Default version is "9.5.0"
	if oapserverconfig.Spec.Version == "" {
		oapserverconfig.Spec.Version = "9.5.0"
	}

	return nil
}

// nolint: lll
//+kubebuilder:webhook:path=/validate-operator-skywalking-apache-org-v1alpha1-oapserverconfig,mutating=false,failurePolicy=fail,sideEffects=None,groups=operator.skywalking.apache.org,resources=oapserverconfigs,verbs=create;update,versions=v1alpha1,name=voapserverconfig.kb.io,admissionReviewVersions=v1

var _ webhook.CustomValidator = &OAPServerConfig{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (r *OAPServerConfig) ValidateCreate(_ context.Context, o runtime.Object) (admission.Warnings, error) {
	oapserverconfig, ok := o.(*OAPServerConfig)
	if !ok {
		return nil, apierrors.NewBadRequest("object is not a OAPServerConfig")
	}

	oapserverconfiglog.Info("validate create", "name", oapserverconfig.Name)
	return nil, oapserverconfig.validate()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (r *OAPServerConfig) ValidateUpdate(_ context.Context, o runtime.Object, _ runtime.Object) (admission.Warnings, error) {
	oapserverconfig, ok := o.(*OAPServerConfig)
	if !ok {
		return nil, apierrors.NewBadRequest("object is not a OAPServerConfig")
	}

	oapserverconfiglog.Info("validate update", "name", oapserverconfig.Name)
	return nil, oapserverconfig.validate()
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (r *OAPServerConfig) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	oapserverconfiglog.Info("validate delete", "name", r.Name)
	return nil, nil
}

func (r *OAPServerConfig) validate() error {
	if r.Spec.Version == "" {
		return fmt.Errorf("OAPServerconfig's version is absent")
	}
	return nil
}
