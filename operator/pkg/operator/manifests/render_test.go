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

package manifests

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	operatorv1alpha1 "github.com/apache/skywalking-swck/operator/apis/operator/v1alpha1"
	"github.com/apache/skywalking-swck/operator/pkg/kubernetes"
)

// render executes a component template against a custom resource and unmarshals the result, so a
// template that produces something the API server would reject fails here instead of in a cluster.
func render(t *testing.T, component, file string, cr interface{}, out interface{}) string {
	t.Helper()
	tmpl, err := NewRepo(component).ReadFile(component + "/templates/" + file)
	if err != nil {
		t.Fatalf("reading %s/%s: %v", component, file, err)
	}
	got, err := kubernetes.GenerateManifests(string(tmpl), cr, nil)
	if err != nil {
		t.Fatalf("rendering %s/%s: %v", component, file, err)
	}
	if out != nil {
		if err := yaml.Unmarshal(got, out); err != nil {
			t.Fatalf("rendered %s/%s is not valid YAML: %v\n%s", component, file, err, got)
		}
	}
	return string(got)
}

func ui(mutate func(*operatorv1alpha1.UISpec)) *operatorv1alpha1.UI {
	spec := operatorv1alpha1.UISpec{
		Kind: "horizon", Version: "1.0.0", Instances: 1,
		Image:            "apache/skywalking-ui:horizon-1.0.0",
		OAPServerAddress: "http://demo-oap.skywalking-system:12800",
	}
	if mutate != nil {
		mutate(&spec)
	}
	return &operatorv1alpha1.UI{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "skywalking-system"},
		Spec:       spec,
	}
}

func containerOf(t *testing.T, d *appsv1.Deployment) corev1.Container {
	t.Helper()
	if len(d.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf("want exactly one container, got %d", len(d.Spec.Template.Spec.Containers))
	}
	return d.Spec.Template.Spec.Containers[0]
}

func envOf(c corev1.Container) map[string]string {
	m := map[string]string{}
	for _, e := range c.Env {
		m[e.Name] = e.Value
	}
	return m
}

// Horizon is configured with environment variables against the tokenised config baked into its
// image. The operator supplies only what it derives; everything else is the user's to pass.
func TestUIDeploymentEnv(t *testing.T) {
	t.Run("the plain case sets the derived variables and mounts no config", func(t *testing.T) {
		var d appsv1.Deployment
		render(t, "ui", "deployment.yaml", ui(nil), &d)
		c := containerOf(t, &d)
		env := envOf(c)

		if env["HORIZON_OAP_QUERY_URL"] != "http://demo-oap.skywalking-system:12800" {
			t.Errorf("HORIZON_OAP_QUERY_URL = %q", env["HORIZON_OAP_QUERY_URL"])
		}
		// The webhook derives this; unset here means webhooks are off, and readonly is the mode
		// that works without an admin host.
		if env["HORIZON_TEMPLATES_MODE"] != "readonly" {
			t.Errorf("HORIZON_TEMPLATES_MODE = %q, want readonly", env["HORIZON_TEMPLATES_MODE"])
		}
		// Unset addresses are omitted rather than emitted empty: Horizon's schema takes URLs, and
		// an empty string fails validation instead of meaning "absent".
		if _, ok := env["HORIZON_OAP_ADMIN_URL"]; ok {
			t.Error("HORIZON_OAP_ADMIN_URL was emitted for an unset address")
		}
		if _, ok := env["HORIZON_OAP_ZIPKIN_URL"]; ok {
			t.Error("HORIZON_OAP_ZIPKIN_URL was emitted for an unset address")
		}
		for _, m := range c.VolumeMounts {
			if m.MountPath == "/app/horizon.yaml" {
				t.Error("the baked config was overmounted when spec.config is unset")
			}
		}
	})

	t.Run("spec.env and spec.envFrom reach the container", func(t *testing.T) {
		var d appsv1.Deployment
		render(t, "ui", "deployment.yaml", ui(func(s *operatorv1alpha1.UISpec) {
			s.TemplatesMode = "readonly"
			s.Env = []corev1.EnvVar{{Name: "HORIZON_TRUST_PROXY", Value: "1"}}
			s.EnvFrom = []corev1.EnvFromSource{{
				SecretRef: &corev1.SecretEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: "horizon-secrets"},
				},
			}}
		}), &d)
		c := containerOf(t, &d)

		if envOf(c)["HORIZON_TRUST_PROXY"] != "1" {
			t.Error("spec.env did not reach the container")
		}
		if envOf(c)["HORIZON_TEMPLATES_MODE"] != "readonly" {
			t.Error("templatesMode was not honoured")
		}
		if len(c.EnvFrom) != 1 || c.EnvFrom[0].SecretRef == nil ||
			c.EnvFrom[0].SecretRef.Name != "horizon-secrets" {
			t.Errorf("spec.envFrom did not reach the container: %+v", c.EnvFrom)
		}
	})

	// An explicit value beats a derived one, and must not be emitted twice: duplicate names in a
	// container's env are accepted by the API server and resolved last-wins, so a duplicate would
	// silently depend on ordering.
	t.Run("an explicit value overrides the derived one exactly once", func(t *testing.T) {
		var d appsv1.Deployment
		render(t, "ui", "deployment.yaml", ui(func(s *operatorv1alpha1.UISpec) {
			s.Env = []corev1.EnvVar{{Name: "HORIZON_OAP_QUERY_URL", Value: "http://elsewhere:12800"}}
		}), &d)
		c := containerOf(t, &d)

		seen := 0
		for _, e := range c.Env {
			if e.Name == "HORIZON_OAP_QUERY_URL" {
				seen++
				if e.Value != "http://elsewhere:12800" {
					t.Errorf("derived value won: %q", e.Value)
				}
			}
		}
		if seen != 1 {
			t.Errorf("HORIZON_OAP_QUERY_URL appears %d times, want 1", seen)
		}
	})

	// A '#' in a value used to truncate the line, because the renderer cut every line at its first
	// one. See TestStripCommentKeepsHashesInsideValues.
	t.Run("a value containing a hash survives", func(t *testing.T) {
		var d appsv1.Deployment
		render(t, "ui", "deployment.yaml", ui(func(s *operatorv1alpha1.UISpec) {
			s.Env = []corev1.EnvVar{{Name: "HORIZON_AI_SYSTEM_PROMPT", Value: "explain #metrics"}}
		}), &d)
		if envOf(containerOf(t, &d))["HORIZON_AI_SYSTEM_PROMPT"] != "explain #metrics" {
			t.Error("the value was truncated at the hash")
		}
	})

	t.Run("spec.config is mounted over the baked file", func(t *testing.T) {
		var d appsv1.Deployment
		render(t, "ui", "deployment.yaml", ui(func(s *operatorv1alpha1.UISpec) {
			s.Config = "server:\n  port: 8081\n"
		}), &d)
		mounted := false
		for _, m := range containerOf(t, &d).VolumeMounts {
			if m.MountPath == "/app/horizon.yaml" {
				mounted = true
			}
		}
		if !mounted {
			t.Error("spec.config was set but no config was mounted")
		}
	})
}

// The ConfigMap exists only to carry a user-supplied file. With no spec.config it renders to
// nothing, which the renderer reports as ErrNothingLoaded and Application.Apply treats as "skip
// this manifest" -- so the UI reconciles normally without one.
func TestUIConfigMap(t *testing.T) {
	tmpl, err := NewRepo("ui").ReadFile("ui/templates/configmap.yaml")
	if err != nil {
		t.Fatalf("reading the template: %v", err)
	}
	if _, err := kubernetes.GenerateManifests(string(tmpl), ui(nil), nil); !errors.Is(err, kubernetes.ErrNothingLoaded) {
		t.Errorf("with no spec.config, rendering gave %v, want ErrNothingLoaded", err)
	}
	var cm corev1.ConfigMap
	render(t, "ui", "configmap.yaml", ui(func(s *operatorv1alpha1.UISpec) {
		s.Config = "server:\n  port: 8081\n"
	}), &cm)
	if !strings.Contains(cm.Data["horizon.yaml"], "port: 8081") {
		t.Errorf("spec.config did not reach the ConfigMap: %q", cm.Data["horizon.yaml"])
	}
}

func oap(mutate func(*operatorv1alpha1.OAPServerSpec)) *operatorv1alpha1.OAPServer {
	spec := operatorv1alpha1.OAPServerSpec{
		Version: "11.0.0", Instances: 1,
		Image:         "apache/skywalking-oap-server:11.0.0",
		StorageConfig: &operatorv1alpha1.RelevantStorage{Name: "storage"},
	}
	if mutate != nil {
		mutate(&spec)
	}
	return &operatorv1alpha1.OAPServer{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "skywalking-system"},
		Spec:       spec,
	}
}

func TestOAPServerDeployment(t *testing.T) {
	// spec.storage.name is mandatory, but InjectStorage fills in .Storage only once it can read
	// the resource. An OAPServer applied before its Storage leaves that nil, and reaching through
	// it used to abort the render -- so the OAP could not deploy until the ordering happened to
	// be right.
	t.Run("a storage name that does not resolve yet still renders", func(t *testing.T) {
		var d appsv1.Deployment
		render(t, "oapserver", "deployment.yaml", oap(nil), &d)
		containerOf(t, &d)
	})

	t.Run("envFrom and valueFrom reach the container", func(t *testing.T) {
		var d appsv1.Deployment
		render(t, "oapserver", "deployment.yaml", oap(func(s *operatorv1alpha1.OAPServerSpec) {
			s.Config = []corev1.EnvVar{{
				Name: "SW_STORAGE_BANYANDB_PASSWORD",
				ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: "bydb-auth"},
					Key:                  "password",
				}},
			}}
			s.EnvFrom = []corev1.EnvFromSource{{
				ConfigMapRef: &corev1.ConfigMapEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: "oap-tuning"},
				},
			}}
		}), &d)
		c := containerOf(t, &d)

		var found *corev1.EnvVar
		for i, e := range c.Env {
			if e.Name == "SW_STORAGE_BANYANDB_PASSWORD" {
				found = &c.Env[i]
			}
		}
		if found == nil || found.ValueFrom == nil || found.ValueFrom.SecretKeyRef == nil {
			t.Fatalf("valueFrom was dropped: %+v", found)
		}
		if found.ValueFrom.SecretKeyRef.Name != "bydb-auth" {
			t.Errorf("secretKeyRef = %+v", found.ValueFrom.SecretKeyRef)
		}
		if len(c.EnvFrom) != 1 || c.EnvFrom[0].ConfigMapRef == nil {
			t.Errorf("envFrom did not reach the container: %+v", c.EnvFrom)
		}
	})

	// The two storages verify TLS differently, and mounting the Elasticsearch keystore for a
	// BanyanDB left the pod waiting on a secret nothing creates.
	for _, tc := range []struct {
		name, storageType, tlsSecret, wantMount, wantSecret string
	}{
		{"banyandb", "banyandb", "bydb-ca", "/skywalking/bydb-tls", "bydb-ca"},
		{"banyandb without a named secret falls back", "banyandb", "", "/skywalking/bydb-tls", "skywalking-storage"},
		{"elasticsearch", "elasticsearch", "", "/skywalking/p12", "skywalking-storage"},
	} {
		t.Run("tls: "+tc.name, func(t *testing.T) {
			var d appsv1.Deployment
			render(t, "oapserver", "deployment.yaml", oap(func(s *operatorv1alpha1.OAPServerSpec) {
				s.StorageConfig.Storage = &operatorv1alpha1.Storage{
					ObjectMeta: metav1.ObjectMeta{Name: "storage", Namespace: "skywalking-system"},
					Spec: operatorv1alpha1.StorageSpec{
						Type:        tc.storageType,
						ConnectType: "external",
						Security: operatorv1alpha1.SecuritySpec{
							TLS: true, TLSSecretName: tc.tlsSecret,
						},
					},
				}
			}), &d)

			mounted := ""
			for _, m := range containerOf(t, &d).VolumeMounts {
				if strings.HasPrefix(m.MountPath, "/skywalking/") {
					mounted = m.MountPath
				}
			}
			if mounted != tc.wantMount {
				t.Errorf("mounted at %q, want %q", mounted, tc.wantMount)
			}
			secret := ""
			for _, v := range d.Spec.Template.Spec.Volumes {
				if v.Secret != nil {
					secret = v.Secret.SecretName
				}
			}
			if secret != tc.wantSecret {
				t.Errorf("secretName = %q, want %q", secret, tc.wantSecret)
			}
		})
	}
}

func TestSatelliteDeploymentEnvFrom(t *testing.T) {
	sat := &operatorv1alpha1.Satellite{
		ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "skywalking-system"},
		Spec: operatorv1alpha1.SatelliteSpec{
			Version: "v0.4.0", Instances: 1, Image: "apache/skywalking-satellite:v0.4.0",
			OAPServerName: "demo",
			EnvFrom: []corev1.EnvFromSource{{
				SecretRef: &corev1.SecretEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: "satellite-secrets"},
				},
			}},
		},
	}
	var d appsv1.Deployment
	render(t, "satellite", "deployment.yaml", sat, &d)
	c := containerOf(t, &d)
	if len(c.EnvFrom) != 1 || c.EnvFrom[0].SecretRef == nil ||
		c.EnvFrom[0].SecretRef.Name != "satellite-secrets" {
		t.Errorf("spec.envFrom did not reach the container: %+v", c.EnvFrom)
	}
}

// Environment variables taken from a Secret are resolved when the container starts and never
// refreshed, so rotating a credential does nothing until the pod restarts. The pod template
// carries the Secret's resourceVersion to make a rotation roll the pods.
func TestOAPServerCredentialRotationRollsPods(t *testing.T) {
	withVersion := func(v string) *appsv1.Deployment {
		var d appsv1.Deployment
		render(t, "oapserver", "deployment.yaml", oap(func(s *operatorv1alpha1.OAPServerSpec) {
			s.StorageConfig.CredentialsVersion = v
		}), &d)
		return &d
	}

	const key = "operator.skywalking.apache.org/storage-credentials-version"

	// No credentials, no annotation -- the pod template must not carry an empty one.
	if _, ok := withVersion("").Spec.Template.Annotations[key]; ok {
		t.Error("an empty credentials version produced an annotation")
	}

	before := withVersion("1234")
	if before.Spec.Template.Annotations[key] != "1234" {
		t.Fatalf("annotation = %q, want 1234", before.Spec.Template.Annotations[key])
	}

	// A rotation must change the pod template, or nothing rolls.
	after := withVersion("5678")
	if after.Spec.Template.Annotations[key] == before.Spec.Template.Annotations[key] {
		t.Error("rotating the Secret left the pod template unchanged")
	}

	// The credential itself must never appear in the template.
	if strings.Contains(fmt.Sprintf("%+v", after.Spec.Template), "password") &&
		!strings.Contains(fmt.Sprintf("%+v", after.Spec.Template), "SecretKeyRef") {
		t.Error("something that looks like a credential value reached the pod template")
	}
}
