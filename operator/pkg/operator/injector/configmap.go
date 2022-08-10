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
	"strings"

	corev1 "k8s.io/api/core/v1"
)

const (
	// DefaultConfigmapNamespace is the system namespace
	DefaultConfigmapNamespace = "skywalking-swck-system"
	// DefaultConfigmapName is the default configmap's name
	DefaultConfigmapName = "skywalking-swck-java-agent-configmap"
	// DefaultConfigmapKey is the key of default configmap
	DefaultConfigmapKey = "agent.config"
)

// Get agent annotation from annotations.yaml
func getAgentAnnotations() []Annotation {
	anno, err := NewAnnotations()
	if err != nil {
		return nil
	}
	agent := GetAnnotationsByPrefix(*anno, agentAnnotationPrefix)
	return agent.Annotations
}

// Remove the prefix of annotation in annotations.yaml
func parse(s string) string {
	i := strings.Index(s, "/")
	return s[i+1:]
}

// GetTmplFunc will create template func used in configmap.yaml
func GetTmplFunc() map[string]interface{} {
	return map[string]interface{}{"getAgentAnnotations": getAgentAnnotations, "parse": parse}
}

// ValidateConfigmap will verify the value of configmap
func ValidateConfigmap(configmap *corev1.ConfigMap) (bool, error) {
	data := configmap.Data
	if data == nil {
		return false, fmt.Errorf("configmap don't have data")
	}
	v, ok := data[DefaultConfigmapKey]
	if !ok {
		return false, fmt.Errorf("DefaultConfigmapKey %s not exist", DefaultConfigmapKey)
	}

	anno, err := NewAnnotations()
	if err != nil {
		return false, fmt.Errorf("get NewAnnotations error:%s", err)
	}

	// the following code is to extract the {option}={value} from data of configmap
	// such as agent.service_name=${SW_AGENT_NAME:Your_ApplicationName}  or
	//         agent.service_name = Your_ApplicationName
	// we will get "agent.service_name" and "Your_ApplicationName"
	// so that we can validate every option's value if it has validate function

	str := strings.Split(v, "\n")
	for i := range str {
		if index := strings.Index(str[i], "="); index != -1 {

			// set option and value
			option := strings.Trim(str[i][:index], " ")
			value := strings.Trim(str[i][index+1:], " ")

			// if option has environment variable like SW_AGENT_NAME
			if strings.Contains(str[i], ":") {
				valueStart := strings.Index(str[i], ":")
				valueEnd := strings.Index(str[i], "}")
				if valueStart == -1 || valueEnd == -1 || valueStart >= valueEnd {
					continue
				}
				value = strings.Trim(str[i][valueStart+1:valueEnd], " ")
			}

			for _, a := range anno.Annotations {
				if strings.Contains(a.Name, option) {
					f := FindValidateFunc(a.ValidateFunc)
					if f != nil {
						err := f(a.Name, value)
						//validate error
						if err != nil {
							return false, fmt.Errorf("validate error:%v", err)
						}
					}
				}
			}
		}
	}
	return true, nil
}

// GetConfigmapConfiguration will get the value of agent.config
func GetConfigmapConfiguration(configmap *corev1.ConfigMap) (map[string]string, error) {
	data := configmap.Data
	configuration := make(map[string]string)

	if nil == data {
		return configuration, nil
	}

	v, ok := data[DefaultConfigmapKey]
	if !ok {
		return configuration, nil
	}

	str := strings.Split(v, "\n")
	for i := range str {
		if index := strings.Index(str[i], "="); index != -1 {

			// set option and value
			option := strings.Trim(str[i][:index], " ")
			value := strings.Trim(str[i][index+1:], " ")

			// if option has environment variable like SW_AGENT_NAME
			if strings.Contains(str[i], ":") {
				valueStart := strings.Index(str[i], ":")
				valueEnd := strings.Index(str[i], "}")
				if valueStart == -1 || valueEnd == -1 || valueStart >= valueEnd {
					continue
				}
				value = strings.Trim(str[i][valueStart+1:valueEnd], " ")
			}
			if len(value) == 0 {
				continue
			}
			configuration[option] = strings.Join([]string{"\"", "\""}, value)
		}
	}
	return configuration, nil
}
