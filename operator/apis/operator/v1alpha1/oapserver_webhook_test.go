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

package v1alpha1

import (
	"context"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// An OAP older than the operator supports is warned about, not rejected: a cluster already running
// one keeps working, and the warning is what tells the user why the UI misbehaves.
func TestOAPServerVersionWarning(t *testing.T) {
	tests := []struct {
		version string
		warn    bool
	}{
		{version: "11.0.0", warn: false},
		{version: "10.4.0", warn: false},
		{version: "10.5.0", warn: false},
		{version: "10.3.0", warn: true},
		{version: "9.5.0", warn: true},
		// Not a version this operator can reason about; say nothing rather than guess.
		{version: "latest", warn: false},
		{version: "", warn: false},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			oap := &OAPServer{
				ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "observability"},
				Spec: OAPServerSpec{
					Version: tt.version,
					Image:   "apache/skywalking-oap-server:x",
					// Storage is mandatory since H2 went; this case is about the version warning.
					StorageConfig: &RelevantStorage{Name: "storage"},
				},
			}
			warnings, err := oap.ValidateCreate(context.Background(), oap)
			if err != nil {
				t.Fatalf("ValidateCreate() returned %v", err)
			}
			if tt.warn && len(warnings) == 0 {
				t.Errorf("version %q produced no warning", tt.version)
			}
			if !tt.warn && len(warnings) != 0 {
				t.Errorf("version %q produced %v", tt.version, warnings)
			}
			if tt.warn && !strings.Contains(warnings[0], minimumSupportedOAPVersion) {
				t.Errorf("the warning does not name the minimum: %v", warnings[0])
			}
		})
	}
}

// An OAPServer with nowhere to write is broken rather than degraded: SkyWalking removed H2 in
// 10.2.0, so there is no fallback and the pod dials 127.0.0.1:17912 forever. Say so at admission.
func TestOAPServerRequiresStorage(t *testing.T) {
	base := func() *OAPServer {
		return &OAPServer{
			ObjectMeta: metav1.ObjectMeta{Name: "oap", Namespace: "default"},
			Spec:       OAPServerSpec{Version: "11.0.0", Image: "apache/skywalking-oap-server:11.0.0"},
		}
	}

	t.Run("no storage at all is refused", func(t *testing.T) {
		err := base().validate()
		if err == nil {
			t.Fatal("an OAPServer with no storage was admitted")
		}
		if !strings.Contains(err.Error(), "no storage is configured") {
			t.Errorf("unhelpful message: %v", err)
		}
	})

	t.Run("a Storage reference is enough", func(t *testing.T) {
		o := base()
		o.Spec.StorageConfig = &RelevantStorage{Name: "storage"}
		if err := o.validate(); err != nil {
			t.Errorf("rejected a valid storage reference: %v", err)
		}
	})

	t.Run("an empty Storage name is not", func(t *testing.T) {
		o := base()
		o.Spec.StorageConfig = &RelevantStorage{}
		if err := o.validate(); err == nil {
			t.Error("spec.storage with no name was admitted")
		}
	})

	// Carrying the storage environment by hand was the only way to reach BanyanDB before the
	// operator learned to configure it, and it still counts as having made the choice.
	t.Run("SW_STORAGE in spec.config is enough", func(t *testing.T) {
		o := base()
		o.Spec.Config = []corev1.EnvVar{
			{Name: "SW_STORAGE", Value: "elasticsearch"},
			{Name: "SW_STORAGE_ES_CLUSTER_NODES", Value: "es:9200"},
		}
		if err := o.validate(); err != nil {
			t.Errorf("rejected a hand-configured storage: %v", err)
		}
	})

	t.Run("unrelated config is not", func(t *testing.T) {
		o := base()
		o.Spec.Config = []corev1.EnvVar{{Name: "SW_OTEL_RECEIVER", Value: "default"}}
		if err := o.validate(); err == nil {
			t.Error("config with no SW_STORAGE was admitted")
		}
	})
}

// A name with nothing behind it is not a storage choice.
func TestOAPServerStorageEnvNeedsAValue(t *testing.T) {
	base := func(cfg []corev1.EnvVar) *OAPServer {
		return &OAPServer{
			ObjectMeta: metav1.ObjectMeta{Name: "oap", Namespace: "default"},
			Spec: OAPServerSpec{
				Version: "11.0.0", Image: "apache/skywalking-oap-server:11.0.0", Config: cfg,
			},
		}
	}
	if err := base([]corev1.EnvVar{{Name: "SW_STORAGE"}}).validate(); err == nil {
		t.Error("SW_STORAGE with no value was accepted")
	}
	if err := base([]corev1.EnvVar{{Name: "SW_STORAGE", Value: ""}}).validate(); err == nil {
		t.Error("SW_STORAGE with an empty value was accepted")
	}
	if err := base([]corev1.EnvVar{{Name: "SW_STORAGE", Value: "banyandb"}}).validate(); err != nil {
		t.Errorf("a real value was rejected: %v", err)
	}
	// A secret or configmap reference is a legitimate way to supply it.
	fromRef := []corev1.EnvVar{{Name: "SW_STORAGE", ValueFrom: &corev1.EnvVarSource{
		ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: "oap-cfg"}, Key: "storage",
		},
	}}}
	if err := base(fromRef).validate(); err != nil {
		t.Errorf("a valueFrom source was rejected: %v", err)
	}
}
