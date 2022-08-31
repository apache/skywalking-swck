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
	"net/url"
	"time"

	"github.com/go-logr/logr"
	l "github.com/sirupsen/logrus"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	apiequal "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	runtimelog "sigs.k8s.io/controller-runtime/pkg/log"

	operatorv1alpha1 "github.com/apache/skywalking-swck/operator/apis/operator/v1alpha1"
	"github.com/apache/skywalking-swck/operator/pkg/kubernetes"
)

var schedDuration, _ = time.ParseDuration("1m")

// OAPServerReconciler reconciles a OAPServer object
type OAPServerReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	FileRepo kubernetes.Repo
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=oapservers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=oapservers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;create;update
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings,verbs=*
// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=storages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=storages/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *OAPServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := runtimelog.FromContext(ctx)
	log.Info("=====================oapserver reconcile started================================")

	oapServer := operatorv1alpha1.OAPServer{}
	if err := r.Client.Get(ctx, req.NamespacedName, &oapServer); err != nil {
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
		CR:       &oapServer,
		GVK:      operatorv1alpha1.GroupVersion.WithKind("OAPServer"),
		Recorder: r.Recorder,
	}

	r.InjectStorage(ctx, log, &oapServer)

	if err := app.ApplyAll(ctx, ff, log); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.checkState(ctx, log, &oapServer); err != nil {
		l.Error(err, "failed to check sub resources state")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: schedDuration}, nil
}

func (r *OAPServerReconciler) checkState(ctx context.Context, log logr.Logger, oapServer *operatorv1alpha1.OAPServer) error {
	overlay := operatorv1alpha1.OAPServerStatus{}
	deployment := apps.Deployment{}
	errCol := new(kubernetes.ErrorCollector)
	if err := r.Client.Get(ctx, client.ObjectKey{Namespace: oapServer.Namespace, Name: oapServer.Name + "-oap"}, &deployment); err != nil && !apierrors.IsNotFound(err) {
		errCol.Collect(fmt.Errorf("failed to get deployment: %w", err))
	} else {
		overlay.Conditions = deployment.Status.Conditions
		overlay.AvailableReplicas = deployment.Status.AvailableReplicas
	}
	service := core.Service{}
	if err := r.Client.Get(ctx, client.ObjectKey{Namespace: oapServer.Namespace, Name: oapServer.Name + "-oap"}, &service); err != nil && !apierrors.IsNotFound(err) {
		errCol.Collect(fmt.Errorf("failed to get service: %w", err))
	} else {
		overlay.Address = fmt.Sprintf("%s.%s", service.Name, service.Namespace)
	}
	if apiequal.Semantic.DeepDerivative(overlay, oapServer.Status) {
		log.Info("Status keeps the same as before")
	}

	if err := r.updateStatus(ctx, oapServer, overlay, errCol); err != nil {
		errCol.Collect(fmt.Errorf("failed to update status of oapServer: %w", err))
	}

	log.Info("updated Status sub resource")

	return errCol.Error()
}

func (r *OAPServerReconciler) updateStatus(ctx context.Context, oapServer *operatorv1alpha1.OAPServer,
	overlay operatorv1alpha1.OAPServerStatus, errCol *kubernetes.ErrorCollector) error {
	// avoid resource conflict
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		if err := r.Client.Get(ctx, client.ObjectKey{Name: oapServer.Name, Namespace: oapServer.Namespace}, oapServer); err != nil {
			errCol.Collect(fmt.Errorf("failed to get oapServer: %w", err))
		}
		oapServer.Status = overlay
		oapServer.Kind = "OAPServer"
		if err := kubernetes.ApplyOverlay(oapServer, &operatorv1alpha1.OAPServer{Status: overlay}); err != nil {
			errCol.Collect(fmt.Errorf("failed to apply overlay: %w", err))
		}
		if err := r.Status().Update(ctx, oapServer); err != nil {
			errCol.Collect(fmt.Errorf("failed to update status of OAPServer: %w", err))
		}
		return errCol.Error()
	})
}

//InjectStorage Inject Storage
func (r *OAPServerReconciler) InjectStorage(ctx context.Context, log logr.Logger, oapServer *operatorv1alpha1.OAPServer) {
	if oapServer.Spec.StorageConfig.Name == "" {
		return
	}
	storage := &operatorv1alpha1.Storage{}
	err := r.Client.Get(ctx, client.ObjectKey{Namespace: oapServer.Namespace, Name: oapServer.Spec.StorageConfig.Name}, storage)
	if err == nil {
		r.ConfigStorage(ctx, log, storage, oapServer)
		log.Info("success inject storage")
	} else {
		log.Info("fail inject storage")
	}
}

func (r *OAPServerReconciler) ConfigStorage(ctx context.Context, log logr.Logger, s *operatorv1alpha1.Storage, o *operatorv1alpha1.OAPServer) {
	user, tls := s.Spec.Security.User, s.Spec.Security.TLS
	SwStorageEsHTTPProtocol := "http"
	SwEsUser := ""
	SwEsPassword := ""
	SwStorageEsSslJksPath := ""
	SwStorageEsSslJksPass := "skywalking"
	SwStorageEsClusterNodes := ""
	o.Spec.StorageConfig.Storage = *s
	if user.SecretName != "" {
		if user.SecretName == "default" {
			SwEsUser = "elastic"
			SwEsPassword = "changeme"
		} else {
			usersecret := &core.Secret{}
			if err := r.Client.Get(ctx, client.ObjectKey{Namespace: s.Namespace, Name: user.SecretName}, usersecret); err != nil && !apierrors.IsNotFound(err) {
				log.Info("fail get usersecret ")
			}
			for k, v := range usersecret.Data {
				if k == "username" {
					SwEsUser = string(v)
				} else if k == "password" {
					SwEsPassword = string(v)
				}
			}
		}
	}
	if tls {
		SwStorageEsHTTPProtocol = "https"
		SwStorageEsSslJksPath = "/skywalking/p12/storage.p12"
		SwStorageEsClusterNodes = "skywalking-storage"
	} else {
		SwStorageEsClusterNodes = s.Name + "-" + s.Spec.Type
	}

	o.Spec.Config = append(o.Spec.Config, core.EnvVar{Name: "SW_STORAGE", Value: s.Spec.Type})
	if user.SecretName != "" {
		o.Spec.Config = append(o.Spec.Config, core.EnvVar{Name: "SW_ES_USER", Value: SwEsUser})
		o.Spec.Config = append(o.Spec.Config, core.EnvVar{Name: "SW_ES_PASSWORD", Value: SwEsPassword})
	}
	if tls {
		o.Spec.Config = append(o.Spec.Config, core.EnvVar{Name: "SW_STORAGE_ES_SSL_JKS_PATH", Value: SwStorageEsSslJksPath})
		o.Spec.Config = append(o.Spec.Config, core.EnvVar{Name: "SW_STORAGE_ES_SSL_JKS_PASS", Value: SwStorageEsSslJksPass})
	}
	if apiequal.Semantic.DeepDerivative(s.Spec.ConnectType, "external") {
		parseurl, _ := url.Parse(s.Spec.ConnectAddress)
		SwStorageEsHTTPProtocol = parseurl.Scheme
		SwStorageEsClusterNodes = parseurl.Host
		o.Spec.Config = append(o.Spec.Config, core.EnvVar{Name: "SW_STORAGE_ES_HTTP_PROTOCOL", Value: SwStorageEsHTTPProtocol})
		o.Spec.Config = append(o.Spec.Config, core.EnvVar{Name: "SW_STORAGE_ES_CLUSTER_NODES", Value: SwStorageEsClusterNodes})
	} else {
		o.Spec.Config = append(o.Spec.Config, core.EnvVar{Name: "SW_STORAGE_ES_HTTP_PROTOCOL", Value: SwStorageEsHTTPProtocol})
		o.Spec.Config = append(o.Spec.Config, core.EnvVar{Name: "SW_STORAGE_ES_CLUSTER_NODES", Value: SwStorageEsClusterNodes + ":9200"})
	}
}

func (r *OAPServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.OAPServer{}).
		Owns(&apps.Deployment{}).
		Owns(&core.Service{}).
		Complete(r)
}
