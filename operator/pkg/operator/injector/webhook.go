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

package injector

import (
	"context"
	"net/http"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/apache/skywalking-swck/operator/apis/operator/v1alpha1"
)

// log is for logging in this package.
var javaagentInjectorLog = logf.Log.WithName("javaagent_injector")

// nolint: lll
// +kubebuilder:webhook:admissionReviewVersions=v1,sideEffects=None,path=/mutate-v1-pod,mutating=true,failurePolicy=fail,groups="",resources=pods,verbs=create;update,versions=v1,name=mpod.kb.io

// JavaagentInjector injects java agent into Pods
type JavaagentInjector struct {
	Client  client.Client
	decoder *admission.Decoder
}

// Handle will process every coming pod under the
// specified namespace which labeled "swck-injection=enabled"
func (r *JavaagentInjector) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}

	if err := r.decoder.Decode(req, pod); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	// set Annotations to avoid repeated judgments
	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}

	swAgentL := r.findMatchedSwAgentL(ctx, req, pod)

	// initialize all annotation types that can be overridden
	anno, err := NewAnnotations()
	if err != nil {
		javaagentInjectorLog.Error(err, "get NewAnnotations error")
	}
	// initialize Annotations to store the overlaied value
	ao := NewAnnotationOverlay()
	// initialize SidecarInjectField and get injected strategy from annotations
	s := NewSidecarInjectField()
	// initialize InjectProcess as a call chain
	ip := NewInjectProcess(ctx, s, anno, ao, swAgentL, pod, req, javaagentInjectorLog, r.Client)
	// do real injection
	return ip.Run()
}

func (r *JavaagentInjector) findMatchedSwAgentL(ctx context.Context, req admission.Request, pod *corev1.Pod) *v1alpha1.SwAgentList {
	swAgentList := &v1alpha1.SwAgentList{}
	if err := r.Client.List(ctx, swAgentList, client.InNamespace(req.Namespace)); err != nil {
		javaagentInjectorLog.Error(err, "get SwAgent error")
	}

	// selector
	var availableSwAgentL []v1alpha1.SwAgent
	for _, swAgent := range swAgentList.Items {
		isMatch := true
		if len(swAgent.Spec.Selector) != 0 {
			for k, v := range swAgent.Spec.Selector {
				if !strings.EqualFold(v, pod.Labels[k]) {
					isMatch = false
				}
			}
		}
		if isMatch {
			availableSwAgentL = append(availableSwAgentL, swAgent)
		}
	}
	swAgentList.Items = availableSwAgentL
	return swAgentList
}

// Javaagent implements admission.DecoderInjector.
// A decoder will be automatically injected.

// InjectDecoder injects the decoder.
func (r *JavaagentInjector) InjectDecoder(d *admission.Decoder) error {
	r.decoder = d
	return nil
}
