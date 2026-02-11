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

const (
	latestVersion = "latest"
	image         = "apache/skywalking-kubernetes-event-exporter"
)

// log is for logging in this package.
var eventexporterlog = logf.Log.WithName("eventexporter-resource")

func (r *EventExporter) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, r).
		WithDefaulter(r).
		WithValidator(r).
		Complete()
}

// nolint: lll
//+kubebuilder:webhook:path=/mutate-operator-skywalking-apache-org-v1alpha1-eventexporter,mutating=true,failurePolicy=fail,sideEffects=None,groups=operator.skywalking.apache.org,resources=eventexporters,verbs=create;update,versions=v1alpha1,name=meventexporter.kb.io,admissionReviewVersions=v1

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (r *EventExporter) Default(_ context.Context, eventexporter *EventExporter) error {
	eventexporterlog.Info("default", "name", eventexporter.Name)

	if eventexporter.Spec.Version == "" {
		eventexporter.Spec.Version = latestVersion
	}

	if eventexporter.Spec.Image == "" {
		eventexporter.Spec.Image = fmt.Sprintf("%s:%s", image, eventexporter.Spec.Version)
	}

	if eventexporter.Spec.Replicas == 0 {
		eventexporter.Spec.Replicas = 1
	}

	return nil
}

// nolint: lll
// +kubebuilder:webhook:admissionReviewVersions=v1,sideEffects=None,path=/mutate-operator-skywalking-apache-org-v1alpha1-eventexporter,mutating=true,failurePolicy=fail,groups=operator.skywalking.apache.org,resources=eventexporters,verbs=create;update,versions=v1alpha1,name=meventexporter.kb.io

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (r *EventExporter) ValidateCreate(_ context.Context, eventexporter *EventExporter) (admission.Warnings, error) {
	eventexporterlog.Info("validate create", "name", eventexporter.Name)

	return nil, eventexporter.validate()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (r *EventExporter) ValidateUpdate(_ context.Context, eventexporter *EventExporter, _ *EventExporter) (admission.Warnings, error) {
	eventexporterlog.Info("validate update", "name", eventexporter.Name)

	return nil, eventexporter.validate()
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (r *EventExporter) ValidateDelete(_ context.Context, eventexporter *EventExporter) (admission.Warnings, error) {
	eventexporterlog.Info("validate delete", "name", eventexporter.Name)

	return nil, nil
}

func (r *EventExporter) validate() error {
	if r.Spec.Image == "" {
		return fmt.Errorf("image is absent")
	}
	return nil
}
