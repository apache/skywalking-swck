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
var uilog = logf.Log.WithName("ui-resource")

func (r *UI) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, r).
		WithDefaulter(r).
		WithValidator(r).
		Complete()
}

// nolint: lll
// +kubebuilder:webhook:admissionReviewVersions=v1,sideEffects=None,path=/mutate-operator-skywalking-apache-org-v1alpha1-ui,mutating=true,failurePolicy=fail,groups=operator.skywalking.apache.org,resources=uis,verbs=create;update,versions=v1alpha1,name=mui.kb.io

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (r *UI) Default(_ context.Context, ui *UI) error {
	uilog.Info("default", "name", ui.Name)

	if ui.Spec.Image == "" {
		ui.Spec.Image = fmt.Sprintf("apache/skywalking-ui:%s", ui.Spec.Version)
	}

	ui.Spec.Service.Template.Default()
	if ui.Spec.OAPServerAddress == "" {
		ui.Spec.OAPServerAddress = fmt.Sprintf("http://%s-oap.%s:12800", ui.Name, ui.Namespace)
	}

	return nil
}

// nolint: lll
// +kubebuilder:webhook:admissionReviewVersions=v1,sideEffects=None,verbs=create;update,path=/validate-operator-skywalking-apache-org-v1alpha1-ui,mutating=false,failurePolicy=fail,groups=operator.skywalking.apache.org,resources=uis,versions=v1alpha1,name=vui.kb.io

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (r *UI) ValidateCreate(_ context.Context, ui *UI) (admission.Warnings, error) {
	uilog.Info("validate create", "name", ui.Name)
	return nil, ui.validate()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (r *UI) ValidateUpdate(_ context.Context, ui *UI, _ *UI) (admission.Warnings, error) {
	uilog.Info("validate update", "name", ui.Name)
	return nil, ui.validate()
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (r *UI) ValidateDelete(_ context.Context, ui *UI) (admission.Warnings, error) {
	uilog.Info("validate delete", "name", ui.Name)
	return nil, nil
}

func (r *UI) validate() error {
	if r.Spec.Image == "" {
		return fmt.Errorf("image is absent")
	}
	if err := r.Spec.Service.Template.Validate(); err != nil {
		return fmt.Errorf("service template is invalid: %w", err)
	}
	if r.Spec.OAPServerAddress == "" {
		return fmt.Errorf("oap server address is absent")
	}
	return nil
}
