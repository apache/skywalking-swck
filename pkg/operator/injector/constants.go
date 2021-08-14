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

// nolint
const (
	//the annotation that contain the sidecar's basic information,such as agent container..etc
	SidecarInitcontainerName        = "sidecar.skywalking.apache.org/initcontainers.name"
	SidecarInitcontainerImage       = "sidecar.skywalking.apache.org/initcontainers.image"
	SidecarInitcontainerCommand     = "sidecar.skywalking.apache.org/initcontainers.command"
	SidecarInitcontainerArgsOption  = "sidecar.skywalking.apache.org/initcontainers.args.option"
	SidecarInitcontainerArgsCommand = "sidecar.skywalking.apache.org/initcontainers.args.command"
	SidecarVolumeName               = "sidecar.skywalking.apache.org/volume.name"
	SidecarVolumemountMountpath     = "sidecar.skywalking.apache.org/containers.volumemounts.mountpath"
	ConfigmapName                   = "sidecar.skywalking.apache.org/configmap.name"
	ConfigmapVolumeName             = "sidecar.skywalking.apache.org/configmap.volume.name"
	ConfigmapVolumemountMountpath   = "sidecar.skywalking.apache.org/configmap.volumemounts.mountpath"
	SidecarEnvVarName               = "sidecar.skywalking.apache.org/envvar.name"
	SidecarEnvValue                 = "sidecar.skywalking.apache.org/envvar.value"
	SidecarInjectContainerName      = "sidecar.skywalking.apache.org/sidecar.containers.name"

	//the annotation that specify the reason for injection failure
	SidecarInjectErrorInfo = "sidecar.skywalking.apache.org/error"

	//the annotation that open or close agent overlay
	SidecarAgentConfigOverlay = "sidecar.skywalking.apache.org/agent.config.overlay"

	//the annotation that specify the agent config
	AgentNamespace              = "agent.skywalking.apache.org/agent.namespace"
	AgentServiceName            = "agent.skywalking.apache.org/agent.service_name"
	AgentSampleNumber           = "agent.skywalking.apache.org/agent.sample_n_per_3_secs"
	AgentAuthentication         = "agent.skywalking.apache.org/agent.authentication"
	AgentSpanLimit              = "agent.skywalking.apache.org/agent.span_limit_per_segment"
	AgentIgnoreSuffix           = "agent.skywalking.apache.org/agent.ignore_suffix"
	AgentIsOpenDebugging        = "agent.skywalking.apache.org/agent.is_open_debugging_class"
	AgentIsCacheEnhaned         = "agent.skywalking.apache.org/agent.is_cache_enhanced_class"
	AgentClassCache             = "agent.skywalking.apache.org/agent.class_cache_mode"
	AgentOperationName          = "agent.skywalking.apache.org/agent.operation_name_threshold"
	AgentForceTLS               = "agent.skywalking.apache.org/agent.force_tls"
	AgentProfileActive          = "agent.skywalking.apache.org/profile.active"
	AgentProfileMaxParallel     = "agent.skywalking.apache.org/profile.max_parallel"
	AgentProfileMaxDuration     = "agent.skywalking.apache.org/profile.max_duration"
	AgentProfileDump            = "agent.skywalking.apache.org/profile.dump_max_stack_depth"
	AgentProfileSnapshot        = "agent.skywalking.apache.org/profile.snapshot_transport_buffer_size"
	AgentCollectorService       = "agent.skywalking.apache.org/collector.backend_service"
	AgentLoggingName            = "agent.skywalking.apache.org/logging.file_name"
	AgentLoggingLevel           = "agent.skywalking.apache.org/logging.level"
	AgentLoggingDir             = "agent.skywalking.apache.org/logging.dir"
	AgentLoggingMaxSize         = "agent.skywalking.apache.org/logging.max_file_size"
	AgentLoggingMaxFiles        = "agent.skywalking.apache.org/logging.max_history_files"
	AgentStatuscheckExceptions  = "agent.skywalking.apache.org/statuscheck.ignored_exceptions"
	AgentStatuscheckDepth       = "agent.skywalking.apache.org/statuscheck.max_recursive_depth"
	AgentPluginMount            = "agent.skywalking.apache.org/plugin.mount"
	AgentExcludePlugins         = "agent.skywalking.apache.org/plugin.exclude_plugins"
	AgentPluginJdbc             = "agent.skywalking.apache.org/plugin.jdbc.trace_sql_parameters"
	AgentPluginKafkaServers     = "agent.skywalking.apache.org/plugin.kafka.bootstrap_servers"
	AgentPluginKafkaNamespace   = "agent.skywalking.apache.org/plugin.kafka.namespace"
	AgentPluginSpringannotation = "agent.skywalking.apache.org/plugin.springannotation.classname_match_regex"
)
