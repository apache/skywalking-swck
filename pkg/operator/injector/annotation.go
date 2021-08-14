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

//Annotation is used to set default value
type Annotation struct {
	Name         string
	DefaultValue string
}

//Annotations is used to set overlaied value
type Annotations map[Annotation]string

func setDefaultAnnotation(name, value string) Annotation {
	return Annotation{
		Name:         name,
		DefaultValue: value,
	}
}

const (
	namePrefix = "skywalking-swck-"
)

// nolint
var (
	//set sidecar's basic information
	AnnoInitcontainerName  = setDefaultAnnotation(SidecarInitcontainerName, "inject-sky-agent")
	AnnoInitcontainerImage = setDefaultAnnotation(SidecarInitcontainerImage,
		"apache/skywalking-java-agent:8.6.0-jdk8")
	AnnoInitcontainerCommand     = setDefaultAnnotation(SidecarInitcontainerCommand, "sh")
	AnnoInitcontainerArgsOption  = setDefaultAnnotation(SidecarInitcontainerArgsOption, "-c")
	AnnoInitcontainerArgsCommand = setDefaultAnnotation(SidecarInitcontainerArgsCommand,
		"mkdir -p /sky/agent && cp -r /skywalking/agent/* /sky/agent")
	AnnoSidecarVolumeName             = setDefaultAnnotation(SidecarVolumeName, "sky-agent")
	AnnoSidecarVolumemountMountpath   = setDefaultAnnotation(SidecarVolumemountMountpath, "/sky/agent")
	AnnoConfigmapName                 = setDefaultAnnotation(ConfigmapName, namePrefix+"java-agent-configmap")
	AnnoConfigmapVolumeName           = setDefaultAnnotation(ConfigmapVolumeName, "java-agent-configmap-volume")
	AnnoConfigmapVolumemountMountpath = setDefaultAnnotation(ConfigmapVolumemountMountpath, "/sky/agent/config")
	AnnoEnvVarName                    = setDefaultAnnotation(SidecarEnvVarName, "AGENT_OPTS")
	AnnoEnvVarValue                   = setDefaultAnnotation(SidecarEnvValue,
		"-javaagent:/sky/agent/skywalking-agent.jar")
	AnnoInjectContainerName = setDefaultAnnotation(SidecarInjectContainerName, "")
	//set nil
	AnnoInjectErrorInfo    = setDefaultAnnotation(SidecarInjectErrorInfo, "")
	AnnoAgentConfigOverlay = setDefaultAnnotation(SidecarAgentConfigOverlay, "false")
	//set agent config
	AnnoAgentNamespace      = setDefaultAnnotation(AgentNamespace, "default-namespace")
	AnnoAgentServiceName    = setDefaultAnnotation(AgentServiceName, "")
	AnnoAgentSampleNumber   = setDefaultAnnotation(AgentSampleNumber, "-1")
	AnnoAgentAuthentication = setDefaultAnnotation(AgentAuthentication, "xxxx")
	AnnoAgentSpanLimit      = setDefaultAnnotation(AgentSpanLimit, "150")
	AnnoAgentIgnoreSuffix   = setDefaultAnnotation(AgentIgnoreSuffix,
		".jpg,.jpeg,.js,.css,.png,.bmp,.gif,.ico,.mp3,.mp4,.html,.svg")
	AnnoAgentIsOpenDebugging        = setDefaultAnnotation(AgentIsOpenDebugging, "true")
	AnnoAgentIsCacheEnhaned         = setDefaultAnnotation(AgentIsCacheEnhaned, "false")
	AnnoAgentClassCache             = setDefaultAnnotation(AgentClassCache, "MEMORY")
	AnnoAgentOperationName          = setDefaultAnnotation(AgentOperationName, "150")
	AnnoAgentForceTLS               = setDefaultAnnotation(AgentForceTLS, "false")
	AnnoAgentProfileActive          = setDefaultAnnotation(AgentProfileActive, "true")
	AnnoAgentProfileMaxParallel     = setDefaultAnnotation(AgentProfileMaxParallel, "5")
	AnnoAgentProfileMaxDuration     = setDefaultAnnotation(AgentProfileMaxDuration, "10")
	AnnoAgentProfileDump            = setDefaultAnnotation(AgentProfileDump, "500")
	AnnoAgentProfileSnapshot        = setDefaultAnnotation(AgentProfileSnapshot, "50")
	AnnoAgentCollectorService       = setDefaultAnnotation(AgentCollectorService, "127.0.0.1:11800")
	AnnoAgentLoggingName            = setDefaultAnnotation(AgentLoggingName, "skywalking-api.log")
	AnnoAgentLoggingLevel           = setDefaultAnnotation(AgentLoggingLevel, "INFO")
	AnnoAgentLoggingDir             = setDefaultAnnotation(AgentLoggingDir, "")
	AnnoAgentLoggingMaxSize         = setDefaultAnnotation(AgentLoggingMaxSize, "314572800")
	AnnoAgentLoggingMaxFiles        = setDefaultAnnotation(AgentLoggingMaxFiles, "-1")
	AnnoAgentStatuscheckExceptions  = setDefaultAnnotation(AgentStatuscheckExceptions, "")
	AnnoAgentStatuscheckDepth       = setDefaultAnnotation(AgentStatuscheckDepth, "1")
	AnnoAgentPluginMount            = setDefaultAnnotation(AgentPluginMount, "plugins,activations")
	AnnoAgentExcludePlugins         = setDefaultAnnotation(AgentExcludePlugins, "")
	AnnoAgentPluginJdbc             = setDefaultAnnotation(AgentPluginJdbc, "false")
	AnnoAgentPluginKafkaServers     = setDefaultAnnotation(AgentPluginKafkaServers, "localhost:9092")
	AnnoAgentPluginKafkaNamespace   = setDefaultAnnotation(AgentPluginKafkaNamespace, "")
	AnnoAgentPluginSpringannotation = setDefaultAnnotation(AgentPluginSpringannotation, "")
)

//NewAnnotations will create a new Annotations
func NewAnnotations() *Annotations {
	a := make(Annotations)
	return &a
}

//GetFinalValue will get overlaied value first , then default
func (as *Annotations) GetFinalValue(a Annotation) string {
	ov := a.DefaultValue
	if v, ok := (*as)[a]; ok {
		ov = v
	}
	return ov
}

//SetOverlay will set overlaied value
func (as *Annotations) SetOverlay(annotations *map[string]string, a Annotation) error {
	if v, ok := (*annotations)[a.Name]; ok {
		//if annotation has validate func then validate
		if f, funok := AnnotationValidate[a.Name]; funok {
			err := f(a.Name, v)
			//validate error
			if err != nil {
				return err
			}
		}
		//if no validate func then set Overlay directly
		(*as)[a] = v
	}
	return nil
}

//GetOverlayValue will get overlaied value, if not then return ""
func (as *Annotations) GetOverlayValue(a Annotation) string {
	if v, ok := (*as)[a]; ok {
		return v
	}
	return ""
}
