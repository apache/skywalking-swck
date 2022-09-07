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
	"fmt"
	"net/http"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/apache/skywalking-swck/operator/apis/operator/v1alpha1"
)

// Injection is each step of the injection process
type Injection interface {
	execute(*InjectProcessData) admission.Response
	// set next step
	setNext(Injection)
}

// InjectProcessData defines data that needs to be processed in the injection process
// Divide the entire injection process into 6 steps
// 1.Get injection strategy
// 2.Overlay the sidecar info
// 3.Overlay the agent by setting jvm string
// 4.Overlay the plugins by setting jvm string and set the optional plugins
// 5.Get the ConfigMap
// 6.Inject fields into Pod
// After all steps are completed, return fully injected Pod, Or there is an error
// in a certain step, inject error info into annotations and return incompletely injected Pod
type InjectProcessData struct {
	injectFileds      *SidecarInjectField
	annotation        *Annotations
	annotationOverlay *AnnotationOverlay
	swAgentL          *v1alpha1.SwAgentList
	pod               *corev1.Pod
	req               admission.Request
	log               logr.Logger
	kubeclient        client.Client
	ctx               context.Context
}

// PatchReq is to fill the injected pod into the request and return the Response
func PatchReq(pod *corev1.Pod, req admission.Request) admission.Response {
	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

// GetStrategy is the first step of injection process
type GetStrategy struct {
	next Injection
}

// get injection strategy from pod's labels and annotations
// if don't need to inject, then return the original pod, otherwise go to the next step
func (gs *GetStrategy) execute(ipd *InjectProcessData) admission.Response {
	log.Info("=============== GetStrategy ================")
	ipd.injectFileds.GetInjectStrategy(*ipd.annotation, &ipd.pod.ObjectMeta.Labels, &ipd.pod.ObjectMeta.Annotations)
	if !ipd.injectFileds.NeedInject {
		log.Info("don't inject agent")
		return admission.Allowed("ok")
	}
	return gs.next.execute(ipd)
}

func (gs *GetStrategy) setNext(next Injection) {
	gs.next = next
}

// OverlaySwAgentCR is the cr config step of injection process
type OverlaySwAgentCR struct {
	next Injection
}

// get configs from SwAgent CR
func (gs *OverlaySwAgentCR) execute(ipd *InjectProcessData) admission.Response {
	log.Info(fmt.Sprintf("=============== OverlaySwAgentCR(%d) ================ ", len(ipd.swAgentL.Items)))
	if !ipd.injectFileds.OverlaySwAgentCR(ipd.swAgentL, ipd.pod) {
		log.Info("overlay SwAgent cr config error.")
		return PatchReq(ipd.pod, ipd.req)
	}
	return gs.next.execute(ipd)
}

func (gs *OverlaySwAgentCR) setNext(next Injection) {
	gs.next = next
}

// OverlaySidecar is the second step of injection process
type OverlaySidecar struct {
	next Injection
}

// OverlaySidecar will inject sidecar information such as image, command, args etc.
// Since we did not set the validation function for these fields, it usually goes to the next step
func (os *OverlaySidecar) execute(ipd *InjectProcessData) admission.Response {
	log.Info("=============== OverlaySidecar ================")
	if !ipd.injectFileds.OverlaySidecar(*ipd.annotation, ipd.annotationOverlay, &ipd.pod.ObjectMeta.Annotations) {
		return PatchReq(ipd.pod, ipd.req)
	}
	return os.next.execute(ipd)
}

func (os *OverlaySidecar) setNext(next Injection) {
	os.next = next
}

// OverlayAgent is the third step of injection process
type OverlayAgent struct {
	next Injection
}

// OverlayAgent overlays the agent by getting the pod's annotations
// If the agent overlay option is not set, go directly to the next step
// If set the wrong value in the annotation , inject the error info and return
func (oa *OverlayAgent) execute(ipd *InjectProcessData) admission.Response {
	log.Info("=============== OverlayAgent ================")
	if !ipd.injectFileds.OverlayAgent(*ipd.annotation, ipd.annotationOverlay, &ipd.pod.ObjectMeta.Annotations) {
		ipd.log.Info("overlay agent config error!please look the error annotation!")
		return PatchReq(ipd.pod, ipd.req)
	}
	return oa.next.execute(ipd)
}

func (oa *OverlayAgent) setNext(next Injection) {
	oa.next = next
}

// OverlayPlugins is the fourth step of injection process
type OverlayPlugins struct {
	next Injection
}

// OverlayPlugins contains two step , the first is to set jvm string , the second is to set optional plugins
// during the step , we need to add jvm string to the Env of injected container
func (op *OverlayPlugins) execute(ipd *InjectProcessData) admission.Response {
	log.Info("=============== OverlayPlugins ================")
	ipd.injectFileds.OverlayPlugins(&ipd.pod.ObjectMeta.Annotations)
	if ipd.injectFileds.JvmAgentConfigStr != "" {
		ipd.injectFileds.Env.Value = strings.Join([]string{ipd.injectFileds.Env.Value, ipd.injectFileds.JvmAgentConfigStr}, "=")
	}
	ipd.injectFileds.OverlayOptional(ipd.swAgentL.Items, &ipd.pod.ObjectMeta.Annotations)
	return op.next.execute(ipd)
}
func (op *OverlayPlugins) setNext(next Injection) {
	op.next = next
}

// GetConfigmap is the fifth step of injection process
type GetConfigmap struct {
	next Injection
}

// GetConfigmap will get the configmap specified in the annotation. if not exist ,
// we will get data from default configmap and create configmap in the req's namespace
func (gc *GetConfigmap) execute(ipd *InjectProcessData) admission.Response {
	log.Info("=============== GetConfigmap ================")
	if !ipd.injectFileds.ValidateConfigmap(ipd.ctx, ipd.kubeclient, ipd.req.Namespace, &ipd.pod.ObjectMeta.Annotations) {
		return PatchReq(ipd.pod, ipd.req)
	}
	return gc.next.execute(ipd)
}
func (gc *GetConfigmap) setNext(next Injection) {
	gc.next = next
}

// PodInject is the sixth step of injection process
type PodInject struct {
	next Injection
}

// PodInject will inject all fields to the pod
func (pi *PodInject) execute(ipd *InjectProcessData) admission.Response {
	log.Info("=============== PodInject ================")
	ipd.injectFileds.Inject(ipd.pod)
	ipd.injectFileds.injectSucceedAnnotation(&ipd.pod.Annotations)
	log.Info("inject successfully!")
	return PatchReq(ipd.pod, ipd.req)
}
func (pi *PodInject) setNext(next Injection) {
	pi.next = next
}

// NewInjectProcess create a new InjectProcess
func NewInjectProcess(ctx context.Context, injectFileds *SidecarInjectField, annotation *Annotations,
	annotationOverlay *AnnotationOverlay, swAgentL *v1alpha1.SwAgentList, pod *corev1.Pod, req admission.Request, log logr.Logger,
	kubeclient client.Client) *InjectProcessData {
	return &InjectProcessData{
		ctx:               ctx,
		injectFileds:      injectFileds,
		annotation:        annotation,
		annotationOverlay: annotationOverlay,
		swAgentL:          swAgentL,
		pod:               pod,
		req:               req,
		log:               log,
		kubeclient:        kubeclient,
	}
}

// Run will connect the above six steps into a chain and start to execute the first step
func (ipd *InjectProcessData) Run() admission.Response {
	// set final step
	podInject := &PodInject{}

	// set next step is PodInject
	getConfigmap := &GetConfigmap{}
	getConfigmap.setNext(podInject)

	// set next step is GetConfigmap
	overlayPlugins := &OverlayPlugins{}
	overlayPlugins.setNext(getConfigmap)

	// set next step is OverlayPlugins
	overlayAgent := &OverlayAgent{}
	overlayAgent.setNext(overlayPlugins)

	// set next step is OverlayAgent
	overlaysidecar := &OverlaySidecar{}
	overlaysidecar.setNext(overlayAgent)

	// set next step is OverlaySwAgentCR
	overlaySwAgentCR := &OverlaySwAgentCR{}
	overlaySwAgentCR.setNext(overlaysidecar)

	// set next step is OverlaySidecar
	getStrategy := &GetStrategy{}
	getStrategy.setNext(overlaySwAgentCR)

	// this is first step and do real injection
	return getStrategy.execute(ipd)
}
