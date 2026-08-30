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

// SkyWalking removed H2 in 10.2.0, so BanyanDB is what an OAP runs on unless someone brings an
// Elasticsearch. This resource has to accept it -- it used to reject everything but elasticsearch.
func TestStorageValidateType(t *testing.T) {
	tests := []struct {
		name        string
		spec        StorageSpec
		wantErr     bool
		errContains string
	}{
		{
			name: "elasticsearch, provisioned here",
			spec: StorageSpec{Type: StorageTypeElasticsearch, ConnectType: "internal", Version: "8.18.8"},
		},
		{
			name: "banyandb, pointing at one the BanyanDB resource deployed",
			spec: StorageSpec{
				Type: StorageTypeBanyanDB, ConnectType: "external",
				ConnectAddress: "storage-banyandb-grpc.skywalking-system:17912", Version: "0.11.0",
			},
		},
		{
			// The BanyanDB resource provisions the database; this one would have nothing to do.
			name:    "banyandb cannot be internal",
			spec:    StorageSpec{Type: StorageTypeBanyanDB, ConnectType: "internal", Version: "0.11.0"},
			wantErr: true, errContains: "external",
		},
		{
			name:    "banyandb needs somewhere to point",
			spec:    StorageSpec{Type: StorageTypeBanyanDB, ConnectType: "external", Version: "0.11.0"},
			wantErr: true, errContains: "17912",
		},
		{
			// H2 is gone from SkyWalking entirely; saying so beats "must be elasticsearch".
			name:    "h2 is refused",
			spec:    StorageSpec{Type: "h2", ConnectType: "internal"},
			wantErr: true, errContains: "H2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &Storage{
				ObjectMeta: metav1.ObjectMeta{Name: "storage", Namespace: "skywalking-system"},
				Spec:       tt.spec,
			}
			_, err := storage.ValidateCreate(context.Background(), storage)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected a rejection")
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("rejection does not mention %q: %v", tt.errContains, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("rejected: %v", err)
			}
		})
	}
}

// A banyandb Storage is a pointer, so it must not pick up the Elasticsearch image default.
func TestStorageDefaultDoesNotGiveBanyanDBAnImage(t *testing.T) {
	storage := &Storage{
		ObjectMeta: metav1.ObjectMeta{Name: "storage", Namespace: "skywalking-system"},
		Spec:       StorageSpec{Type: StorageTypeBanyanDB, ConnectType: "internal"},
	}
	if err := storage.Default(context.Background(), storage); err != nil {
		t.Fatalf("Default() returned %v", err)
	}
	if storage.Spec.Image != "" {
		t.Errorf("banyandb storage was given an image: %q", storage.Spec.Image)
	}
}

// TLS to BanyanDB needs a CA file, and the operator has no conventional secret to fall back on.
// Accepting tls without a secret name left the OAP pod waiting on one nothing creates.
func TestStorageBanyanDBTLSRequiresASecret(t *testing.T) {
	banyandb := func(mutate func(*StorageSpec)) *Storage {
		spec := StorageSpec{
			Type:           StorageTypeBanyanDB,
			ConnectType:    "external",
			ConnectAddress: "storage-banyandb-grpc.skywalking-system:17912",
			Version:        "0.11.0",
		}
		if mutate != nil {
			mutate(&spec)
		}
		return &Storage{
			ObjectMeta: metav1.ObjectMeta{Name: "storage", Namespace: "skywalking-system"},
			Spec:       spec,
		}
	}

	t.Run("tls without a secret name is refused", func(t *testing.T) {
		err := banyandb(func(s *StorageSpec) { s.Security.TLS = true }).valid()
		if err == nil {
			t.Fatal("banyandb with tls and no tlsSecretName was admitted")
		}
		if !strings.Contains(err.Error(), "tlsSecretName") {
			t.Errorf("the message does not name the field: %v", err)
		}
	})

	t.Run("tls with a secret name is accepted", func(t *testing.T) {
		err := banyandb(func(s *StorageSpec) {
			s.Security.TLS = true
			s.Security.TLSSecretName = "banyandb-ca"
		}).valid()
		if err != nil {
			t.Errorf("rejected a valid TLS configuration: %v", err)
		}
	})

	t.Run("no tls needs no secret name", func(t *testing.T) {
		if err := banyandb(nil).valid(); err != nil {
			t.Errorf("rejected a plaintext banyandb storage: %v", err)
		}
	})
}
