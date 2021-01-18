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
	"time"

	"github.com/go-logr/logr"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1alpha1 "github.com/apache/skywalking-swck/apis/operator/v1alpha1"
	"github.com/apache/skywalking-swck/pkg/kubernetes"
)

// FetcherReconciler reconciles a Fetcher object
type FetcherReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	FileRepo kubernetes.Repo
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=fetchers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=fetchers/status,verbs=get;update;patch

func (r *FetcherReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("fetcher", req.NamespacedName)
	log.Info("=====================reconcile started================================")

	fetcher := &operatorv1alpha1.Fetcher{}
	if err := r.Client.Get(ctx, req.NamespacedName, fetcher); err != nil {
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
		CR:       fetcher,
		GVK:      operatorv1alpha1.GroupVersion.WithKind("Fetcher"),
		Recorder: r.Recorder,
	}
	if err := app.ApplyAll(ctx, ff, log); err != nil {
		_ = r.updateStatus(ctx, fetcher, core.ConditionFalse, "Failed to apply resources")
		return ctrl.Result{}, err
	}
	if err := r.updateStatus(ctx, fetcher, core.ConditionTrue, "Reconciled all of resources"); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: schedDuration}, nil
}

func (r *FetcherReconciler) updateStatus(ctx context.Context, fetcher *operatorv1alpha1.Fetcher, status core.ConditionStatus, msg string) error {
	if fetcher.Status.Replicas == 0 {
		fetcher.Status.Replicas = 1
	}
	now := metav1.NewTime(time.Now())
	changed := false
	if len(fetcher.Status.Conditions) < 1 {
		changed = true
		fetcher.Status.Conditions = append(fetcher.Status.Conditions, operatorv1alpha1.FetcherCondition{
			Type:               operatorv1alpha1.FetcherConditionTypeRead,
			Status:             status,
			Message:            msg,
			LastTransitionTime: now,
			LastUpdateTime:     now,
		})
	} else {
		current := fetcher.Status.Conditions[0]
		if current.Status != status || current.Message != msg {
			changed = true
			current.Status = status
			current.Message = msg
			current.LastUpdateTime = now
		}
	}
	if !changed {
		return nil
	}
	if err := r.Status().Update(ctx, fetcher); err != nil {
		r.Log.Error(err, "failed to update status")
		return err
	}
	return nil
}

func (r *FetcherReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.Fetcher{}).
		Owns(&apps.Deployment{}).
		Owns(&core.ConfigMap{}).
		Complete(r)
}
