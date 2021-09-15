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
		ValidateIpandPort,
	}
)

// FindValidateFunc is find the validate function for an annotation
func FindValidateFunc(funcName string) AnnotationValidateFunc {
	for _, f := range AnnotationValidateFuncs {
		// extract the function name into a string , it will be like following
		// github.com/apache/skywalking-swck/pkg/operator/injector.ValidateBool
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

//ValidateIpandPort validates an annotation's value is valid ip and port
func ValidateIpandPort(annotation, value string) error {
	match, err := regexp.MatchString(`(^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.`+
		`(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.`+
		`(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.`+
		`(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])|`+
		`localhost)\:`+
		`([0-9]|[1-9]\d{1,3}|[1-5]\d{4}|6[0-5]{2}[0-3][0-5])$`, value)

	if err != nil {
		return fmt.Errorf("%s error:%s", annotation, err.Error())
	}
	if !match {
		return fmt.Errorf("%s error:not a valid ip and port", annotation)
	}
	return nil
}
