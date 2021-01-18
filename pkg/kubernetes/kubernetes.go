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

package kubernetes

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"strings"
	"text/template"
	"unicode"

	"github.com/Masterminds/sprig/v3"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/ghodss/yaml"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

var ErrNothingLoaded = errors.New("LoadTemplate: failed load anything from manifests")

// ApplyOverlay applies an overlay using JSON patch strategy over the current Object in place.
func ApplyOverlay(current, overlay runtime.Object) error {
	cj, err := runtime.Encode(unstructured.UnstructuredJSONScheme, current)
	if err != nil {
		return err
	}
	uj, err := runtime.Encode(unstructured.UnstructuredJSONScheme, overlay)
	if err != nil {
		return err
	}
	merged, err := jsonpatch.MergePatch(cj, uj)
	if err != nil {
		return err
	}
	return runtime.DecodeInto(unstructured.UnstructuredJSONScheme, merged, current)
}

// LoadTemplate loads a YAML file to a component
func LoadTemplate(manifest string, values interface{}, funcMap template.FuncMap, spec interface{}) error {
	bb, err := GenerateManifests(manifest, values, funcMap)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(bb, spec)
}

// GenerateManifests generate manifests from templates, CR and values
func GenerateManifests(manifest string, values interface{}, funcMap template.FuncMap) ([]byte, error) {
	tmplBuilder := template.New("manifest").
		Funcs(template.FuncMap{
			"toYAML": toYAML,
		}).
		Funcs(sprig.TxtFuncMap())
	if funcMap != nil {
		tmplBuilder = tmplBuilder.Funcs(funcMap)
	}
	tmpl, err := tmplBuilder.Parse(manifest)
	if err != nil {
		return nil, err
	}
	buf := bytes.Buffer{}
	err = tmpl.Execute(&buf, values)
	if err != nil {
		return nil, err
	}
	bb := stripCharacters(buf.Bytes())
	if len(bb) < 1 {
		return nil, ErrNothingLoaded
	}
	return bb, nil
}

func toYAML(v interface{}) string {
	b, _ := yaml.Marshal(v)
	return string(b)
}

func stripCharacters(bb []byte) []byte {
	s := string(bb)
	s = strings.TrimSpace(s)
	s = stripComment(s)
	s = strings.TrimSpace(s)
	return []byte(s)
}

const commentChars = "#"

func stripComment(source string) string {
	sc := bufio.NewScanner(strings.NewReader(source))
	ll := make([]string, 0)
	for sc.Scan() {
		l := sc.Text()
		if cut := strings.IndexAny(l, commentChars); cut >= 0 {
			l = strings.TrimRightFunc(l[:cut], unicode.IsSpace)
		}
		if strings.TrimSpace(l) != "" {
			ll = append(ll, l)
		}
	}
	return strings.Join(ll, "\n")
}

type ErrorCollector []error

func (c *ErrorCollector) Collect(e error) { *c = append(*c, e) }

func (c *ErrorCollector) Error() error {
	if len(*c) < 1 {
		return nil
	}
	err := "Collected errors:\n"
	for i, e := range *c {
		err += fmt.Sprintf("\tError %d: %s\n", i, e.Error())
	}
	return fmt.Errorf(err)
}
