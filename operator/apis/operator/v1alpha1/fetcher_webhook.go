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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var fetcherlog = logf.Log.WithName("fetcher-resource")

func (r *Fetcher) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithDefaulter(r).
		WithValidator(r).
		Complete()
}

// nolint: lll
// +kubebuilder:webhook:admissionReviewVersions=v1,sideEffects=None,path=/mutate-operator-skywalking-apache-org-v1alpha1-fetcher,mutating=true,failurePolicy=fail,groups=operator.skywalking.apache.org,resources=fetchers,verbs=create;update,versions=v1alpha1,name=mfetcher.kb.io

var _ webhook.CustomDefaulter = &Fetcher{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Fetcher) Default(_ context.Context, _ runtime.Object) error {
	fetcherlog.Info("default", "name", r.Name)
	if r.Spec.ClusterName == "" {
		r.Spec.ClusterName = r.Name
	}
	return nil
}

// nolint: lll
// +kubebuilder:webhook:admissionReviewVersions=v1,sideEffects=None,verbs=create;update,path=/validate-operator-skywalking-apache-org-v1alpha1-fetcher,mutating=false,failurePolicy=fail,groups=operator.skywalking.apache.org,resources=fetchers,versions=v1alpha1,name=vfetcher.kb.io

var _ webhook.CustomValidator = &Fetcher{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (r *Fetcher) ValidateCreate(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	fetcherlog.Info("validate create", "name", r.Name)
	return nil, r.validate()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (r *Fetcher) ValidateUpdate(_ context.Context, _ runtime.Object, _ runtime.Object) (admission.Warnings, error) {
	fetcherlog.Info("validate update", "name", r.Name)
	return nil, r.validate()
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (r *Fetcher) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	fetcherlog.Info("validate delete", "name", r.Name)
	return nil, nil
}

func (r *Fetcher) validate() error {
	if r.Spec.ClusterName == "" {
		return fmt.Errorf("cluster name is absent")
	}
	return nil
}
