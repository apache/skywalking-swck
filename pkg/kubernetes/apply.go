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
	apiequal "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Repo provides tools to access templates
type Repo interface {
	ReadFile(path string) ([]byte, error)
}

// Application contains the resource of one single component which is applied to api server
type Application struct {
	client.Client
	CR       client.Object
	Log      logr.Logger
	FileRepo Repo
	GVK      schema.GroupVersionKind
}

// K8SObj is a sub template of an Application
type K8SObj struct {
	Name      string
	Key       client.ObjectKey
	Prototype client.Object
	TmplFunc  template.FuncMap
	Extract   ExtractFunc
}

type ExtractFunc func(obj client.Object) interface{}

// Apply a template represents a component to api server
func (a *Application) Apply(ctx context.Context, input K8SObj) error {
	log := a.Log.WithName(input.Name)
	manifests, err := a.FileRepo.ReadFile(fmt.Sprintf("%s.yaml", input.Name))
	if err != nil {
		return err
	}
	overlay := input.Prototype.DeepCopyObject().(client.Object)
	current := input.Prototype.DeepCopyObject().(client.Object)
	err = a.Get(ctx, input.Key, current)
	if apierrors.IsNotFound(err) {
		log.Info("could not find existing resource, creating one...")

		curr, errComp := compose(string(manifests), current, overlay, a.CR, a.GVK, input.TmplFunc)
		if errComp != nil {
			return errComp
		}

		if err = a.Create(ctx, curr); err != nil {
			return err
		}

		log.Info("created resource")
		return nil
	}
	if err != nil {
		return err
	}
	if input.Extract == nil {
		return nil
	}

	object, err := compose(string(manifests), current, overlay, a.CR, a.GVK, input.TmplFunc)
	if err != nil {
		return err
	}

	if apiequal.Semantic.DeepDerivative(input.Extract(object), input.Extract(current)) {
		log.Info("resource keeps the same as before")
		return nil
	}
	if err := a.Update(ctx, object); err != nil {
		return err
	}
	log.Info("updated resource")
	return nil
}

func compose(manifests string, base, overlay, cr client.Object,
	gvk schema.GroupVersionKind, tmplFunc template.FuncMap) (client.Object, error) {
	err := LoadTemplate(manifests, cr, tmplFunc, overlay)
	if err != nil {
		return nil, err
	}

	overlay.SetOwnerReferences([]metav1.OwnerReference{*metav1.NewControllerRef(cr, gvk)})
	if base == nil {
		return overlay, nil
	}
	if err := ApplyOverlay(base, overlay); err != nil {
		return nil, err
	}
	return base, nil
}
