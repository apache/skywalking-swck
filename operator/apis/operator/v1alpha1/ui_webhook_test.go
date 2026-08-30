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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// The image a UI gets when none is given is the one thing here that no other test and no e2e case
// covers: every e2e CR sets `image` explicitly. That is how the default came to name
// apache/skywalking-horizon-ui, a Docker Hub repository that does not exist -- so every UI created
// without an image, which is the documented happy path, could never pull.
func TestUIDefaultImage(t *testing.T) {
	tests := []struct {
		name      string
		kind      string
		version   string
		image     string
		wantKind  string
		wantImage string
	}{
		{
			// Horizon and Booster releases share the skywalking-ui repository and are told apart
			// by the horizon- tag prefix.
			name:      "horizon by default",
			version:   "1.0.0",
			wantKind:  "horizon",
			wantImage: "apache/skywalking-ui:horizon-1.0.0",
		},
		{
			name:      "horizon explicitly",
			kind:      "horizon",
			version:   "1.0.0",
			wantKind:  "horizon",
			wantImage: "apache/skywalking-ui:horizon-1.0.0",
		},
		{
			name:      "an explicit image wins",
			kind:      "horizon",
			version:   "1.0.0",
			image:     "ghcr.io/apache/skywalking-horizon-ui:deadbeef",
			wantKind:  "horizon",
			wantImage: "ghcr.io/apache/skywalking-horizon-ui:deadbeef",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ui := &UI{
				ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "observability"},
				Spec:       UISpec{Kind: tt.kind, Version: tt.version, Image: tt.image},
			}
			if err := ui.Default(context.Background(), ui); err != nil {
				t.Fatalf("Default() returned %v", err)
			}
			if ui.Spec.Kind != tt.wantKind {
				t.Errorf("kind = %q, want %q", ui.Spec.Kind, tt.wantKind)
			}
			if ui.Spec.Image != tt.wantImage {
				t.Errorf("image = %q, want %q", ui.Spec.Image, tt.wantImage)
			}
		})
	}
}

// Horizon needs the OAP query URL and cannot guess it, so that one is derived. The admin and
// Zipkin URLs are NOT: the admin host arrived in OAP 11.x and port 17128 belongs to a different
// service on 10.x, and the OAPServer this operator deploys exposes no Zipkin query port. Guessing
// either hands Horizon a URL that resolves to the wrong thing, or to nothing.
func TestUIDefaultOAPAddresses(t *testing.T) {
	ui := &UI{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "observability"},
		Spec:       UISpec{Version: "1.0.0"},
	}
	if err := ui.Default(context.Background(), ui); err != nil {
		t.Fatalf("Default() returned %v", err)
	}
	if got, want := ui.Spec.OAPServerAddress, "http://demo-oap.observability:12800"; got != want {
		t.Errorf("OAPServerAddress = %q, want %q", got, want)
	}
	if ui.Spec.OAPServerAdminAddress != "" {
		t.Errorf("OAPServerAdminAddress was guessed: %q", ui.Spec.OAPServerAdminAddress)
	}
	if ui.Spec.OAPServerZipkinAddress != "" {
		t.Errorf("OAPServerZipkinAddress was guessed: %q", ui.Spec.OAPServerZipkinAddress)
	}

	// What the user sets is kept.
	explicit := &UI{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "observability"},
		Spec: UISpec{
			Version:                "1.0.0",
			OAPServerAdminAddress:  "http://demo-oap.observability:17128",
			OAPServerZipkinAddress: "http://demo-oap.observability:9411",
		},
	}
	if err := explicit.Default(context.Background(), explicit); err != nil {
		t.Fatalf("Default() returned %v", err)
	}
	if explicit.Spec.OAPServerAdminAddress != "http://demo-oap.observability:17128" {
		t.Errorf("an explicit admin address was overwritten: %q", explicit.Spec.OAPServerAdminAddress)
	}
}

// horizon is the only UI this operator deploys. A resource that still asks for the retired Booster
// UI must be refused with a message that says what to do, not silently rendered as horizon.
func TestUIValidateKind(t *testing.T) {
	for _, kind := range []string{"horizon", ""} {
		ui := &UI{
			ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "observability"},
			Spec:       UISpec{Kind: kind, Version: "1.0.0"},
		}
		if err := ui.Default(context.Background(), ui); err != nil {
			t.Fatalf("Default() returned %v", err)
		}
		if _, err := ui.ValidateCreate(context.Background(), ui); err != nil {
			t.Errorf("kind %q was rejected: %v", kind, err)
		}
	}

	for _, kind := range []string{"booster", "booster-ui", "rocketbot"} {
		bad := &UI{
			ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "observability"},
			Spec:       UISpec{Kind: kind, Version: "1.0.0", Image: "x:1"},
		}
		_, err := bad.ValidateCreate(context.Background(), bad)
		if err == nil {
			t.Errorf("kind %q was accepted", kind)
			continue
		}
		if !strings.Contains(err.Error(), "horizon") {
			t.Errorf("kind %q rejected without saying what to use instead: %v", kind, err)
		}
	}
}

// live template mode is only reachable over an OAP admin host, which this operator does not derive.
// Defaulting to live regardless would leave a UI probing 127.0.0.1:17128 and blocking every
// layer-driven page, so the mode follows the address.
func TestUITemplatesModeFollowsTheAdminAddress(t *testing.T) {
	for _, tt := range []struct {
		name, admin, explicit, want string
	}{
		{"no admin host means the bundled templates", "", "", "readonly"},
		{"an admin host makes the OAP store reachable", "http://oap.ns:17128", "", "live"},
		{"an explicit mode is left alone", "", "live", "live"},
		{"an explicit readonly survives an admin host", "http://oap.ns:17128", "readonly", "readonly"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ui := &UI{
				ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "skywalking-system"},
				Spec: UISpec{
					Version: "1.0.0", Instances: 1,
					OAPServerAdminAddress: tt.admin,
					TemplatesMode:         tt.explicit,
				},
			}
			if err := ui.Default(context.Background(), ui); err != nil {
				t.Fatalf("Default() returned %v", err)
			}
			if ui.Spec.TemplatesMode != tt.want {
				t.Errorf("templatesMode = %q, want %q", ui.Spec.TemplatesMode, tt.want)
			}
		})
	}
}
