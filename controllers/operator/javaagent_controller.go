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

package controllers

import (
	"context"
	"strings"

	"github.com/go-logr/logr"
	core "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	operatorv1alpha1 "github.com/apache/skywalking-swck/apis/operator/v1alpha1"
	"github.com/apache/skywalking-swck/pkg/kubernetes"
	"github.com/apache/skywalking-swck/pkg/operator/injector"
)

// JavaAgentReconciler reconciles a JavaAgent object
type JavaAgentReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	FileRepo kubernetes.Repo
}

// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=javaagents,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=javaagents/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete

func (r *JavaAgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("javaagent", req.NamespacedName)
	log.Info("=====================javaagent started================================")

	pod := &core.Pod{}
	err := r.Client.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: req.Name}, pod)
	if err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, "failed to get pod")
		return ctrl.Result{}, err
	}

	configmap := &core.ConfigMap{}
	configmapName := ""
	for i := range pod.Spec.Volumes {
		if pod.Spec.Volumes[i].ConfigMap != nil {
			configmapName = pod.Spec.Volumes[i].ConfigMap.Name
		}
	}

	if configmapName == "" {
		log.Error(err, "configmap is nil")
		return ctrl.Result{}, err
	}

	err = r.Client.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: configmapName}, configmap)
	if err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, "failed to get configmap")
		return ctrl.Result{}, err
	}

	config, err := injector.GetConfigmapConfiguration(configmap)
	if err != nil {
		log.Error(err, "failed to get configmap's configuration")
		return ctrl.Result{}, err
	}
	injector.GetInjectedAgentConfig(&pod.Annotations, &config)

	app := kubernetes.Application{
		Client:   r.Client,
		FileRepo: r.FileRepo,
		CR:       pod,
		GVK:      operatorv1alpha1.GroupVersion.WithKind("JavaAgent"),
		TmplFunc: map[string]interface{}{
			"config": func() map[string]string {
				return config
			},
			"req": func() ctrl.Request {
				return req
			},
			"ServiceName": func() string {
				return injector.GetServiceName(&config)
			},
			"BackendService": func() string {
				return injector.GetBackendService(&config)
			},
		},
	}

	// true means need to compose , such as ownerReferences
	_, err = app.Apply(ctx, "injector/templates/javaagent.yaml", log, true)
	if err != nil {
		log.Error(err, "failed to apply javaagent")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *JavaAgentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// handle the injected pod
	return ctrl.NewControllerManagedBy(mgr).
		For(&core.Pod{}).
		WithEventFilter(predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool {
				annotations := e.Object.GetAnnotations()
				if annotations != nil && strings.ToLower(annotations[injector.SidecarInjectSucceedAnno]) == "true" {
					return true
				}
				return false
			},
			UpdateFunc: func(e event.UpdateEvent) bool {
				annotations := e.ObjectNew.GetAnnotations()
				if annotations != nil && strings.ToLower(annotations[injector.SidecarInjectSucceedAnno]) == "true" {
					return true
				}
				return false
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				return false
			},
		}).
		Owns(&operatorv1alpha1.JavaAgent{}).
		Complete(r)
}
