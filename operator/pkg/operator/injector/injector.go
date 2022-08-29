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
	"regexp"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/apache/skywalking-swck/operator/apis/operator/v1alpha1"
)

const (
	// the label means whether to enbale injection , "true" or "false".
	ActiveInjectorLabel = "swck-java-agent-injected"
	// SidecarInjectSucceedAnno represents injection succeed
	SidecarInjectSucceedAnno = "sidecar.skywalking.apache.org/succeed"
	// the annotation means which container to inject
	sidecarInjectContainerAnno = "strategy.skywalking.apache.org/inject.Container"
	// the annotation that specify the reason for injection failure
	sidecarInjectErrorAnno = "sidecar.skywalking.apache.org/error"
	// those annotations with the following prefixes represent sidecar information
	sidecarAnnotationPrefix = "sidecar.skywalking.apache.org/"
	// those annotations with the following prefixes represent agent information
	agentAnnotationPrefix = "agent.skywalking.apache.org/"
	// If user want to use other Plugins' config ,the annotation must have the following form
	// plugins.skywalking.apache.org/${config.name} = ${config.value}
	// for example , if user want to enable plugin.mongodb.trace_param
	// the annotation is plugins.skywalking.apache.org/plugin.mongodb.trace_param: "true"
	pluginsAnnotationPrefix = "plugins.skywalking.apache.org/"
	// If user want to use optional-plugins , the annotation must match a optinal plugin
	// such as optional.skywalking.apache.org: "trace|webflux|cloud-gateway-2.1.x"
	// Notice , If the injected container's image don't has the optional plugin ,
	// the container will panic
	optionsAnnotation = "optional.skywalking.apache.org"
	// If user want to use optional-reporter-plugins , the annotation must match a optinal-reporter plugin
	// such as optional-exporter.skywalking.apache.org: "kafka"
	optionsReporterAnnotation = "optional-reporter.skywalking.apache.org"
	// the mount path of empty volume, which shared by init container and target container.
	mountPath = "/sky/agent"
)

// log is for logging in this package.
var log = logf.Log.WithName("injector")

// SidecarInjectField contains all info that will be injected
type SidecarInjectField struct {
	// determine whether to inject , default is not to inject
	NeedInject bool
	// Initcontainer is a container that has the agent folder
	Initcontainer corev1.Container
	// sidecarVolume is a shared directory between App's container and initcontainer
	SidecarVolume corev1.Volume
	// sidecarVolumeMount is a path that specifies a shared directory
	SidecarVolumeMount corev1.VolumeMount
	// configmapVolume is volume that provide user with configmap
	ConfigmapVolume corev1.Volume
	// configmapVolumeMount is the configmap's mountpath for user
	// Notice : the mount path will overwrite the original agent/config/agent.config
	// So the mount path must match the path of agent.config in the image
	ConfigmapVolumeMount corev1.VolumeMount
	// env is used to set java agent’s parameters
	Env corev1.EnvVar
	// envs is the envs pass to target containers
	Envs []corev1.EnvVar
	// the string is used to set jvm agent ,just like following
	// -javaagent: /sky/agent/skywalking-agent,jar=jvmAgentConfigStr
	JvmAgentConfigStr string
	// determine which container to inject , default is to inject all containers
	InjectContainer string
}

// NewSidecarInjectField will create a new SidecarInjectField
func NewSidecarInjectField() *SidecarInjectField {
	return new(SidecarInjectField)
}

// Inject will do real injection
func (s *SidecarInjectField) Inject(pod *corev1.Pod) {
	log.Info(fmt.Sprintf("inject pod : %s", pod.GenerateName))
	// add initcontrainers to spec
	if pod.Spec.InitContainers != nil {
		pod.Spec.InitContainers = append(pod.Spec.InitContainers, s.Initcontainer)
	} else {
		pod.Spec.InitContainers = []corev1.Container{s.Initcontainer}
	}

	// add volume to spec
	if pod.Spec.Volumes == nil {
		pod.Spec.Volumes = []corev1.Volume{}
	}
	pod.Spec.Volumes = append(pod.Spec.Volumes, s.SidecarVolume)
	if len(s.ConfigmapVolume.Name) > 0 && len(s.ConfigmapVolume.ConfigMap.Name) > 0 {
		pod.Spec.Volumes = append(pod.Spec.Volumes, s.ConfigmapVolume)
	}

	// choose a specific container to inject
	targetContainers := s.findInjectContainer(pod.Spec.Containers)

	// add volumemount and env to container
	for i := range targetContainers {
		log.Info(fmt.Sprintf("inject container : %s", targetContainers[i].Name))
		if (*targetContainers[i]).VolumeMounts == nil {
			(*targetContainers[i]).VolumeMounts = []corev1.VolumeMount{}
		}

		(*targetContainers[i]).VolumeMounts = append((*targetContainers[i]).VolumeMounts, s.SidecarVolumeMount)
		if len(s.ConfigmapVolumeMount.Name) > 0 && len(s.ConfigmapVolumeMount.MountPath) > 0 {
			(*targetContainers[i]).VolumeMounts = append((*targetContainers[i]).VolumeMounts, s.ConfigmapVolumeMount)
		}

		if (*targetContainers[i]).Env != nil {
			(*targetContainers[i]).Env = append((*targetContainers[i]).Env, s.Env)
		} else {
			(*targetContainers[i]).Env = []corev1.EnvVar{s.Env}
		}

		// envs to be append
		var envsTBA []corev1.EnvVar
		for j, envInject := range s.Envs {
			isExists := false
			for _, envExists := range targetContainers[i].Env {
				if strings.EqualFold(envExists.Name, envInject.Name) {
					isExists = true
					break
				}
			}
			if !isExists {
				envsTBA = append(envsTBA, s.Envs[j])
			}
		}
		if len(s.Envs) > 0 {
			(*targetContainers[i]).Env = append((*targetContainers[i]).Env, envsTBA...)
		}
	}
}

// GetInjectStrategy gets user's injection strategy
func (s *SidecarInjectField) GetInjectStrategy(a Annotations, labels,
	annotation *map[string]string) {
	// set default value
	s.NeedInject = false

	// set NeedInject's value , if the pod has the label "swck-java-agent-injected=true", means need inject
	if *labels == nil {
		return
	}

	if strings.EqualFold((*labels)[ActiveInjectorLabel], "true") {
		s.NeedInject = true
	}

	if *annotation == nil {
		return
	}

	// set injectContainer's value
	if v, ok := (*annotation)[sidecarInjectContainerAnno]; ok {
		s.InjectContainer = v
	}
}

func (s *SidecarInjectField) findInjectContainer(containers []corev1.Container) []*corev1.Container {

	var targetContainers []*corev1.Container

	if len(containers) == 0 {
		return targetContainers
	}

	// validate the container is or not exist
	if len(s.InjectContainer) == 0 {
		for i := range containers {
			targetContainers = append(targetContainers, &containers[i])
		}
		return targetContainers
	}

	containerMatcher, err := regexp.Compile(s.InjectContainer)
	if err != nil {
		log.Error(err, fmt.Sprintf("failed to compile regular expression: %s", s.InjectContainer))
		targetContainers = append(targetContainers, &containers[0])
		return targetContainers
	}
	for i := range containers {
		if containerMatcher.MatchString(containers[i].Name) {
			log.Info(fmt.Sprintf("container %s matched. ", containers[i].Name))
			targetContainers = append(targetContainers, &containers[i])
		}
	}
	return targetContainers
}

func (s *SidecarInjectField) injectErrorAnnotation(annotation *map[string]string, errorInfo string) {
	(*annotation)[sidecarInjectErrorAnno] = errorInfo
}

func (s *SidecarInjectField) injectSucceedAnnotation(annotation *map[string]string) {
	(*annotation)[SidecarInjectSucceedAnno] = "true"
	(*annotation)[ActiveInjectorLabel] = "true"
}

// SidecarOverlayandGetValue get final value of sidecar
func (s *SidecarInjectField) SidecarOverlayandGetValue(ao *AnnotationOverlay, annotation *map[string]string,
	a Annotation) (string, bool) {
	if _, ok := (*annotation)[a.Name]; ok {
		err := ao.SetOverlay(annotation, a)
		if err != nil {
			s.injectErrorAnnotation(annotation, err.Error())
			return "", false
		}
	}
	return (*ao)[a], true
}

// annotation > swAgent > default
func (s *SidecarInjectField) setValue(config *string, ao *AnnotationOverlay, annotation *map[string]string,
	a Annotation) bool {
	if v, ok := s.SidecarOverlayandGetValue(ao, annotation, a); ok {
		if len(v) > 0 {
			*config = v
			return true
		} else if len(*config) > 0 {
			return true
		} else {
			*config = a.DefaultValue
			return true
		}
	}
	return false
}

func (s *SidecarInjectField) OverlaySwAgentCR(swAgentL *v1alpha1.SwAgentList, pod *corev1.Pod) bool {
	s.ConfigmapVolume.ConfigMap = new(corev1.ConfigMapVolumeSource)

	// chose the last matched SwAgent
	if len(swAgentL.Items) > 0 {
		swAgent := swAgentL.Items[len(swAgentL.Items)-1]
		log.Info(fmt.Sprintf("agent %s loaded.", swAgent.Name))
		// shared volume, mount path is fixed
		s.SidecarVolume.Name = swAgent.Spec.SharedVolumeName
		s.SidecarVolume.VolumeSource.EmptyDir = &corev1.EmptyDirVolumeSource{}
		s.SidecarVolumeMount.Name = swAgent.Spec.SharedVolumeName
		s.SidecarVolumeMount.MountPath = mountPath

		// agent configmap
		if swAgent.Spec.SwConfigMapVolume != nil {
			if len(swAgent.Spec.SwConfigMapVolume.Name) > 0 &&
				len(swAgent.Spec.SwConfigMapVolume.ConfigMapName) > 0 &&
				len(swAgent.Spec.SwConfigMapVolume.ConfigMapMountFile) > 0 {
				//s.ConfigmapVolume = corev1.Volume{}
				s.ConfigmapVolume.Name = swAgent.Spec.SwConfigMapVolume.Name
				s.ConfigmapVolume.ConfigMap = new(corev1.ConfigMapVolumeSource)
				s.ConfigmapVolume.ConfigMap.Name = swAgent.Spec.SwConfigMapVolume.ConfigMapName
				//s.ConfigmapVolumeMount = corev1.VolumeMount{}
				s.ConfigmapVolumeMount.Name = swAgent.Spec.SwConfigMapVolume.Name
				s.ConfigmapVolumeMount.MountPath = "/sky/agent/config/" + swAgent.Spec.SwConfigMapVolume.ConfigMapMountFile
				s.ConfigmapVolumeMount.SubPath = swAgent.Spec.SwConfigMapVolume.ConfigMapMountFile
			}
		}

		// init container
		s.Initcontainer.Name = swAgent.Spec.JavaSidecar.Name
		s.Initcontainer.Image = swAgent.Spec.JavaSidecar.Image
		s.Initcontainer.Args = swAgent.Spec.JavaSidecar.Args
		s.Initcontainer.Command = swAgent.Spec.JavaSidecar.Command
		s.Initcontainer.VolumeMounts = append(s.Initcontainer.VolumeMounts, corev1.VolumeMount{
			Name:      swAgent.Spec.SharedVolumeName,
			MountPath: mountPath,
		})

		// target container
		s.Envs = swAgent.Spec.JavaSidecar.Env
		s.InjectContainer = swAgent.Spec.ContainerMatcher
	}

	return true
}

// OverlaySidecar overlays default config
func (s *SidecarInjectField) OverlaySidecar(a Annotations, ao *AnnotationOverlay, annotation *map[string]string) bool {
	s.Initcontainer.Command = make([]string, 1)
	s.Initcontainer.Args = make([]string, 2)
	if nil == s.ConfigmapVolume.ConfigMap {
		s.ConfigmapVolume.ConfigMap = new(corev1.ConfigMapVolumeSource)
	}

	limitsStr := ""
	requestStr := ""
	// every annotation map a pointer to the field of SidecarInjectField
	annoField := map[string]*string{
		"initcontainer.Name":               &s.Initcontainer.Name,
		"initcontainer.Image":              &s.Initcontainer.Image,
		"initcontainer.Command":            &s.Initcontainer.Command[0],
		"initcontainer.args.Option":        &s.Initcontainer.Args[0],
		"initcontainer.args.Command":       &s.Initcontainer.Args[1],
		"initcontainer.resources.limits":   &limitsStr,
		"initcontainer.resources.requests": &requestStr,
		"sidecarVolume.Name":               &s.SidecarVolume.Name,
		"sidecarVolumeMount.MountPath":     &s.SidecarVolumeMount.MountPath,
		"configmapVolume.ConfigMap.Name":   &s.ConfigmapVolume.ConfigMap.Name,
		"configmapVolume.Name":             &s.ConfigmapVolume.Name,
		"configmapVolumeMount.MountPath":   &s.ConfigmapVolumeMount.MountPath,
		"env.Name":                         &s.Env.Name,
		"env.Value":                        &s.Env.Value,
	}

	anno := GetAnnotationsByPrefix(a, sidecarAnnotationPrefix)
	for _, v := range anno.Annotations {
		fieldName := strings.TrimPrefix(v.Name, sidecarAnnotationPrefix)
		if pointer, ok := annoField[fieldName]; ok {
			if !s.setValue(pointer, ao, annotation, v) {
				return false
			}
		}
	}

	s.SidecarVolumeMount.Name = s.SidecarVolume.Name
	s.ConfigmapVolumeMount.Name = s.ConfigmapVolume.Name
	s.Initcontainer.VolumeMounts = []corev1.VolumeMount{s.SidecarVolumeMount}

	// add requests and limits to initcontainer
	if limitsStr != "nil" {
		limits := make(corev1.ResourceList)
		err := json.Unmarshal([]byte(limitsStr), &limits)
		if err != nil {
			log.Error(err, "unmarshal limitsStr error")
			return false
		}
		s.Initcontainer.Resources.Limits = limits
	}

	if requestStr != "nil" {
		requests := make(corev1.ResourceList)
		err := json.Unmarshal([]byte(requestStr), &requests)
		if err != nil {
			log.Error(err, "unmarshal requestStr error")
			return false
		}
		s.Initcontainer.Resources.Requests = requests
	}

	// the sidecar volume's type is determined
	s.SidecarVolume.VolumeSource.EmptyDir = nil

	return true
}

// AgentOverlayandGetValue will do real annotation overlay
func (s *SidecarInjectField) AgentOverlayandGetValue(ao *AnnotationOverlay, annotation *map[string]string,
	a Annotation) bool {
	if _, ok := (*annotation)[a.Name]; ok {
		err := ao.SetOverlay(annotation, a)
		if err != nil {
			s.injectErrorAnnotation(annotation, err.Error())
			return false
		}
	}
	return true
}

// OverlayAgent overlays agent
func (s *SidecarInjectField) OverlayAgent(a Annotations, ao *AnnotationOverlay, annotation *map[string]string) bool {
	// jvmAgentConfigStr init
	s.JvmAgentConfigStr = ""
	anno := GetAnnotationsByPrefix(a, agentAnnotationPrefix)
	for k, v := range *annotation {
		if strings.HasPrefix(k, agentAnnotationPrefix) {
			for _, an := range anno.Annotations {
				if strings.EqualFold(k, an.Name) {
					if !s.AgentOverlayandGetValue(ao, annotation, an) {
						return false
					}
				}
			}
			configName := strings.TrimPrefix(k, agentAnnotationPrefix)
			config := strings.Join([]string{configName, v}, "=")
			// add to jvmAgentConfigStr
			if s.JvmAgentConfigStr != "" {
				s.JvmAgentConfigStr = strings.Join([]string{s.JvmAgentConfigStr, config}, ",")
			} else {
				s.JvmAgentConfigStr = config
			}
		}
	}
	return true
}

// OverlayOptional overlays optional plugins and move optional plugins to the directory(/plugins)
// user must ensure that the optional plugins are in the injected container's image
// Notice , user must specify the correctness of the regular value
// such as optional.skywalking.apache.org: "trace|webflux|cloud-gateway-2.1.x" or
// optional-reporter.skywalking.apache.org: "kafka"
// the final command will be "cd /optional-plugins && ls | grep -E "trace|webflux|cloud-gateway-2.1.x" | xargs -i cp {}  /plugins
// or "cd /optional-reporter-plugins && ls | grep -E "kafka" | xargs -i cp {}  /plugins"
func (s *SidecarInjectField) OverlayOptional(swAgentL []v1alpha1.SwAgent, annotation *map[string]string) {
	sourceOptionalPath := strings.Join([]string{s.SidecarVolumeMount.MountPath, "optional-plugins/"}, "/")
	sourceOptionalReporterPath := strings.Join([]string{s.SidecarVolumeMount.MountPath, "optional-reporter-plugins/"}, "/")
	targetPath := strings.Join([]string{s.SidecarVolumeMount.MountPath, "plugins/"}, "/")

	command := ""
	optionalPlugin := ""
	optinalReporterPlugins := ""

	// chose the last matched SwAgent
	if len(swAgentL) > 0 {
		swAgent := swAgentL[len(swAgentL)-1]
		log.Info(fmt.Sprintf("agent %s loaded.", swAgent.Name))
		if len(swAgent.Spec.OptionalPlugins) > 0 {
			optionalPlugin = strings.Join(swAgent.Spec.OptionalPlugins, "|")
		}
		if len(swAgent.Spec.OptionalReporterPlugins) > 0 {
			optinalReporterPlugins = strings.Join(swAgent.Spec.OptionalReporterPlugins, "|")
		}
	}

	for k, v := range *annotation {
		if strings.EqualFold(k, optionsAnnotation) {
			optionalPlugin = v
		} else if strings.EqualFold(k, optionsReporterAnnotation) {
			optinalReporterPlugins = v
		}
		if command != "" {
			s.Initcontainer.Args[1] = strings.Join([]string{s.Initcontainer.Args[1], command}, " && ")
		}
	}

	if len(optionalPlugin) > 0 {
		command = "cd " + sourceOptionalPath + " && ls | grep -E \"" + optionalPlugin + "\"  | xargs -i cp {} " + targetPath
		s.Initcontainer.Args[1] = strings.Join([]string{s.Initcontainer.Args[1], command}, " && ")
	}

	if len(optinalReporterPlugins) > 0 {
		command = "cd " + sourceOptionalReporterPath + " && ls | grep -E \"" + optinalReporterPlugins + "\"  | xargs -i cp {} " + targetPath
		s.Initcontainer.Args[1] = strings.Join([]string{s.Initcontainer.Args[1], command}, " && ")
	}

}

// OverlayPlugins will add Plugins' config to JvmAgentStr without verification
// Notice, if a config is not in agent.config, it will be seen as a plugin config
// user must ensure the accuracy of configuration.
// Otherwides,If a separator(, or =) in the option or value, it should be wrapped in quotes.
func (s *SidecarInjectField) OverlayPlugins(annotation *map[string]string) {
	for k, v := range *annotation {
		if strings.HasPrefix(k, pluginsAnnotationPrefix) {
			configName := strings.TrimPrefix(k, pluginsAnnotationPrefix)
			config := strings.Join([]string{configName, v}, "=")
			// add to jvmAgentConfigStr
			if s.JvmAgentConfigStr != "" {
				s.JvmAgentConfigStr = strings.Join([]string{s.JvmAgentConfigStr, config}, ",")
			} else {
				s.JvmAgentConfigStr = config
			}
		}
	}
}

// ValidateConfigmap will validate a configmap(if exists) to set java agent config.
func (s *SidecarInjectField) ValidateConfigmap(ctx context.Context, kubeclient client.Client, namespace string,
	annotation *map[string]string) bool {
	if len(s.ConfigmapVolume.Name) == 0 || len(s.ConfigmapVolume.ConfigMap.Name) == 0 {
		return true
	}
	configmap := &corev1.ConfigMap{}
	configmapName := s.ConfigmapVolume.VolumeSource.ConfigMap.LocalObjectReference.Name
	// check whether the configmap is existed
	err := kubeclient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: configmapName}, configmap)
	if err != nil && !errors.IsNotFound(err) {
		log.Error(err, "Get Configmap failed", "configmapName", configmapName, "namespace", namespace)
		return false
	}
	// if configmap exist , validate it
	if !errors.IsNotFound(err) {
		ok, errinfo := ValidateConfigmap(configmap)
		if ok {
			log.Info("the configmap validate true", "configmapName", configmapName)
			return true
		}
		log.Error(errinfo, "the configmap validate false", "configmapName", configmapName)
	}
	return true
}

func GetInjectedAgentConfig(annotation *map[string]string, configuration *map[string]string) {
	for k, v := range *annotation {
		if strings.HasPrefix(k, agentAnnotationPrefix) {
			option := strings.TrimPrefix(k, agentAnnotationPrefix)
			(*configuration)[option] = strings.Join([]string{"\"", "\""}, v)
		} else if strings.HasPrefix(k, pluginsAnnotationPrefix) {
			option := strings.TrimPrefix(k, pluginsAnnotationPrefix)
			(*configuration)[option] = strings.Join([]string{"\"", "\""}, v)
		} else if strings.EqualFold(k, optionsAnnotation) {
			option := "optional-plugin"
			(*configuration)[option] = strings.Join([]string{"\"", "\""}, v)
		} else if strings.EqualFold(k, optionsReporterAnnotation) {
			option := "optional-reporter-plugin"
			(*configuration)[option] = strings.Join([]string{"\"", "\""}, v)
		}
	}
}
