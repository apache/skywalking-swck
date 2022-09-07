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
	"time"

	"github.com/go-logr/logr"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	runtimelog "sigs.k8s.io/controller-runtime/pkg/log"

	operatorv1alpha1 "github.com/apache/skywalking-swck/operator/apis/operator/v1alpha1"
	"github.com/apache/skywalking-swck/operator/pkg/kubernetes"
)

// FetcherReconciler reconciles a Fetcher object
type FetcherReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	FileRepo kubernetes.Repo
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=fetchers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=fetchers/status,verbs=get;update;patch

func (r *FetcherReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := runtimelog.FromContext(ctx)
	log.Info("=====================fetcher reconcile started================================")

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
		_ = r.UpdateStatus(ctx, fetcher, core.ConditionFalse, "Failed to apply resources")
		return ctrl.Result{}, err
	}
	if err := r.UpdateStatus(ctx, fetcher, core.ConditionTrue, "Reconciled all of resources"); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: schedDuration}, nil
}

func (r *FetcherReconciler) UpdateStatus(ctx context.Context, fetcher *operatorv1alpha1.Fetcher, status core.ConditionStatus, msg string) error {
	log := runtimelog.FromContext(ctx)

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
	overlay := fetcher.Status
	if err := r.updateStatus(ctx, fetcher, overlay, log); err != nil {
		log.Error(err, "failed to update status")
		return err
	}
	return nil
}

func (r *FetcherReconciler) updateStatus(ctx context.Context, fetcher *operatorv1alpha1.Fetcher,
	overlay operatorv1alpha1.FetcherStatus, log logr.Logger) error {
	// avoid resource conflict
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		if err := r.Client.Get(ctx, client.ObjectKey{Name: fetcher.Name, Namespace: fetcher.Namespace}, fetcher); err != nil {
			log.Error(err, "failed to get fetcher")
		}
		fetcher.Status = overlay
		fetcher.Kind = "Fetcher"
		if err := kubernetes.ApplyOverlay(fetcher, &operatorv1alpha1.Fetcher{Status: overlay}); err != nil {
			log.Error(err, "failed to apply overlay")
		}
		if err := r.Status().Update(ctx, fetcher); err != nil {
			log.Error(err, "failed to update status of fetcher")
		}
		return nil
	})
}

func (r *FetcherReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.Fetcher{}).
		Owns(&apps.Deployment{}).
		Owns(&core.ConfigMap{}).
		Complete(r)
}
