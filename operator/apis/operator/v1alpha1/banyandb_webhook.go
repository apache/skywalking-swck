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

var banyandbLog = logf.Log.WithName("banyandb-resource")

func (r *BanyanDB) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// nolint: lll
//+kubebuilder:webhook:path=/mutate-operator-skywalking-apache-org-v1alpha1-banyandb,mutating=true,failurePolicy=fail,sideEffects=None,groups=operator.skywalking.apache.org,resources=banyandbs,verbs=create;update,versions=v1alpha1,name=mbanyandb.kb.io,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &BanyanDB{}

func (r *BanyanDB) Default(_ context.Context, _ runtime.Object) error {
	banyandbLog.Info("default", "name", r.Name)

	if r.Spec.Version == "" {
		// use the latest version by default
		r.Spec.Version = "latest"
	}

	if r.Spec.Image == "" {
		r.Spec.Image = fmt.Sprintf("apache/skywalking-banyandb:%s", r.Spec.Version)
	}

	if r.Spec.Counts == 0 {
		// currently only support one data copy
		r.Spec.Counts = 1
	}

	return nil
}

// nolint: lll
// +kubebuilder:webhook:admissionReviewVersions=v1,sideEffects=None,verbs=create;update,path=/validate-operator-skywalking-apache-org-v1alpha1-banyandb,mutating=false,failurePolicy=fail,groups=operator.skywalking.apache.org,resources=banyandbs,versions=v1alpha1,name=vbanyandb.kb.io

var _ webhook.CustomValidator = &BanyanDB{}

func (r *BanyanDB) ValidateCreate(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	banyandbLog.Info("validate create", "name", r.Name)
	return nil, r.validate()
}

func (r *BanyanDB) ValidateUpdate(_ context.Context, _ runtime.Object, _ runtime.Object) (admission.Warnings, error) {
	banyandbLog.Info("validate update", "name", r.Name)
	return nil, r.validate()
}

func (r *BanyanDB) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	banyandbLog.Info("validate delete", "name", r.Name)
	return nil, r.validate()
}

func (r *BanyanDB) validate() error {
	if r.Spec.Image == "" {
		return fmt.Errorf("image is absent")
	}

	if r.Spec.Counts != 1 {
		return fmt.Errorf("banyandb only support 1 copy for now")
	}

	return nil
}
