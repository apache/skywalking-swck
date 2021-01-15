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
	"fmt"

	"github.com/go-logr/logr"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	apiequal "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	uiv1alpha1 "github.com/apache/skywalking-swck/apis/operator/v1alpha1"
	"github.com/apache/skywalking-swck/pkg/kubernetes"
)

// UIReconciler reconciles a UI object
type UIReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	FileRepo kubernetes.Repo
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=uis,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=uis/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete

func (r *UIReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("ui", req.NamespacedName)
	log.Info("=====================reconcile started================================")

	ui := uiv1alpha1.UI{}
	if err := r.Client.Get(ctx, req.NamespacedName, &ui); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	ff, err := r.FileRepo.GetFilesRecursive("templates")
	if err != nil {
		log.Error(err, "failed to load resource templates")
		return ctrl.Result{}, err
	}
	app := kubernetes.Application{
		Client:   r.Client,
		FileRepo: r.FileRepo,
		CR:       &ui,
		GVK:      uiv1alpha1.GroupVersion.WithKind("UI"),
		Recorder: r.Recorder,
	}
	if err := app.ApplyAll(ctx, ff, log); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.checkState(ctx, log, &ui); err != nil {
		log.Error(err, "failed to check sub resources state")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: schedDuration}, nil
}

func (r *UIReconciler) checkState(ctx context.Context, log logr.Logger, ui *uiv1alpha1.UI) error {
	overlay := uiv1alpha1.UIStatus{}
	deployment := apps.Deployment{}
	errCol := new(kubernetes.ErrorCollector)
	if err := r.Client.Get(ctx, client.ObjectKey{Namespace: ui.Namespace, Name: ui.Name + "-ui"}, &deployment); err != nil && !apierrors.IsNotFound(err) {
		errCol.Collect(fmt.Errorf("failed to get deployment: %w", err))
	} else {
		overlay.Conditions = deployment.Status.Conditions
		overlay.AvailableReplicas = deployment.Status.AvailableReplicas
	}
	svc := core.Service{}
	if err := r.Client.Get(ctx, client.ObjectKey{Namespace: ui.Namespace, Name: ui.Name + "-ui"}, &svc); err != nil && !apierrors.IsNotFound(err) {
		errCol.Collect(fmt.Errorf("failed to get svc: %w", err))
	} else {
		for _, i := range svc.Status.LoadBalancer.Ingress {
			overlay.ExternalIPs = append(overlay.ExternalIPs, i.IP)
		}
		for _, p := range svc.Spec.Ports {
			overlay.Ports = append(overlay.Ports, p.Port)
		}
		if len(overlay.Ports) < 1 {
			overlay.Ports = []int32{0}
		}
		overlay.InternalAddress = fmt.Sprintf("%s.%s", svc.Name, svc.Namespace)
	}
	if apiequal.Semantic.DeepDerivative(overlay, ui.Status) {
		log.Info("Status keeps the same as before")
	}
	ui.Status = overlay
	ui.Kind = "UI"
	if err := kubernetes.ApplyOverlay(ui, &uiv1alpha1.UI{Status: overlay}); err != nil {
		errCol.Collect(fmt.Errorf("failed to apply overlay: %w", err))
		return errCol.Error()
	}
	if err := r.Status().Update(ctx, ui); err != nil {
		errCol.Collect(fmt.Errorf("failed to update status of UI: %w", err))
	}
	log.Info("updated Status sub resource")

	return errCol.Error()
}

func (r *UIReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&uiv1alpha1.UI{}).
		Owns(&apps.Deployment{}).
		Owns(&core.Service{}).
		Owns(&networkingv1beta1.Ingress{}).
		Complete(r)
}
