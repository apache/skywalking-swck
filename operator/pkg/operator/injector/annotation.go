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

	"github.com/ghodss/yaml"

	"github.com/apache/skywalking-swck/operator/pkg/operator/manifests"
)

// Annotation defines configuration that user can overlay, including
// sidecar configuration and java agent configuration.
type Annotation struct {
	// Name defines the reference to configuration used in Pod's AnnotationOverlay.
	Name string `yaml:"name"`
	// DefaultValue defines the default value of configuration, if there
	// isn't a default value, it will be "nil".
	DefaultValue string `yaml:"defaultValue"`
	// ValidateFunc defines a Validate func to judge whether the value
	// is valid, if there isn't a validate func, it will be "nil".
	ValidateFunc string `yaml:"validateFunc"`
	// EnvName represent the environment variable , just like following
	// in agent.namespace=${SW_AGENT_NAMESPACE:} , the EnvName is SW_AGENT_NAMESPACE
	EnvName string `yaml:"envName"`
}

// Annotations are set of
type Annotations struct {
	Annotations []Annotation
}

// AnnotationOverlay is used to set overlaied value
type AnnotationOverlay map[Annotation]string

// NewAnnotations will create a new AnnotationOverlay
func NewAnnotations() (*Annotations, error) {
	fileRepo := manifests.NewRepo("injector")
	data, err := fileRepo.ReadFile("injector/templates/annotations.yaml")
	if err != nil {
		return nil, err
	}
	a := new(Annotations)
	if err := yaml.Unmarshal(data, a); err != nil {
		return nil,
			fmt.Errorf("unmarshal annotations.yaml to struct Annotations :%s", err.Error())
	}
	return a, nil
}

// GetAnnotationsByPrefix gets annotations from annotations.yaml
func GetAnnotationsByPrefix(a Annotations, prefixName string) *Annotations {
	anno := new(Annotations)
	for _, v := range a.Annotations {
		if strings.HasPrefix(v.Name, prefixName) {
			anno.Annotations = append(anno.Annotations, v)
		}
	}
	return anno
}

// NewAnnotationOverlay will create a new AnnotationOverlay
func NewAnnotationOverlay() *AnnotationOverlay {
	a := make(AnnotationOverlay)
	return &a
}

// GetFinalValue will get overlaied value first , then default
func (as *AnnotationOverlay) GetFinalValue(a Annotation) string {
	ov := a.DefaultValue
	if v, ok := (*as)[a]; ok {
		ov = v
	}
	return ov
}

// SetOverlay will set overlaied value
func (as *AnnotationOverlay) SetOverlay(AnnotationOverlay *map[string]string, a Annotation) error {
	if v, ok := (*AnnotationOverlay)[a.Name]; ok {
		// if annotation has validate func then validate
		f := FindValidateFunc(a.ValidateFunc)
		if f != nil {
			err := f(a.Name, v)
			// validate error
			if err != nil {
				return err
			}
		}
		// if no validate func then set Overlay directly
		(*as)[a] = v
	}
	return nil
}

// GetOverlayValue will get overlaied value, if not then return ""
func (as *AnnotationOverlay) GetOverlayValue(a Annotation) string {
	if v, ok := (*as)[a]; ok {
		return v
	}
	return ""
}
