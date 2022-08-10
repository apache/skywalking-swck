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
	"context"

	core "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	runtimelog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/apache/skywalking-swck/operator/pkg/kubernetes"
	"github.com/apache/skywalking-swck/operator/pkg/operator/injector"
)

// ConfigMapReconciler reconciles a ConfigMap object
type ConfigMapReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	FileRepo kubernetes.Repo
}

// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps/status,verbs=get;update;patch

func (r *ConfigMapReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := runtimelog.FromContext(ctx)
	log.Info("=====================configmap reconcile started================================")

	configmap := &core.ConfigMap{}
	err := r.Client.Get(ctx, client.ObjectKey{Namespace: injector.DefaultConfigmapNamespace, Name: injector.DefaultConfigmapName}, configmap)
	if err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, "failed to get configmap")
		return ctrl.Result{}, err
	}
	// if change the default configmap, we need to validate the value
	// if validate false , we will delete the configmap and recreate a default configmap
	if !apierrors.IsNotFound(err) {
		ok, errinfo := injector.ValidateConfigmap(configmap)
		if ok {
			return ctrl.Result{}, nil
		}
		log.Error(errinfo, "the default configmap validate false")
		if deleteErr := r.Client.Delete(ctx, configmap); deleteErr != nil {
			log.Error(deleteErr, "failed to delete the configmap that validate false")
			return ctrl.Result{}, deleteErr
		}
		log.Info("deleted the configmap that validate false")
	}
	app := kubernetes.Application{
		Client:   r.Client,
		FileRepo: r.FileRepo,
		CR:       configmap,
		GVK:      core.SchemeGroupVersion.WithKind("ConfigMap"),
		TmplFunc: injector.GetTmplFunc(),
	}

	// adding false means the configmap don't need to compose , such as ownerReferences
	ap, err := app.Apply(ctx, "injector/templates/configmap.yaml", log, false)
	if err != nil {
		log.Error(err, "failed to apply default configmap")
		return ctrl.Result{}, err
	}
	if ap {
		log.Info("create default configmap")
	}

	return ctrl.Result{}, nil
}

func (r *ConfigMapReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Only monitor the default configmap template
	return ctrl.NewControllerManagedBy(mgr).
		For(&core.ConfigMap{}).
		WithEventFilter(
			predicate.Funcs{
				CreateFunc: func(e event.CreateEvent) bool {
					if e.Object.GetNamespace() == injector.DefaultConfigmapNamespace &&
						e.Object.GetName() == injector.DefaultConfigmapName {
						return true
					}
					return false
				},
				UpdateFunc: func(e event.UpdateEvent) bool {
					if (e.ObjectNew.GetName() == injector.DefaultConfigmapName &&
						e.ObjectNew.GetNamespace() == injector.DefaultConfigmapNamespace) ||
						(e.ObjectOld.GetName() == injector.DefaultConfigmapName &&
							e.ObjectOld.GetNamespace() == injector.DefaultConfigmapNamespace) {
						return true
					}
					return false
				},
				DeleteFunc: func(e event.DeleteEvent) bool {
					if e.Object.GetNamespace() == injector.DefaultConfigmapNamespace &&
						e.Object.GetName() == injector.DefaultConfigmapName {
						return true
					}
					return false
				},
			}).
		Complete(r)
}
