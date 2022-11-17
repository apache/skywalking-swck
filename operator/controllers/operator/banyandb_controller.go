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
	"fmt"

	"github.com/go-logr/logr"
	apps "k8s.io/api/apps/v1"
	apiequal "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	runtimelog "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/apache/skywalking-swck/operator/apis/operator/v1alpha1"
	operatorv1alpha1 "github.com/apache/skywalking-swck/operator/apis/operator/v1alpha1"
	"github.com/apache/skywalking-swck/operator/pkg/kubernetes"
)

// BanyanDBReconciler reconciles a BanyanDB object
type BanyanDBReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	FileRepo kubernetes.Repo
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=banyandbs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=banyandbs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=banyandbs/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=services;serviceaccounts;persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings,verbs=*

func (r *BanyanDBReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := runtimelog.FromContext(ctx)
	log.Info("=====================reconcile started================================")

	var banyanDB v1alpha1.BanyanDB
	if err := r.Get(ctx, req.NamespacedName, &banyanDB); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	ff, err := r.FileRepo.GetFilesRecursive("templates")
	if err != nil {
		log.Error(err, "failed to load resource templates")
		return ctrl.Result{}, err
	}
	app := kubernetes.Application{
		Client:   r.Client,
		CR:       &banyanDB,
		FileRepo: r.FileRepo,
		GVK:      operatorv1alpha1.GroupVersion.WithKind("BanyanDB"),
		Recorder: r.Recorder,
	}

	if err := app.ApplyAll(ctx, ff, log); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.checkState(ctx, log, &banyanDB); err != nil {
		log.Error(err, "failed to check sub resources state")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: schedDuration}, nil
}

func (r *BanyanDBReconciler) checkState(ctx context.Context, log logr.Logger, banyanDB *operatorv1alpha1.BanyanDB) error {
	overlay := operatorv1alpha1.BanyanDBStatus{}
	deployment := apps.Deployment{}
	errCol := new(kubernetes.ErrorCollector)
	if err := r.Client.Get(ctx, client.ObjectKey{Namespace: banyanDB.Namespace, Name: banyanDB.Name + "-banyandb"}, &deployment); err != nil && !apierrors.IsNotFound(err) {
		errCol.Collect(fmt.Errorf("failed to get deployment: %w", err))
	} else {
		overlay.Conditions = deployment.Status.Conditions
		overlay.AvailableReplicas = deployment.Status.AvailableReplicas
	}
	if apiequal.Semantic.DeepDerivative(overlay, banyanDB.Status) {
		log.Info("Status keeps the same as before")
		return errCol.Error()
	}

	if err := r.updateStatus(ctx, banyanDB, overlay, errCol); err != nil {
		errCol.Collect(fmt.Errorf("failed to update status of banyanDB: %w", err))
	}

	log.Info("updated Status sub resource")

	return errCol.Error()
}

func (r *BanyanDBReconciler) updateStatus(ctx context.Context, banyanDB *operatorv1alpha1.BanyanDB,
	overlay operatorv1alpha1.BanyanDBStatus, errCol *kubernetes.ErrorCollector) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		if err := r.Client.Get(ctx, client.ObjectKey{Namespace: banyanDB.Namespace, Name: banyanDB.Name}, banyanDB); err != nil {
			errCol.Collect(fmt.Errorf("failed to get banyanDB: %w", err))
		}
		banyanDB.Status = overlay
		banyanDB.Kind = "BanyanDB"
		if err := kubernetes.ApplyOverlay(banyanDB, &operatorv1alpha1.BanyanDB{Status: overlay}); err != nil {
			errCol.Collect(fmt.Errorf("failed to apply overlay: %w", err))
		}
		if err := r.Status().Update(ctx, banyanDB); err != nil {
			errCol.Collect(fmt.Errorf("failed to update status of banyandb: %w", err))
		}
		return errCol.Error()
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *BanyanDBReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.BanyanDB{}).
		Complete(r)
}
