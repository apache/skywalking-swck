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
	"fmt"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestSidecarInjectField_Inject(t *testing.T) {
	type fields struct {
		needInject           bool
		agentOverlay         bool
		initcontainer        corev1.Container
		sidecarVolume        corev1.Volume
		sidecarVolumeMount   corev1.VolumeMount
		configmapVolume      corev1.Volume
		configmapVolumeMount corev1.VolumeMount
		env                  corev1.EnvVar
		jvmAgentConfigStr    string
		injectContainer      string
	}
	type args struct {
		pod *corev1.Pod
	}

	Pod := &corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "app",
				},
			},
		},
	}

	// injected sidecar Volumes
	injectedSV := corev1.Volume{
		Name:         "sky-agent",
		VolumeSource: corev1.VolumeSource{EmptyDir: nil},
	}

	// injected configmap Volumes
	injectedCV := corev1.Volume{
		Name: "java-agent-configmap-volume",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "java-agent-configmap",
				},
			},
		},
	}

	// injected Sidecar's VolumeMount
	injectedSVM := corev1.VolumeMount{
		MountPath: "/sky/agent",
		Name:      "sky-agent",
	}

	// injected Configmap's VolumeMount
	injectedCVM := corev1.VolumeMount{
		Name:      "java-agent-configmap-volume",
		MountPath: "/sky/agent/config",
	}

	// injected InitContainer
	injectedIC := corev1.Container{
		Name:         "inject-sky-agent",
		Image:        "apache/skywalking-java-agent:8.6.0-jdk8",
		Command:      []string{"sh"},
		Args:         []string{"-c", "mkdir -p /sky/agent && cp -r /skywalking/agent/* /sky/agent"},
		VolumeMounts: []corev1.VolumeMount{injectedSVM},
	}

	// injected Container's EnvVar
	injectedEV := corev1.EnvVar{
		Name:  "AGENT_OPTS",
		Value: " -javaagent:/sky/agent/skywalking-agent.jar",
	}

	injectedPod := &corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "app",
					VolumeMounts: []corev1.VolumeMount{
						injectedSVM, injectedCVM,
					},
					Env: []corev1.EnvVar{
						injectedEV,
					},
				},
			},
			InitContainers: []corev1.Container{
				injectedIC,
			},
			Volumes: []corev1.Volume{
				injectedSV, injectedCV,
			},
		},
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "test inject function",
			fields: fields{
				initcontainer:        injectedIC,
				sidecarVolume:        injectedSV,
				sidecarVolumeMount:   injectedSVM,
				configmapVolume:      injectedCV,
				configmapVolumeMount: injectedCVM,
				env:                  injectedEV,
				injectContainer:      "app",
			},
			args: args{
				pod: Pod,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SidecarInjectField{
				initcontainer:        tt.fields.initcontainer,
				sidecarVolume:        tt.fields.sidecarVolume,
				sidecarVolumeMount:   tt.fields.sidecarVolumeMount,
				configmapVolume:      tt.fields.configmapVolume,
				configmapVolumeMount: tt.fields.configmapVolumeMount,
				env:                  tt.fields.env,
				jvmAgentConfigStr:    tt.fields.jvmAgentConfigStr,
				needInject:           tt.fields.needInject,
				injectContainer:      tt.fields.injectContainer,
				agentOverlay:         tt.fields.agentOverlay,
			}
			s.Inject(tt.args.pod)
			if !reflect.DeepEqual(injectedPod, tt.args.pod) {
				fmt.Println(injectedPod)
				t.Errorf("Inject() got= %v,want %v", tt.args.pod, injectedPod)
			}
		})
	}
}

func TestSidecarInjectField_GetInjectStrategy(t *testing.T) {
	type fields struct {
		needInject           bool
		agentOverlay         bool
		initcontainer        corev1.Container
		sidecarVolume        corev1.Volume
		sidecarVolumeMount   corev1.VolumeMount
		configmapVolume      corev1.Volume
		configmapVolumeMount corev1.VolumeMount
		env                  corev1.EnvVar
		jvmAgentConfigStr    string
		injectContainer      string
	}
	type args struct {
		containers []corev1.Container
		labels     *map[string]string
		annotation *map[string]string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "test right",
			fields: fields{
				needInject:      false,
				injectContainer: "",
				agentOverlay:    false,
			},
			args: args{
				containers: []corev1.Container{
					{
						Name: "container1",
					},
				},
				labels: &map[string]string{
					labelKeyagentInjector: "true",
				},
				annotation: &map[string]string{
					AnnoInjectContainerName.Name: "container1",
					AnnoAgentConfigOverlay.Name:  "true",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SidecarInjectField{
				initcontainer:        tt.fields.initcontainer,
				sidecarVolume:        tt.fields.sidecarVolume,
				sidecarVolumeMount:   tt.fields.sidecarVolumeMount,
				configmapVolume:      tt.fields.configmapVolume,
				configmapVolumeMount: tt.fields.configmapVolumeMount,
				env:                  tt.fields.env,
				jvmAgentConfigStr:    tt.fields.jvmAgentConfigStr,
				needInject:           tt.fields.needInject,
				injectContainer:      tt.fields.injectContainer,
				agentOverlay:         tt.fields.agentOverlay,
			}
			s.GetInjectStrategy(tt.args.containers, tt.args.labels, tt.args.annotation)
			if !s.needInject || s.injectContainer != "container1" || !s.agentOverlay {
				t.Errorf("GetInjectStrategy got= [%v,%v,%v], want [true,container1,true]", s.needInject,
					s.injectContainer, s.agentOverlay)
			}
		})
	}
}

func TestSidecarInjectField_OverlaySidecar(t *testing.T) {
	type fields struct {
		needInject           bool
		agentOverlay         bool
		initcontainer        corev1.Container
		sidecarVolume        corev1.Volume
		sidecarVolumeMount   corev1.VolumeMount
		configmapVolume      corev1.Volume
		configmapVolumeMount corev1.VolumeMount
		env                  corev1.EnvVar
		jvmAgentConfigStr    string
		injectContainer      string
	}
	type args struct {
		as         *Annotations
		annotation *map[string]string
	}
	initcontainer := corev1.Container{
		Name:    AnnoInitcontainerName.DefaultValue,
		Image:   AnnoInitcontainerImage.DefaultValue,
		Command: []string{AnnoInitcontainerCommand.DefaultValue},
		Args: []string{AnnoInitcontainerArgsOption.DefaultValue, AnnoInitcontainerArgsCommand.
			DefaultValue},
	}
	sidecarVolume := corev1.Volume{
		Name: AnnoSidecarVolumeName.DefaultValue,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: nil,
		},
	}
	sidecarVolumeMount := corev1.VolumeMount{
		Name:      AnnoSidecarVolumeName.DefaultValue,
		MountPath: AnnoSidecarVolumemountMountpath.DefaultValue,
	}
	configmapVolume := corev1.Volume{
		Name: AnnoConfigmapVolumeName.DefaultValue,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: AnnoConfigmapName.DefaultValue,
				},
			},
		},
	}
	configmapVolumeMount := corev1.VolumeMount{
		Name:      AnnoConfigmapVolumeName.DefaultValue,
		MountPath: AnnoConfigmapVolumemountMountpath.DefaultValue,
	}
	env := corev1.EnvVar{
		Name:  AnnoEnvVarName.DefaultValue,
		Value: AnnoEnvVarValue.DefaultValue,
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "test sidecar overlay",
			fields: fields{
				initcontainer:        initcontainer,
				sidecarVolume:        sidecarVolume,
				sidecarVolumeMount:   sidecarVolumeMount,
				configmapVolume:      configmapVolume,
				configmapVolumeMount: configmapVolumeMount,
				env:                  env,
			},
			args: args{
				as: NewAnnotations(),
				annotation: &map[string]string{
					AnnoInitcontainerName.Name:             "test-inject-agent",
					AnnoInitcontainerImage.Name:            "apache/skywalking-java-agent:8.5.0-jdk8",
					AnnoInitcontainerCommand.Name:          "sh",
					AnnoInitcontainerArgsOption.Name:       "-c",
					AnnoInitcontainerArgsCommand.Name:      "mkdir -p /skytest/agent && cp -r /skywalking/agent/* /skytest/agent",
					AnnoSidecarVolumemountMountpath.Name:   "/skytest/agent",
					AnnoConfigmapVolumemountMountpath.Name: "/skytest/agent/config",
					AnnoEnvVarValue.Name:                   " -javaagent:/skytest/agent/skywalking-agent.jar",
				},
			},
			want: true,
		},
	}

	overlaied := &SidecarInjectField{
		initcontainer:        initcontainer,
		sidecarVolume:        sidecarVolume,
		sidecarVolumeMount:   sidecarVolumeMount,
		configmapVolume:      configmapVolume,
		configmapVolumeMount: configmapVolumeMount,
		env:                  env,
	}
	overlaied.initcontainer.Name = "test-inject-agent"
	overlaied.initcontainer.Image = "apache/skywalking-java-agent:8.5.0-jdk8"
	overlaied.initcontainer.Command = []string{"sh"}
	overlaied.initcontainer.Args = []string{"-c", "mkdir -p /skytest/agent && cp -r /skywalking/agent/* /skytest/agent"}
	overlaied.sidecarVolumeMount.MountPath = "/skytest/agent"
	overlaied.configmapVolumeMount.MountPath = "/skytest/agent/config"
	overlaied.env.Value = " -javaagent:/skytest/agent/skywalking-agent.jar"
	overlaied.initcontainer.VolumeMounts = []corev1.VolumeMount{overlaied.sidecarVolumeMount}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SidecarInjectField{
				initcontainer:        tt.fields.initcontainer,
				sidecarVolume:        tt.fields.sidecarVolume,
				sidecarVolumeMount:   tt.fields.sidecarVolumeMount,
				configmapVolume:      tt.fields.configmapVolume,
				configmapVolumeMount: tt.fields.configmapVolumeMount,
				env:                  tt.fields.env,
				jvmAgentConfigStr:    tt.fields.jvmAgentConfigStr,
				needInject:           tt.fields.needInject,
				injectContainer:      tt.fields.injectContainer,
				agentOverlay:         tt.fields.agentOverlay,
			}
			if got := s.OverlaySidecar(tt.args.as, tt.args.annotation); got != tt.want || !reflect.DeepEqual(overlaied, s) {
				t.Errorf("SidecarInjectField.OverlaySidecar() = %v, want %v", got, tt.want)
				t.Errorf("after OverlaySidecar,the SidecarInjectField is %v\nwant %v", s, overlaied)
			}
		})
	}
}

func TestSidecarInjectField_OverlayAgentConfig(t *testing.T) {
	type fields struct {
		needInject           bool
		agentOverlay         bool
		initcontainer        corev1.Container
		sidecarVolume        corev1.Volume
		sidecarVolumeMount   corev1.VolumeMount
		configmapVolume      corev1.Volume
		configmapVolumeMount corev1.VolumeMount
		env                  corev1.EnvVar
		jvmAgentConfigStr    string
		injectContainer      string
	}
	type args struct {
		as         *Annotations
		annotation *map[string]string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "test right OverlayAgentConfig",
			fields: fields{},
			args: args{
				as: NewAnnotations(),
				annotation: &map[string]string{
					AnnoAgentNamespace.Name:              "default-namespace",
					AnnoAgentServiceName.Name:            "test",
					AnnoAgentSampleNumber.Name:           "6",
					AnnoAgentAuthentication.Name:         "test",
					AnnoAgentSpanLimit.Name:              "100",
					AnnoAgentIgnoreSuffix.Name:           "jpg,.jpeg",
					AnnoAgentIsOpenDebugging.Name:        "false",
					AnnoAgentIsCacheEnhaned.Name:         "false",
					AnnoAgentClassCache.Name:             "MEMORY",
					AnnoAgentOperationName.Name:          "100",
					AnnoAgentForceTLS.Name:               "false",
					AnnoAgentProfileActive.Name:          "false",
					AnnoAgentProfileMaxParallel.Name:     "10",
					AnnoAgentProfileMaxDuration.Name:     "10",
					AnnoAgentProfileDump.Name:            "200",
					AnnoAgentProfileSnapshot.Name:        "200",
					AnnoAgentCollectorService.Name:       "localhost:2021",
					AnnoAgentLoggingName.Name:            "skywalking-api.log",
					AnnoAgentLoggingLevel.Name:           "INFO",
					AnnoAgentLoggingDir.Name:             "test",
					AnnoAgentLoggingMaxSize.Name:         "1000",
					AnnoAgentLoggingMaxFiles.Name:        "-1",
					AnnoAgentStatuscheckExceptions.Name:  "test",
					AnnoAgentStatuscheckDepth.Name:       "10",
					AnnoAgentPluginMount.Name:            "plugins,activations",
					AnnoAgentExcludePlugins.Name:         "test",
					AnnoAgentPluginJdbc.Name:             "false",
					AnnoAgentPluginKafkaServers.Name:     "127.0.0.1:9092",
					AnnoAgentPluginKafkaNamespace.Name:   "test",
					AnnoAgentPluginSpringannotation.Name: "test",
				},
			},
			want: true,
		},
	}
	finalStr := `agent.namespace=default-namespace,agent.service_name=test,` +
		`agent.sample_n_per_3_secs=6,agent.authentication=test,agent.span_limit_per_segment=100,` +
		`agent.ignore_suffix='jpg,.jpeg',agent.is_open_debugging_class=false,agent.is_cache_enhanced_class=false,` +
		`agent.class_cache_mode=MEMORY,agent.operation_name_threshold=100,agent.force_tls=false,profile.active=false,` +
		`profile.max_parallel=10,profile.max_duration=10,profile.dump_max_stack_depth=200,` +
		`profile.snapshot_transport_buffer_size=200,collector.backend_service=localhost:2021,` +
		`logging.file_name=skywalking-api.log,logging.level=INFO,logging.dir=test,logging.max_file_size=1000,` +
		`logging.max_history_files=-1,statuscheck.ignored_exceptions=test,statuscheck.max_recursive_depth=10,` +
		`plugin.mount='plugins,activations',plugin.exclude_plugins=test,plugin.jdbc.trace_sql_parameters=false,` +
		`plugin.kafka.bootstrap_servers=127.0.0.1:9092,plugin.kafka.namespace=test,` +
		`plugin.springannotation.classname_match_regex=test`
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SidecarInjectField{
				initcontainer:        tt.fields.initcontainer,
				sidecarVolume:        tt.fields.sidecarVolume,
				sidecarVolumeMount:   tt.fields.sidecarVolumeMount,
				configmapVolume:      tt.fields.configmapVolume,
				configmapVolumeMount: tt.fields.configmapVolumeMount,
				env:                  tt.fields.env,
				jvmAgentConfigStr:    tt.fields.jvmAgentConfigStr,
				needInject:           tt.fields.needInject,
				injectContainer:      tt.fields.injectContainer,
				agentOverlay:         tt.fields.agentOverlay,
			}
			if got := s.OverlayAgentConfig(tt.args.as, tt.args.annotation); got != tt.want ||
				finalStr != s.jvmAgentConfigStr {
				t.Errorf("SidecarInjectField.OverlayAgentConfig() = %v, want %v", got, tt.want)
				t.Errorf("SidecarInjectField.OverlayAgentConfig():%s\nwant %s", s.jvmAgentConfigStr, finalStr)
			}
		})
	}
}

func TestSidecarInjectField_OverlayPluginsConfig(t *testing.T) {
	type fields struct {
		needInject           bool
		agentOverlay         bool
		initcontainer        corev1.Container
		sidecarVolume        corev1.Volume
		sidecarVolumeMount   corev1.VolumeMount
		configmapVolume      corev1.Volume
		configmapVolumeMount corev1.VolumeMount
		env                  corev1.EnvVar
		jvmAgentConfigStr    string
		injectContainer      string
	}
	type args struct {
		annotation *map[string]string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "test right OverlayPluginsConfig",
			fields: fields{},
			args: args{
				annotation: &map[string]string{
					otherPluginsAnnotationPrefix + "plugin.influxdb.trace_influxql": "false",
					otherPluginsAnnotationPrefix + "plugin.mongodb.trace_param":     "true",
				},
			},
			want: true,
		},
	}
	// avoid random traversal of map type
	finalStr1 := `plugin.influxdb.trace_influxql=false,plugin.mongodb.trace_param=true`
	finalStr2 := `plugin.mongodb.trace_param=true,plugin.influxdb.trace_influxql=false`
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SidecarInjectField{
				initcontainer:        tt.fields.initcontainer,
				sidecarVolume:        tt.fields.sidecarVolume,
				sidecarVolumeMount:   tt.fields.sidecarVolumeMount,
				configmapVolume:      tt.fields.configmapVolume,
				configmapVolumeMount: tt.fields.configmapVolumeMount,
				env:                  tt.fields.env,
				jvmAgentConfigStr:    tt.fields.jvmAgentConfigStr,
				needInject:           tt.fields.needInject,
				injectContainer:      tt.fields.injectContainer,
				agentOverlay:         tt.fields.agentOverlay,
			}
			if got := s.OverlayPluginsConfig(tt.args.annotation); got != tt.want ||
				(s.jvmAgentConfigStr != finalStr1 && s.jvmAgentConfigStr != finalStr2) {
				t.Errorf("SidecarInjectField.OverlayPluginsConfig() = %v, want %v", got, tt.want)
				t.Errorf("SidecarInjectField.OverlayPluginsConfig():%s\nwant %s or %s",
					s.jvmAgentConfigStr, finalStr1, finalStr2)
			}
		})
	}
}
