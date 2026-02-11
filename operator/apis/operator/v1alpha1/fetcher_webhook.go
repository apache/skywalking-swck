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
var fetcherlog = logf.Log.WithName("fetcher-resource")

func (r *Fetcher) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, r).
		WithDefaulter(r).
		WithValidator(r).
		Complete()
}

// nolint: lll
// +kubebuilder:webhook:admissionReviewVersions=v1,sideEffects=None,path=/mutate-operator-skywalking-apache-org-v1alpha1-fetcher,mutating=true,failurePolicy=fail,groups=operator.skywalking.apache.org,resources=fetchers,verbs=create;update,versions=v1alpha1,name=mfetcher.kb.io

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (r *Fetcher) Default(_ context.Context, fetcher *Fetcher) error {
	fetcherlog.Info("default", "name", fetcher.Name)
	if fetcher.Spec.ClusterName == "" {
		fetcher.Spec.ClusterName = fetcher.Name
	}
	return nil
}

// nolint: lll
// +kubebuilder:webhook:admissionReviewVersions=v1,sideEffects=None,verbs=create;update,path=/validate-operator-skywalking-apache-org-v1alpha1-fetcher,mutating=false,failurePolicy=fail,groups=operator.skywalking.apache.org,resources=fetchers,versions=v1alpha1,name=vfetcher.kb.io

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (r *Fetcher) ValidateCreate(_ context.Context, fetcher *Fetcher) (admission.Warnings, error) {
	fetcherlog.Info("validate create", "name", fetcher.Name)
	return nil, fetcher.validate()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (r *Fetcher) ValidateUpdate(_ context.Context, fetcher *Fetcher, _ *Fetcher) (admission.Warnings, error) {
	fetcherlog.Info("validate update", "name", fetcher.Name)
	return nil, fetcher.validate()
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (r *Fetcher) ValidateDelete(_ context.Context, fetcher *Fetcher) (admission.Warnings, error) {
	fetcherlog.Info("validate delete", "name", fetcher.Name)
	return nil, nil
}

func (r *Fetcher) validate() error {
	if r.Spec.ClusterName == "" {
		return fmt.Errorf("cluster name is absent")
	}
	return nil
}
