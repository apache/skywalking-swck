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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var storagelog = logf.Log.WithName("storage-resource")

func (r *Storage) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithDefaulter(r).
		WithValidator(r).
		Complete()
}

// nolint: lll
// +kubebuilder:webhook:admissionReviewVersions=v1,sideEffects=None,path=/mutate-operator-skywalking-apache-org-v1alpha1-storage,mutating=true,failurePolicy=fail,groups=operator.skywalking.apache.org,resources=storages,verbs=create;update,versions=v1alpha1,name=mstorage.kb.io

var _ webhook.CustomDefaulter = &Storage{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (r *Storage) Default(_ context.Context, o runtime.Object) error {
	storage, ok := o.(*Storage)
	if !ok {
		return apierrors.NewBadRequest("object is not a Storage")
	}

	storagelog.Info("default", "name", storage.Name)
	if storage.Spec.ConnectType == "internal" {
		if storage.Spec.Image == "" {
			storage.Spec.Image = "docker.elastic.co/elasticsearch/elasticsearch:7.5.1"
		}
		if storage.Spec.Instances == 0 {
			storage.Spec.Instances = 3
		}
	}
	return nil
}

// nolint: lll
// +kubebuilder:webhook:admissionReviewVersions=v1,sideEffects=None,verbs=create;update,path=/validate-operator-skywalking-apache-org-v1alpha1-storage,mutating=false,failurePolicy=fail,groups=operator.skywalking.apache.org,resources=storages,versions=v1alpha1,name=vstorage.kb.io

var _ webhook.CustomValidator = &Storage{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (r *Storage) ValidateCreate(_ context.Context, o runtime.Object) (admission.Warnings, error) {
	storage, ok := o.(*Storage)
	if !ok {
		return nil, apierrors.NewBadRequest("object is not a Storage")
	}

	storagelog.Info("validate create", "name", storage.Name)
	return nil, storage.valid()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (r *Storage) ValidateUpdate(_ context.Context, o runtime.Object, _ runtime.Object) (admission.Warnings, error) {
	storage, ok := o.(*Storage)
	if !ok {
		return nil, apierrors.NewBadRequest("object is not a Storage")
	}

	storagelog.Info("validate update", "name", storage.Name)
	return nil, storage.valid()
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (r *Storage) ValidateDelete(_ context.Context, o runtime.Object) (admission.Warnings, error) {
	storage, ok := o.(*Storage)
	if !ok {
		return nil, apierrors.NewBadRequest("object is not a Storage")
	}

	storagelog.Info("validate delete", "name", storage.Name)
	return nil, nil
}

func (r *Storage) valid() error {
	var allErrs field.ErrorList
	if r.Spec.Type != "elasticsearch" {
		storagelog.Info("Invalid Storage Type")
		err := field.Invalid(field.NewPath("spec").Child("type"),
			r.Spec.Type,
			"d. must be elasticsearch")
		allErrs = append(allErrs, err)
	}
	if r.Spec.ConnectType != "internal" && r.Spec.ConnectType != "external" {
		storagelog.Info("Invalid Storage ConnectType")
		err := field.Invalid(field.NewPath("spec").Child("connecttype"),
			r.Spec.ConnectType,
			"d. must be internal or external ")
		allErrs = append(allErrs, err)
	}
	if len(allErrs) != 0 {
		return apierrors.NewInvalid(
			schema.GroupKind{Group: r.GroupVersionKind().Group, Kind: r.GroupVersionKind().Kind},
			r.Name,
			allErrs)
	}
	return nil
}
