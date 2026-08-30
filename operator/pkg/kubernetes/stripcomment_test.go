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
	"strings"
	"testing"

	"sigs.k8s.io/yaml"
)

// A '#' inside a rendered value used to truncate the line, because every line was cut at its
// first '#'. Values reach here from OAPServer.spec.config, Satellite.spec.config and
// UI.spec.env, so a password, a prompt or a URL fragment containing one produced YAML that no
// longer parsed and a resource that never deployed.
func TestStripCommentKeepsHashesInsideValues(t *testing.T) {
	manifest := `# license header, dropped
apiVersion: v1
kind: ConfigMap
metadata:
  # an indented whole-line comment, also dropped
  name: demo
data:
  password: "p#ssw0rd"
  prompt: "explain #metrics and #traces"
  fragment: "https://example.com/docs#section"
`
	got := stripComment(manifest)

	for _, want := range []string{`"p#ssw0rd"`, `"explain #metrics and #traces"`, `"https://example.com/docs#section"`} {
		if !strings.Contains(got, want) {
			t.Errorf("value was truncated, %s is missing from:\n%s", want, got)
		}
	}
	if strings.Contains(got, "license header") || strings.Contains(got, "whole-line comment") {
		t.Errorf("a whole-line comment survived:\n%s", got)
	}

	// The result still has to be YAML, which is the whole point.
	var out map[string]interface{}
	if err := yaml.Unmarshal([]byte(got), &out); err != nil {
		t.Fatalf("stripped manifest does not parse: %v\n%s", err, got)
	}
	data, _ := out["data"].(map[string]interface{})
	if data["password"] != "p#ssw0rd" {
		t.Errorf("password = %v, want p#ssw0rd", data["password"])
	}
}
