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

const annotationKeyIstioSetup = "istio-setup-command"

// log is for logging in this package.
var oapserverlog = logf.Log.WithName("oapserver-resource")

func (r *OAPServer) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// nolint: lll
// +kubebuilder:webhook:path=/mutate-operator-skywalking-apache-org-v1alpha1-oapserver,mutating=true,failurePolicy=fail,groups=operator.skywalking.apache.org,resources=oapservers,verbs=create;update,versions=v1alpha1,name=moapserver.kb.io

var _ webhook.Defaulter = &OAPServer{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *OAPServer) Default() {
	oapserverlog.Info("default", "name", r.Name)

	image := r.Spec.Image
	if image == "" {
		v := r.Spec.Version
		vTmpl := "apache/skywalking-oap-server:%s-%s"
		vES := "es6"
		for _, e := range r.Spec.Config {
			if e.Name != "SW_STORAGE" {
				continue
			}
			if e.Value == "elasticsearch7" {
				vES = "es7"
			}
		}
		image = fmt.Sprintf(vTmpl, v, vES)
		r.Spec.Image = image
	}
	for _, envVar := range r.Spec.Config {
		if envVar.Name == "SW_ENVOY_METRIC_ALS_HTTP_ANALYSIS" &&
			r.ObjectMeta.Annotations[annotationKeyIstioSetup] == "" {
			r.Annotations[annotationKeyIstioSetup] = fmt.Sprintf("istioctl install --set profile=demo "+
				"--set meshConfig.defaultConfig.envoyAccessLogService.address=%s.%s:11800 "+
				"--set meshConfig.enableEnvoyAccessLogService=true", r.Name, r.Namespace)
		}
	}
}

// nolint: lll
// +kubebuilder:webhook:verbs=create;update,path=/validate-operator-skywalking-apache-org-v1alpha1-oapserver,mutating=false,failurePolicy=fail,groups=operator.skywalking.apache.org,resources=oapservers,versions=v1alpha1,name=voapserver.kb.io

var _ webhook.Validator = &OAPServer{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *OAPServer) ValidateCreate() error {
	oapserverlog.Info("validate create", "name", r.Name)
	return r.validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *OAPServer) ValidateUpdate(old runtime.Object) error {
	oapserverlog.Info("validate update", "name", r.Name)
	return r.validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *OAPServer) ValidateDelete() error {
	oapserverlog.Info("validate delete", "name", r.Name)
	return nil
}

func (r *OAPServer) validate() error {
	if r.Spec.Image == "" {
		return fmt.Errorf("image is absent")
	}
	return nil
}
