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
	"net"
	"reflect"
	"regexp"
	"runtime"
	"strings"
)

// AnnotationValidateFunc is the type of validate function
type AnnotationValidateFunc func(annotation, value string) error

var (
	//AnnotationValidateFuncs define all validate functions
	AnnotationValidateFuncs = []AnnotationValidateFunc{
		ValidateServiceName,
		ValidateBackendServices,
		ValidateIPv4OrHostname,
	}
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
