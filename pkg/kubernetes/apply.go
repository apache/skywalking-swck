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
	"context"
	"fmt"
	"text/template"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Repo provides tools to access templates
type Repo interface {
	ReadFile(path string) ([]byte, error)
	GetFilesRecursive(path string) ([]string, error)
}

// Application contains the resource of one single component which is applied to api server
type Application struct {
	client.Client
	CR       client.Object
	FileRepo Repo
	GVK      schema.GroupVersionKind
	TmplFunc template.FuncMap
}

// Apply a template represents a component to api server
func (a *Application) Apply(ctx context.Context, manifest string, log logr.Logger) error {
	manifests, err := a.FileRepo.ReadFile(manifest)
	if err != nil {
		return err
	}
	proto := &unstructured.Unstructured{}
	err = LoadTemplate(string(manifests), a.CR, a.TmplFunc, proto)
	if err == ErrNothingLoaded {
		log.Info("nothing is loaded")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to load %s template: %w", manifest, err)
	}
	return a.apply(ctx, proto, log)
}

// ApplyFromObject apply an object to api server
func (a *Application) ApplyFromObject(ctx context.Context, obj runtime.Object, log logr.Logger) error {
	proto, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return fmt.Errorf("failed to convert object to unstructed: %v", err)
	}
	return a.apply(ctx, &unstructured.Unstructured{Object: proto}, log)
}

func (a *Application) apply(ctx context.Context, obj *unstructured.Unstructured, log logr.Logger) error {
	key := client.ObjectKeyFromObject(obj)
	current := &unstructured.Unstructured{}
	current.SetGroupVersionKind(obj.GetObjectKind().GroupVersionKind())
	err := a.Get(ctx, key, current)
	if apierrors.IsNotFound(err) {
		log.Info("could not find existing resource, creating one...")
		curr, errComp := a.compose(obj)
		if errComp != nil {
			return fmt.Errorf("failed to compose: %w", errComp)
		}

		if err = a.Create(ctx, curr); err != nil {
			return fmt.Errorf("failed to create: %w", err)
		}

		log.Info("created")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get %v : %w", key, err)
	}

	object, err := a.compose(obj)
	if err != nil {
		return fmt.Errorf("failed to compose: %w", err)
	}

	if getVersion(current, a.versionKey()) == getVersion(object, a.versionKey()) {
		log.Info("resource keeps the same as before")
		return nil
	}
	if err := a.Update(ctx, object); err != nil {
		return fmt.Errorf("failed to update: %w", err)
	}
	log.Info("updated")
	return nil
}

func (a *Application) setVersionAnnotation(o *unstructured.Unstructured) error {
	h, err := hash(o)
	if err != nil {
		return err
	}
	setVersion(o, a.versionKey(), h)
	return nil
}

func (a *Application) versionKey() string {
	return a.GVK.Group + "/version"
}

func (a *Application) compose(object *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	object.SetOwnerReferences([]metav1.OwnerReference{*metav1.NewControllerRef(a.CR, a.GVK)})
	err := a.setVersionAnnotation(object)
	if err != nil {
		return nil, err
	}
	return object, nil
}
