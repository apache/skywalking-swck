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
	"strconv"
	"strings"
)

// AnnotationValidateFunc is the type of validate function
type AnnotationValidateFunc func(annotation, value string) error

var (
	//AnnotationValidateFuncs define all validate functions
	AnnotationValidateFuncs = []AnnotationValidateFunc{
		ValidateBool,
		ValidateInt,
		ValidateClassCacheMode,
		ValidateIpandPort,
		ValidateLoggingLevel,
		ValidateResolver,
		ValidateOutput,
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

//ValidateBool validates an annotation's value is bool
func ValidateBool(annotation, value string) error {
	_, err := strconv.ParseBool(value)
	if err != nil {
		return fmt.Errorf("%s error:%s", annotation, err.Error())
	}
	return nil
}

//ValidateInt validates an annotation's value is int
func ValidateInt(annotation, value string) error {
	_, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return fmt.Errorf("%s error:%s", annotation, err.Error())
	}
	return nil
}

//ValidateClassCacheMode validates an annotation's value is right cache mode
func ValidateClassCacheMode(annotation, value string) error {
	if !strings.EqualFold(value, "MEMORY") && !strings.EqualFold(value, "FILE") {
		return fmt.Errorf("%s error:the mode is not MEMORY or FILE", annotation)
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

//ValidateLoggingLevel validates an annotation's value is right logging level
func ValidateLoggingLevel(annotation, value string) error {
	if !strings.EqualFold(value, "TRACE") && !strings.EqualFold(value, "DEBUG") &&
		!strings.EqualFold(value, "INFO") && !strings.EqualFold(value, "WARN") &&
		!strings.EqualFold(value, "ERROR") && !strings.EqualFold(value, "OFF") {
		return fmt.Errorf("%s error:the Level is not in [TRACE,DEBUG,INFO,WARN,ERROR,OFF]", annotation)
	}
	return nil
}

//ValidateResolver validates logging.resolver
func ValidateResolver(annotation, value string) error {
	if !strings.EqualFold(value, "PATTERN") && !strings.EqualFold(value, "JSON") {
		return fmt.Errorf("%s error:the mode is not PATTERN or JSON", annotation)
	}
	return nil
}

//ValidateOutput validates logging.output
func ValidateOutput(annotation, value string) error {
	if !strings.EqualFold(value, "FILE") && !strings.EqualFold(value, "CONSOLE") {
		return fmt.Errorf("%s error:the mode is not FILE or CONSOLE", annotation)
	}
	return nil
}
