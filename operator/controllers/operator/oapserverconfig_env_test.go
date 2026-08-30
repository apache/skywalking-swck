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

package operator

import (
	"testing"

	core "k8s.io/api/core/v1"
)

// An OAPServerConfig overlays the OAP's environment; it does not own it. The storage wiring the
// OAPServer controller injects has to survive, or the OAP comes back looking for a database on
// localhost -- which is what happened for as long as SkyWalking still had the embedded H2 it
// removed in 10.2.0.
func TestOverlayEnv(t *testing.T) {
	base := []core.EnvVar{
		{Name: "JAVA_OPTS", Value: "-Xmx1g"},
		{Name: "SW_STORAGE", Value: "banyandb"},
		{Name: "SW_STORAGE_BANYANDB_TARGETS", Value: "storage-banyandb-grpc.skywalking-system:17912"},
	}
	overrides := []core.EnvVar{
		{Name: "SW_CORE_RECORD_DATA_TTL", Value: "4"},
		{Name: "JAVA_OPTS", Value: "-Xmx2g"},
	}

	got := overlayEnv(base, overrides)
	byName := map[string]string{}
	for _, e := range got {
		if _, dup := byName[e.Name]; dup {
			t.Errorf("%s appears twice", e.Name)
		}
		byName[e.Name] = e.Value
	}

	// The storage wiring is untouched.
	if byName["SW_STORAGE"] != "banyandb" {
		t.Errorf("SW_STORAGE = %q, want banyandb", byName["SW_STORAGE"])
	}
	if byName["SW_STORAGE_BANYANDB_TARGETS"] == "" {
		t.Error("SW_STORAGE_BANYANDB_TARGETS was dropped")
	}
	// An override of the same name wins.
	if byName["JAVA_OPTS"] != "-Xmx2g" {
		t.Errorf("JAVA_OPTS = %q, want the override", byName["JAVA_OPTS"])
	}
	// A new name is added.
	if byName["SW_CORE_RECORD_DATA_TTL"] != "4" {
		t.Error("the new variable was not added")
	}
	if len(got) != 4 {
		t.Errorf("got %d variables, want 4: %v", len(got), got)
	}

	// Re-applying the same overlay must not churn the deployment.
	if second := overlayEnv(got, overrides); len(second) != len(got) {
		t.Errorf("overlay is not idempotent: %v", second)
	}
}

// Removing a name from spec.env has to take it back off the container. An overlay that only ever
// adds leaves the value in place while the md5 label says the deployment converged, so the OAP
// keeps running settings the user deleted.
func TestApplyEnvOverlayRemoval(t *testing.T) {
	// What the OAPServer template renders.
	template := []core.EnvVar{
		{Name: "JAVA_OPTS", Value: "-Xmx2048M"},
		{Name: "SW_STORAGE", Value: "banyandb"},
		{Name: "SW_STORAGE_BANYANDB_TARGETS", Value: "storage-banyandb-grpc.skywalking-system:17912"},
	}
	overrides := []core.EnvVar{
		{Name: "JAVA_OPTS", Value: "-Xmx8g"},          // shadows a template value
		{Name: "SW_CORE_RECORD_DATA_TTL", Value: "4"}, // introduced by the overlay
	}

	overlaid, originals := applyEnvOverlay(template, overrides, map[string]shadowedEnv{})
	got := map[string]string{}
	for _, e := range overlaid {
		got[e.Name] = e.Value
	}
	if got["JAVA_OPTS"] != "-Xmx8g" || got["SW_CORE_RECORD_DATA_TTL"] != "4" {
		t.Fatalf("overlay did not apply: %v", got)
	}
	if got["SW_STORAGE"] != "banyandb" {
		t.Error("the storage wiring was lost")
	}

	// Now the user empties spec.env.
	reverted, originalsAfter := applyEnvOverlay(overlaid, nil, originals)
	got = map[string]string{}
	for _, e := range reverted {
		got[e.Name] = e.Value
	}
	if _, present := got["SW_CORE_RECORD_DATA_TTL"]; present {
		t.Error("a variable the overlay introduced survived its removal")
	}
	if got["JAVA_OPTS"] != "-Xmx2048M" {
		t.Errorf("JAVA_OPTS = %q, want the template value back", got["JAVA_OPTS"])
	}
	if got["SW_STORAGE"] != "banyandb" || got["SW_STORAGE_BANYANDB_TARGETS"] == "" {
		t.Error("reverting the overlay disturbed the storage wiring")
	}
	if len(originalsAfter) != 0 {
		t.Errorf("nothing is overlaid any more, so nothing should be recorded: %v", originalsAfter)
	}

	// Dropping just one of the two leaves the other alone.
	overlaid2, originals2 := applyEnvOverlay(template, overrides, map[string]shadowedEnv{})
	kept, _ := applyEnvOverlay(overlaid2, []core.EnvVar{{Name: "SW_CORE_RECORD_DATA_TTL", Value: "4"}}, originals2)
	got = map[string]string{}
	for _, e := range kept {
		got[e.Name] = e.Value
	}
	if got["JAVA_OPTS"] != "-Xmx2048M" {
		t.Errorf("JAVA_OPTS = %q, want the template value back", got["JAVA_OPTS"])
	}
	if got["SW_CORE_RECORD_DATA_TTL"] != "4" {
		t.Error("the still-requested override was dropped")
	}
}

// An OAPServerConfig's static files are mounted alongside whatever the pod already carries. The
// overlay used to assign both lists, which under an RFC 7386 merge patch replaces them -- so a
// storage's TLS certificate volume disappeared the moment a spec.file was applied.
func TestMergeVolumesKeepsWhatIsAlreadyThere(t *testing.T) {
	baseMounts := []core.VolumeMount{
		{Name: "storage-tls", MountPath: "/skywalking/bydb-tls", ReadOnly: true},
	}
	added := []core.VolumeMount{
		{Name: "static-config", MountPath: "/skywalking/config/alarm-settings.yml"},
	}

	got := mergeVolumeMounts(baseMounts, added)
	names := map[string]string{}
	for _, m := range got {
		names[m.Name] = m.MountPath
	}
	if names["storage-tls"] != "/skywalking/bydb-tls" {
		t.Error("the storage TLS mount was dropped")
	}
	if names["static-config"] == "" {
		t.Error("the static file mount was not added")
	}

	// Re-applying must not duplicate, or the pod template churns on every reconcile.
	if second := mergeVolumeMounts(got, added); len(second) != len(got) {
		t.Errorf("not idempotent: %d then %d", len(got), len(second))
	}

	baseVols := []core.Volume{{Name: "storage-tls"}}
	vols := mergeVolumes(baseVols, []core.Volume{{Name: "static-config"}})
	if len(vols) != 2 {
		t.Errorf("got %d volumes, want 2: %v", len(vols), vols)
	}
	if same := mergeVolumes(vols, []core.Volume{{Name: "static-config"}}); len(same) != 2 {
		t.Errorf("volume merge is not idempotent: %v", same)
	}
}
