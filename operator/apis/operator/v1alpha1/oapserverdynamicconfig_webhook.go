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

	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var oapserverdynamicconfiglog = logf.Log.WithName("oapserverdynamicconfig-resource")

func (r *OAPServerDynamicConfig) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, r).
		WithDefaulter(r).
		WithValidator(r).
		Complete()
}

// nolint: lll
//+kubebuilder:webhook:path=/mutate-operator-skywalking-apache-org-v1alpha1-oapserverdynamicconfig,mutating=true,failurePolicy=fail,sideEffects=None,groups=operator.skywalking.apache.org,resources=oapserverdynamicconfigs,verbs=create;update,versions=v1alpha1,name=moapserverdynamicconfig.kb.io,admissionReviewVersions=v1

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (r *OAPServerDynamicConfig) Default(_ context.Context, oapserverdynamicconfig *OAPServerDynamicConfig) error {
	oapserverdynamicconfiglog.Info("default", "name", oapserverdynamicconfig.Name)

	// Default version is "9.5.0"
	if oapserverdynamicconfig.Spec.Version == "" {
		oapserverdynamicconfig.Spec.Version = "9.5.0"
	}

	// Default labelselector is "app=collector,release=skywalking"
	if oapserverdynamicconfig.Spec.LabelSelector == "" {
		oapserverdynamicconfig.Spec.LabelSelector = "app=collector,release=skywalking"
	}
	return nil
}

// nolint: lll
//+kubebuilder:webhook:path=/validate-operator-skywalking-apache-org-v1alpha1-oapserverdynamicconfig,mutating=false,failurePolicy=fail,sideEffects=None,groups=operator.skywalking.apache.org,resources=oapserverdynamicconfigs,verbs=create;update,versions=v1alpha1,name=voapserverdynamicconfig.kb.io,admissionReviewVersions=v1

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (r *OAPServerDynamicConfig) ValidateCreate(_ context.Context, oapserverdynamicconfig *OAPServerDynamicConfig) (admission.Warnings, error) {
	oapserverdynamicconfiglog.Info("validate create", "name", oapserverdynamicconfig.Name)
	return nil, oapserverdynamicconfig.validate()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
// nolint: lll
func (r *OAPServerDynamicConfig) ValidateUpdate(_ context.Context, oapserverdynamicconfig *OAPServerDynamicConfig, _ *OAPServerDynamicConfig) (admission.Warnings, error) {
	oapserverdynamicconfiglog.Info("validate update", "name", oapserverdynamicconfig.Name)
	return nil, oapserverdynamicconfig.validate()
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (r *OAPServerDynamicConfig) ValidateDelete(_ context.Context, oapserverdynamicconfig *OAPServerDynamicConfig) (admission.Warnings, error) {
	oapserverdynamicconfiglog.Info("validate delete", "name", oapserverdynamicconfig.Name)
	return nil, nil
}

func (r *OAPServerDynamicConfig) validate() error {
	if r.Spec.Version == "" {
		return fmt.Errorf("OAPServerDynamicConfig's version is absent")
	} else if r.Spec.LabelSelector == "" {
		return fmt.Errorf("OAPServerDynamicConfig's labelselector is absent")
	}
	return nil
}
