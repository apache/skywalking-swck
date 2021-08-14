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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	labelKeyagentInjector       = "swck-agent-injected"
	defaultConfigmapNamespace   = "skywalking-swck-system"
	agentConfigAnnotationPrefix = "agent.skywalking.apache.org/"
	// If user want to use other Plugins' config ,the annotation must have the following form
	// plugins.skywalking.apache.org/${config.name} = ${config.value}
	// for example , if user want to enable plugin.mongodb.trace_param
	// the annotation is plugins.skywalking.apache.org/plugin.mongodb.trace_param: "true"
	otherPluginsAnnotationPrefix = "plugins.skywalking.apache.org/"
)

// log is for logging in this package.
var log = logf.Log.WithName("injector")

// SidecarInjectField contains all info that will be injected
type SidecarInjectField struct {
	// initcontainer is a container that has the agent folder
	initcontainer corev1.Container
	// sidecarVolume is a shared directory between App's container and initcontainer
	sidecarVolume corev1.Volume
	// sidecarVolumeMount is a path that specifies a shared directory
	sidecarVolumeMount corev1.VolumeMount
	// configmapVolume is volume that provide user with configmap
	configmapVolume corev1.Volume
	// configmapVolumeMount is the configmap's mountpath for user
	// Notice : the mount path will overwrite the original agent/config/agent.config
	// So the mount path must match the path of agent.config in the image
	configmapVolumeMount corev1.VolumeMount
	// env is used to set java agent’s parameters
	env corev1.EnvVar
	// the string is used to set jvm agent ,just like following
	// -javaagent: /sky/agent/skywalking-agent,jar=jvmAgentConfigStr
	jvmAgentConfigStr string
	// determine whether to inject , default is not to inject
	needInject bool
	// determine which container to inject , default is to inject all containers
	injectContainer string
	// determine whether to use annotation to overlay agent config ,
	// default is not to overlay，which means only use configmap to set agent config
	// Otherwise, the way to overlay is set jvm agent ,just like following
	// -javaagent: /sky/agent/skywalking-agent,jar={config1}={value1},{config2}={value2}
	agentOverlay bool
}

// NewSidecarInjectField will create a new SidecarInjectField
func NewSidecarInjectField() *SidecarInjectField {
	return new(SidecarInjectField)
}

// Inject will do real injection
func (s *SidecarInjectField) Inject(pod *corev1.Pod) {
	// add initcontrainers to spec
	if pod.Spec.InitContainers != nil {
		pod.Spec.InitContainers = append(pod.Spec.InitContainers, s.initcontainer)
	} else {
		pod.Spec.InitContainers = []corev1.Container{s.initcontainer}
	}

	// add volume to spec
	if pod.Spec.Volumes != nil {
		pod.Spec.Volumes = append(pod.Spec.Volumes, s.sidecarVolume, s.configmapVolume)
	} else {
		pod.Spec.Volumes = []corev1.Volume{s.sidecarVolume, s.configmapVolume}
	}

	// add volumemount and env to container
	for i, c := range pod.Spec.Containers {
		// choose a specific container to inject
		if s.injectContainer != "" && c.Name == s.injectContainer {
			if c.VolumeMounts != nil {
				pod.Spec.Containers[i].VolumeMounts = append(pod.Spec.Containers[i].VolumeMounts,
					s.sidecarVolumeMount, s.configmapVolumeMount)
			} else {
				pod.Spec.Containers[i].VolumeMounts = []corev1.VolumeMount{s.sidecarVolumeMount,
					s.configmapVolumeMount}
			}
			if c.Env != nil {
				pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env, s.env)
			} else {
				pod.Spec.Containers[i].Env = []corev1.EnvVar{s.env}
			}
			break
		} else {
			if c.VolumeMounts != nil {
				pod.Spec.Containers[i].VolumeMounts = append(pod.Spec.Containers[i].VolumeMounts,
					s.sidecarVolumeMount, s.configmapVolumeMount)
			} else {
				pod.Spec.Containers[i].VolumeMounts = []corev1.VolumeMount{s.sidecarVolumeMount,
					s.configmapVolumeMount}
			}
			if c.Env != nil {
				pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env, s.env)
			} else {
				pod.Spec.Containers[i].Env = []corev1.EnvVar{s.env}
			}
		}
	}
}

// GetInjectStrategy gets user's injection strategy
func (s *SidecarInjectField) GetInjectStrategy(containers []corev1.Container, labels,
	annotation *map[string]string) {
	// set default value
	s.needInject = false
	s.injectContainer = ""
	s.agentOverlay = false

	// set needInject's value , if the pod has the label "swck-agent-injected=true",means need inject
	if *labels == nil {
		labels = new(map[string]string)
		s.needInject = false
	}

	switch strings.ToLower((*labels)[labelKeyagentInjector]) {
	case "true":
		s.needInject = true
	}

	// set injectContainer's value
	if *annotation == nil {
		annotation = new(map[string]string)
		s.injectContainer = ""
		s.agentOverlay = false
	}

	// validate the container is or not exist
	if v, ok := (*annotation)[AnnoInjectContainerName.Name]; ok {
		for _, c := range containers {
			if c.Name == v {
				s.injectContainer = v
				break
			}
		}
	}

	// set agentOverlay's value
	switch strings.ToLower((*annotation)[AnnoAgentConfigOverlay.Name]) {
	case "true":
		s.agentOverlay = true
	}

}

func (s *SidecarInjectField) injectErrorAnnotation(annotation *map[string]string, errorInfo string) {
	(*annotation)[AnnoInjectErrorInfo.Name] = errorInfo
}

// SidecarOverlayandGetValue get final value of sidecar
func (s *SidecarInjectField) SidecarOverlayandGetValue(as *Annotations, annotation *map[string]string,
	a Annotation) (string, bool) {
	if _, ok := (*annotation)[a.Name]; ok {
		err := as.SetOverlay(annotation, a)
		if err != nil {
			s.injectErrorAnnotation(annotation, err.Error())
			return "", false
		}
	}
	return as.GetFinalValue(a), true
}

func (s *SidecarInjectField) setValue(config *string, as *Annotations, annotation *map[string]string,
	a Annotation) bool {
	if v, ok := s.SidecarOverlayandGetValue(as, annotation, a); ok {
		*config = v
		return true
	}
	return false
}

// OverlaySidecar overlays default config
func (s *SidecarInjectField) OverlaySidecar(as *Annotations, annotation *map[string]string) bool {
	var (
		command       string
		argOption     string
		argCommand    string
		configmapName string
	)

	switch {
	case !s.setValue(&s.initcontainer.Name, as, annotation, AnnoInitcontainerName):
		return false
	case !s.setValue(&s.initcontainer.Image, as, annotation, AnnoInitcontainerImage):
		return false
	case !s.setValue(&command, as, annotation, AnnoInitcontainerCommand):
		return false
	case !s.setValue(&argOption, as, annotation, AnnoInitcontainerArgsOption):
		return false
	case !s.setValue(&argCommand, as, annotation, AnnoInitcontainerArgsCommand):
		return false
	case !s.setValue(&s.sidecarVolume.Name, as, annotation, AnnoSidecarVolumeName):
		return false
	case !s.setValue(&s.sidecarVolumeMount.Name, as, annotation, AnnoSidecarVolumeName):
		return false
	case !s.setValue(&s.sidecarVolumeMount.MountPath, as, annotation, AnnoSidecarVolumemountMountpath):
		return false
	case !s.setValue(&configmapName, as, annotation, AnnoConfigmapName):
		return false
	case !s.setValue(&s.configmapVolume.Name, as, annotation, AnnoConfigmapVolumeName):
		return false
	case !s.setValue(&s.configmapVolumeMount.Name, as, annotation, AnnoConfigmapVolumeName):
		return false
	case !s.setValue(&s.configmapVolumeMount.MountPath, as, annotation, AnnoConfigmapVolumemountMountpath):
		return false
	case !s.setValue(&s.env.Name, as, annotation, AnnoEnvVarName):
		return false
	case !s.setValue(&s.env.Value, as, annotation, AnnoEnvVarValue):
		return false
	}

	s.initcontainer.Command = []string{command}
	s.initcontainer.Args = []string{argOption, argCommand}
	s.initcontainer.VolumeMounts = []corev1.VolumeMount{s.sidecarVolumeMount}

	// the sidecar volume's type is determined
	s.sidecarVolume.VolumeSource.EmptyDir = nil

	s.configmapVolume.VolumeSource = corev1.VolumeSource{
		ConfigMap: &corev1.ConfigMapVolumeSource{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: configmapName,
			},
		},
	}

	return true
}

// AgentOverlayandGetValue will do real annotation overlay
func (s *SidecarInjectField) AgentOverlayandGetValue(as *Annotations, annotation *map[string]string,
	a Annotation) (string, bool) {
	if _, ok := (*annotation)[a.Name]; ok {
		err := as.SetOverlay(annotation, a)
		if err != nil {
			s.injectErrorAnnotation(annotation, err.Error())
			return "", false
		}
	}
	return as.GetOverlayValue(a), true
}

func (s *SidecarInjectField) setJvmAgentStr(as *Annotations, annotation *map[string]string, a Annotation,
	specialStr bool) bool {
	v, ok := s.AgentOverlayandGetValue(as, annotation, a)
	if v != "" && ok {
		// get {config1}={value1}
		configName := strings.TrimPrefix(a.Name, agentConfigAnnotationPrefix)
		if specialStr {
			v = strings.Join([]string{"'", "'"}, v)
		}
		config := strings.Join([]string{configName, v}, "=")

		// add to jvmAgentConfigStr
		if s.jvmAgentConfigStr != "" {
			s.jvmAgentConfigStr = strings.Join([]string{s.jvmAgentConfigStr, config}, ",")
		} else {
			s.jvmAgentConfigStr = config
		}
	}
	return ok
}

// OverlayAgentConfig overlays agent config
func (s *SidecarInjectField) OverlayAgentConfig(as *Annotations, annotation *map[string]string) bool {
	// jvmAgentConfigStr init
	s.jvmAgentConfigStr = ""

	switch {
	case !s.setJvmAgentStr(as, annotation, AnnoAgentNamespace, false):
		return false
	case !s.setJvmAgentStr(as, annotation, AnnoAgentServiceName, false):
		return false
	case !s.setJvmAgentStr(as, annotation, AnnoAgentSampleNumber, false):
		return false
	case !s.setJvmAgentStr(as, annotation, AnnoAgentAuthentication, false):
		return false
	case !s.setJvmAgentStr(as, annotation, AnnoAgentSpanLimit, false):
		return false
	case !s.setJvmAgentStr(as, annotation, AnnoAgentIgnoreSuffix, true):
		return false
	case !s.setJvmAgentStr(as, annotation, AnnoAgentIsOpenDebugging, false):
		return false
	case !s.setJvmAgentStr(as, annotation, AnnoAgentIsCacheEnhaned, false):
		return false
	case !s.setJvmAgentStr(as, annotation, AnnoAgentClassCache, false):
		return false
	case !s.setJvmAgentStr(as, annotation, AnnoAgentOperationName, false):
		return false
	case !s.setJvmAgentStr(as, annotation, AnnoAgentForceTLS, false):
		return false
	case !s.setJvmAgentStr(as, annotation, AnnoAgentProfileActive, false):
		return false
	case !s.setJvmAgentStr(as, annotation, AnnoAgentProfileMaxParallel, false):
		return false
	case !s.setJvmAgentStr(as, annotation, AnnoAgentProfileMaxDuration, false):
		return false
	case !s.setJvmAgentStr(as, annotation, AnnoAgentProfileDump, false):
		return false
	case !s.setJvmAgentStr(as, annotation, AnnoAgentProfileSnapshot, false):
		return false
	case !s.setJvmAgentStr(as, annotation, AnnoAgentCollectorService, false):
		return false
	case !s.setJvmAgentStr(as, annotation, AnnoAgentLoggingName, false):
		return false
	case !s.setJvmAgentStr(as, annotation, AnnoAgentLoggingLevel, false):
		return false
	case !s.setJvmAgentStr(as, annotation, AnnoAgentLoggingDir, false):
		return false
	case !s.setJvmAgentStr(as, annotation, AnnoAgentLoggingMaxSize, false):
		return false
	case !s.setJvmAgentStr(as, annotation, AnnoAgentLoggingMaxFiles, false):
		return false
	case !s.setJvmAgentStr(as, annotation, AnnoAgentStatuscheckExceptions, false):
		return false
	case !s.setJvmAgentStr(as, annotation, AnnoAgentStatuscheckDepth, false):
		return false
	case !s.setJvmAgentStr(as, annotation, AnnoAgentPluginMount, true):
		return false
	case !s.setJvmAgentStr(as, annotation, AnnoAgentExcludePlugins, false):
		return false
	case !s.setJvmAgentStr(as, annotation, AnnoAgentPluginJdbc, false):
		return false
	case !s.setJvmAgentStr(as, annotation, AnnoAgentPluginKafkaServers, false):
		return false
	case !s.setJvmAgentStr(as, annotation, AnnoAgentPluginKafkaNamespace, false):
		return false
	case !s.setJvmAgentStr(as, annotation, AnnoAgentPluginSpringannotation, false):
		return false
	}

	return true
}

// OverlayPluginsConfig will add Plugins' config to JvmAgentStr without verification
// Notice, if a config is not in agent.config, it will be seen as a plugin config
// user must ensure the accuracy of configuration.
// Otherwides,If a separator(, or =) in the option or value, it should be wrapped in quotes.
func (s *SidecarInjectField) OverlayPluginsConfig(annotation *map[string]string) bool {
	for k, v := range *annotation {
		if strings.HasPrefix(k, otherPluginsAnnotationPrefix) {
			configName := strings.TrimPrefix(k, otherPluginsAnnotationPrefix)
			config := strings.Join([]string{configName, v}, "=")
			// add to jvmAgentConfigStr
			if s.jvmAgentConfigStr != "" {
				s.jvmAgentConfigStr = strings.Join([]string{s.jvmAgentConfigStr, config}, ",")
			} else {
				s.jvmAgentConfigStr = config
			}
		}
	}
	return true
}

// CreateConfigmap will find a configmap to set agent config
// if not exist , then create a configmap
func (s *SidecarInjectField) CreateConfigmap(ctx context.Context, kubeclient client.Client, namespace string,
	annotation *map[string]string) bool {
	configmap := &corev1.ConfigMap{}
	configmapName := s.configmapVolume.VolumeSource.ConfigMap.LocalObjectReference.Name
	// check whether the configmap is existed
	err := kubeclient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: configmapName}, configmap)
	// if configmap not exist , get configmap from defaultConfigmapNamespace
	if err != nil {
		defaultConfigmap := &corev1.ConfigMap{}
		if err := kubeclient.Get(ctx, client.ObjectKey{Namespace: defaultConfigmapNamespace,
			Name: AnnoConfigmapName.DefaultValue}, defaultConfigmap); err != nil {
			log.Info(err.Error())
			s.injectErrorAnnotation(annotation, fmt.Sprintf("get configmap %s from namespace %s error[%s]",
				AnnoConfigmapName.DefaultValue, defaultConfigmapNamespace, err.Error()))
			return false
		}
		// create new configmap and update namespace
		injectConfigmap := corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      configmapName,
				Namespace: namespace,
			},
			Data: defaultConfigmap.Data,
		}

		if err := kubeclient.Create(ctx, &injectConfigmap); err != nil {
			log.Info(err.Error())
			s.injectErrorAnnotation(annotation, fmt.Sprintf("create configmap %s in namespace %s error[%s]",
				configmapName, namespace, err.Error()))
			return false
		}
	}
	return true
}

// PodInjector injects agent into Pods
type PodInjector struct {
	Client  client.Client
	decoder *admission.Decoder
}

// Handle will process every coming pod under the
// specified namespace which labeled "swck-injection=enabled"
func (r *PodInjector) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}

	err := r.decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	// init Annotations to store the overlaied value
	as := NewAnnotations()

	// init SidecarInjectField and get injected strategy from annotations
	s := NewSidecarInjectField()
	s.GetInjectStrategy(pod.Spec.Containers, &pod.ObjectMeta.Labels, &pod.ObjectMeta.Annotations)

	if s.needInject {
		log.Info("will inject agent,please wait for a moment!")
		// the overlation function is always on
		s.OverlaySidecar(as, &pod.ObjectMeta.Annotations)
		switch {
		// only if user choose to overlay agent config in annotations, then start agent and plugins' overlay
		case s.agentOverlay && (!s.OverlayAgentConfig(as, &pod.ObjectMeta.Annotations) ||
			!s.OverlayPluginsConfig(&pod.ObjectMeta.Annotations)):
			log.Info("overlay agent config error!please look the error annotation!")
			break
		// configmap will always overwrite the original agent.config in image
		case !s.CreateConfigmap(ctx, r.Client, req.Namespace, &pod.ObjectMeta.Annotations):
			log.Info("create configmap error!please look the error annotation!")
			break
		// add jvmStr to EnvVar and do real injection
		default:
			if s.jvmAgentConfigStr != "" {
				s.env.Value = strings.Join([]string{s.env.Value, s.jvmAgentConfigStr}, "=")
			}
			s.Inject(pod)
			log.Info("inject successfully!")
		}
	} else {
		// if the pod don't have the label "swck-agent-injected=true",return ok
		log.Info("don't inject agent!")
		return admission.Allowed("ok")
	}

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
