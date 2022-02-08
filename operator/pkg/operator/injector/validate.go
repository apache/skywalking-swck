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
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"regexp"
	"runtime"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

// AnnotationValidateFunc is the type of validate function
type AnnotationValidateFunc func(annotation, value string) error

var (
	//AnnotationValidateFuncs define all validate functions
	AnnotationValidateFuncs = []AnnotationValidateFunc{
		ValidateServiceName,
		ValidateBackendServices,
		ValidateIPv4OrHostname,
		ValidateResourceRequirements,
	}
)

const (
	// CPU is in cores (100m = .1 cores)
	CPU string = "cpu"
	// Memory is in bytes (100Gi = 100GiB = 100 * 1024 * 1024 * 1024)
	Memory string = "memory"
	// EphemeralStorage is in bytes (100Gi = 100GiB = 100 * 1024 * 1024 * 1024)
	EphemeralStorage string = "ephemeral-storage"
	// HugePagesPrefix is huge pages prefix
	HugePagesPrefix string = "hugepages-"
)

var standardContainerResources = sets.NewString(
	CPU,
	Memory,
	EphemeralStorage,
)

// FindValidateFunc is find the validate function for an annotation
func FindValidateFunc(funcName string) AnnotationValidateFunc {
	for _, f := range AnnotationValidateFuncs {
		// extract the function name into a string , it will be like following
		// github.com/apache/skywalking-swck/operator/pkg/operator/injector.ValidateBool
		fname := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
		// get real validate function name in injector
		index := strings.LastIndex(fname, ".")
		funcname := fname[index+1:]

		if funcname == funcName {
			return f
		}
	}
	return nil
}

//ValidateServiceName validates the ServiceName is nil or not
func ValidateServiceName(annotation, value string) error {
	if value == "" {
		return fmt.Errorf("%s error:the service name is nil", annotation)
	}
	return nil
}

//ValidateBackendServices validates an annotation's value is valid backend services
func ValidateBackendServices(annotation, value string) error {
	services := []string{value}
	if strings.Contains(value, ",") {
		services = strings.Split(value, ",")
	}
	allEmpty := true
	for _, service := range services {
		service = strings.TrimSpace(service)
		if service == "" {
			continue
		}
		allEmpty = false
		err := ValidateIPv4OrHostname(annotation, service)
		if err != nil {
			return err
		}
	}
	if allEmpty {
		return fmt.Errorf("%s error:every service is nil", annotation)
	}
	return nil
}

//ValidateIPv4OrHostname validates an annotation's value is valid ipv4 or hostname
func ValidateIPv4OrHostname(annotation, service string) error {
	service = strings.TrimSpace(service)
	colonIndex := strings.LastIndex(service, ":")
	host := service
	if colonIndex != -1 {
		host = service[:colonIndex]
	}
	if host == "" {
		return fmt.Errorf("%s error:the service name is nil", annotation)
	}
	ip := net.ParseIP(host)
	if ip != nil && ip.To4() != nil {
		return nil
	}

	match, err := regexp.MatchString(`^([a-zA-Z0-9][a-zA-Z0-9_-]{0,62})(\.[a-zA-Z0-9_][a-zA-Z0-9_-]{0,62})*?$`,
		host)
	if err != nil {
		return fmt.Errorf("%s error:%s", annotation, err.Error())
	}
	if !match {
		return fmt.Errorf("%s=%s error:not a valid ipv4 or hostname", annotation, host)
	}

	return nil
}

//ValidateResourceRequirements validates the resource requirement
func ValidateResourceRequirements(annotation, value string) error {
	if value == "nil" {
		return nil
	}

	resource := make(corev1.ResourceList)
	err := json.Unmarshal([]byte(value), &resource)
	if err != nil {
		return fmt.Errorf("%s unmarshal error:%s", annotation, err.Error())
	}

	for resourceName, quantity := range resource {
		//validate resource name
		if !standardContainerResources.Has(string(resourceName)) && !strings.HasPrefix(string(resourceName), HugePagesPrefix) {
			return fmt.Errorf("%s error:%s isn't a standard resource type", annotation, string(resourceName))
		}

		//validate resource quantity value
		if quantity.MilliValue() <= int64(0) {
			return fmt.Errorf("%s error:%d must be greater than 0", annotation, quantity.MilliValue())
		}
	}

	return nil
}
