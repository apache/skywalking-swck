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
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var swagentlog = logf.Log.WithName("swagent-resource")

func (r *SwAgent) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// nolint: lll
//+kubebuilder:webhook:path=/mutate-operator-skywalking-apache-org-v1alpha1-swagent,mutating=true,failurePolicy=fail,sideEffects=None,groups=operator.skywalking.apache.org,resources=swagents,verbs=create;update,versions=v1alpha1,name=mswagent.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &SwAgent{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *SwAgent) Default() {
	swagentlog.Info("default", "name", r.Name)
	r.setDefault()
}

// nolint: lll
//+kubebuilder:webhook:path=/validate-operator-skywalking-apache-org-v1alpha1-swagent,mutating=false,failurePolicy=fail,sideEffects=None,groups=operator.skywalking.apache.org,resources=swagents,verbs=create;update,versions=v1alpha1,name=vswagent.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &SwAgent{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *SwAgent) ValidateCreate() error {
	swagentlog.Info("validate create", "name", r.Name)
	r.setDefault()
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *SwAgent) ValidateUpdate(_ runtime.Object) error {
	swagentlog.Info("validate update", "name", r.Name)
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *SwAgent) ValidateDelete() error {
	swagentlog.Info("validate delete", "name", r.Name)
	return nil
}

const (
	LabelJavaAgent = "swck-java-agent-injected"
)

func (r *SwAgent) setDefault() {
	if nil != r {
		if len(r.Spec.Selector) == 0 {
			if r.Spec.Selector == nil {
				r.Spec.Selector = make(map[string]string)
			}
			r.Spec.Selector[LabelJavaAgent] = "true"
		}
		if len(r.Spec.ContainerMatcher) == 0 {
			r.Spec.ContainerMatcher = ".*"
		}

		// default values for java sidecar
		if len(r.Spec.JavaSidecar.Name) == 0 {
			r.Spec.JavaSidecar.Name = "inject-skywalking-agent"
		}
		if len(r.Spec.JavaSidecar.Image) == 0 {
			r.Spec.JavaSidecar.Image = "apache/skywalking-java-agent:8.16.0-java8"
		}
		if len(r.Spec.JavaSidecar.Command) == 0 {
			if r.Spec.JavaSidecar.Command == nil {
				r.Spec.JavaSidecar.Command = []string{}
			}
			r.Spec.JavaSidecar.Command = append(r.Spec.JavaSidecar.Command, "sh")
		}
		if len(r.Spec.JavaSidecar.Args) == 0 {
			if r.Spec.JavaSidecar.Args == nil {
				r.Spec.JavaSidecar.Args = []string{}
			}
			r.Spec.JavaSidecar.Args = append(r.Spec.JavaSidecar.Args, "-c")
			r.Spec.JavaSidecar.Args = append(r.Spec.JavaSidecar.Args, "mkdir -p /sky/agent && cp -r /skywalking/agent/* /sky/agent")
		}
		r.setOrAppendEnv("JAVA_TOOL_OPTIONS", " -javaagent:/sky/agent/skywalking-agent.jar")

		// default values for shared volume
		if len(r.Spec.SharedVolumeName) == 0 {
			r.Spec.SharedVolumeName = "sky-agent"
		}

	}
}

func (r *SwAgent) setOrAppendEnv(envKey string, envValue string) {
	if !r.appendEnvIfExists(&r.Spec.JavaSidecar.Env, envKey, envValue) {
		javaToolOptionsEnv := corev1.EnvVar{
			Name:  envKey,
			Value: envValue,
		}
		r.Spec.JavaSidecar.Env = append(r.Spec.JavaSidecar.Env, javaToolOptionsEnv)
	}
}

func (r *SwAgent) appendEnvIfExists(envs *[]corev1.EnvVar, envKey string, envValue string) bool {
	for _, env := range *envs {
		if strings.EqualFold(env.Name, envKey) {
			env.Value += envValue
			return true
		}
	}
	return false
}
