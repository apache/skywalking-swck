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

// Package v1alpha1 contains API Schema definitions for the operator v1alpha1 API group
// +kubebuilder:object:generate=true
// +groupName=operator.skywalking.apache.org
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: "operator.skywalking.apache.org", Version: "v1alpha1"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &groupVersionSchemeBuilder{gv: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

// groupVersionSchemeBuilder registers API types into a scheme without depending
// on sigs.k8s.io/controller-runtime/pkg/scheme (which is deprecated for api packages).
type groupVersionSchemeBuilder struct {
	gv    schema.GroupVersion
	funcs runtime.SchemeBuilder
}

// Register adds objects to the scheme under the package's GroupVersion.
func (b *groupVersionSchemeBuilder) Register(objects ...runtime.Object) *groupVersionSchemeBuilder {
	b.funcs = append(b.funcs, func(s *runtime.Scheme) error {
		s.AddKnownTypes(b.gv, objects...)
		metav1.AddToGroupVersion(s, b.gv)
		return nil
	})
	return b
}

// AddToScheme runs all registered scheme-builder functions against the given scheme.
func (b *groupVersionSchemeBuilder) AddToScheme(s *runtime.Scheme) error {
	return b.funcs.AddToScheme(s)
}
