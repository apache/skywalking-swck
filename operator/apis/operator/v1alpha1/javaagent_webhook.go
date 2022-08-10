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
var javaagentlog = logf.Log.WithName("javaagent-resource")

const (
	// the ServiceName and BackendService are important information that need to be printed
	ServiceName    = "agent.service_name"
	BackendService = "collector.backend_service"
)

func (r *JavaAgent) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// nolint: lll
// +kubebuilder:webhook:admissionReviewVersions=v1,sideEffects=None,path=/mutate-operator-skywalking-apache-org-v1alpha1-javaagent,mutating=true,failurePolicy=fail,groups=operator.skywalking.apache.org,resources=javaagents,verbs=create;update,versions=v1alpha1,name=mjavaagent.kb.io

var _ webhook.Defaulter = &JavaAgent{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *JavaAgent) Default() {
	javaagentlog.Info("default", "name", r.Name)

	config := r.Spec.AgentConfiguration
	if config == nil {
		return
	}

	service := GetServiceName(&config)
	backend := GetBackendService(&config)

	if r.Spec.ServiceName == "" && service != "" {
		r.Spec.ServiceName = service
	}
	if r.Spec.BackendService == "" && backend != "" {
		r.Spec.BackendService = backend
	}
}

// nolint: lll
// +kubebuilder:webhook:admissionReviewVersions=v1,sideEffects=None,verbs=create;update,path=/validate-operator-skywalking-apache-org-v1alpha1-javaagent,mutating=false,failurePolicy=fail,groups=operator.skywalking.apache.org,resources=javaagents,versions=v1alpha1,name=vjavaagent.kb.io

var _ webhook.Validator = &JavaAgent{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *JavaAgent) ValidateCreate() error {
	javaagentlog.Info("validate create", "name", r.Name)
	return r.validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *JavaAgent) ValidateUpdate(old runtime.Object) error {
	javaagentlog.Info("validate update", "name", r.Name)
	return r.validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *JavaAgent) ValidateDelete() error {
	javaagentlog.Info("validate delete", "name", r.Name)
	return nil
}

func (r *JavaAgent) validate() error {
	if r.Spec.ServiceName == "" {
		return fmt.Errorf("service name is absent")
	}
	if r.Spec.BackendService == "" {
		return fmt.Errorf("backend service is absent")
	}
	return nil
}

func GetServiceName(configuration *map[string]string) string {
	v, ok := (*configuration)[ServiceName]
	if !ok {
		return "Your_ApplicationName"
	}
	return v
}

func GetBackendService(configuration *map[string]string) string {
	v, ok := (*configuration)[BackendService]
	if !ok {
		return "127.0.0.1:11800"
	}
	return v
}
