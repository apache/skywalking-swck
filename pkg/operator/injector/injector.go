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
	"encoding/json"
	"net/http"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	annotationKeyagentInjector = "swck-agent-injected"
)

// PodInjector injects agent into Pods
type PodInjector struct {
	Client  client.Client
	decoder *admission.Decoder
}

// PodInjector will process every coming pod under the
// specified namespace which labeled "swck-injection=enabled"
func (r *PodInjector) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}
	PodInjectorLog := logf.Log.WithName("PodInjector")
	err := r.decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	//if the pod don't have the label "swck-agent-injected=true",return ok
	if !needInject(pod) {
		PodInjectorLog.Info("don't inject agent")
		return admission.Allowed("ok")
	}

	//if the pod has the label "swck-agent-injected=true",add agent
	addAgent(pod)
	PodInjectorLog.Info("will inject agent,please wait for a moment!")

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

// PodInjector implements admission.DecoderInjector.
// A decoder will be automatically injected.

// InjectDecoder injects the decoder.
func (r *PodInjector) InjectDecoder(d *admission.Decoder) error {
	r.decoder = d
	return nil
}

//if pod has label "swck-agent-injected=true" , it means the pod needs agent injected
func needInject(pod *corev1.Pod) bool {
	injected := false

	labels := pod.ObjectMeta.Labels
	if labels == nil {
		return injected
	}

	switch strings.ToLower(labels[annotationKeyagentInjector]) {
	case "true":
		injected = true
	}

	return injected
}

//when agent is injected,we need to push agent image and
//  mount the volume to the specified directory
func addAgent(pod *corev1.Pod) {
	// set InitContainer's VolumeMount
	vm := corev1.VolumeMount{
		MountPath: "/sky/agent",
		Name:      "sky-agent",
	}

	// set the agent image to be injected
	needAddInitContainer := corev1.Container{
		Name:         "inject-sky-agent",
		Image:        "apache/skywalking-java-agent:8.6.0-jdk8",
		Command:      []string{"sh"},
		Args:         []string{"-c", "mkdir -p /sky/agent && cp -r /skywalking/agent/* /sky/agent"},
		VolumeMounts: []corev1.VolumeMount{vm},
	}

	// set emptyDir Volume
	needAddVolumes := corev1.Volume{
		Name:         "sky-agent",
		VolumeSource: corev1.VolumeSource{EmptyDir: nil},
	}

	// set container's VolumeMount
	needAddVolumeMount := corev1.VolumeMount{
		MountPath: "/sky/agent",
		Name:      "sky-agent",
	}

	// set container's EnvVar
	needAddEnv := corev1.EnvVar{
		Name:  "AGENT_OPTS",
		Value: " -javaagent:/sky/agent/skywalking-agent.jar",
	}

	// add VolumeMount to spec
	if pod.Spec.Volumes != nil {
		pod.Spec.Volumes = append(pod.Spec.Volumes, needAddVolumes)
	} else {
		pod.Spec.Volumes = []corev1.Volume{needAddVolumes}
	}

	// add InitContrainers to spec
	if pod.Spec.InitContainers != nil {
		pod.Spec.InitContainers = append(pod.Spec.InitContainers, needAddInitContainer)
	} else {
		pod.Spec.InitContainers = []corev1.Container{needAddInitContainer}
	}

	// add VolumeMount and env to every container
	for i := 0; i < len(pod.Spec.Containers); i++ {
		pod.Spec.Containers[i].VolumeMounts = append(pod.Spec.Containers[i].VolumeMounts, needAddVolumeMount)
		pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env, needAddEnv)
	}
}
