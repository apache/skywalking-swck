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
	"strings"
	"time"

	"github.com/go-logr/logr"
	l "github.com/sirupsen/logrus"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	apiequal "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	runtimelog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	operatorv1alpha1 "github.com/apache/skywalking-swck/operator/apis/operator/v1alpha1"
	"github.com/apache/skywalking-swck/operator/pkg/kubernetes"
)

var schedDuration, _ = time.ParseDuration("1m")

// OAPServerReconciler reconciles a OAPServer object
type OAPServerReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	FileRepo kubernetes.Repo
	Recorder events.EventRecorder
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

	// A named Storage that cannot be read must stop the reconcile, not be reconciled around. The
	// rendered Deployment would otherwise carry no SW_STORAGE, no targets, no credentials and no
	// TLS volume -- and applying it rolls a healthy OAP into one that never becomes ready, for as
	// long as the Storage is missing. Requeue instead and leave the running Deployment alone;
	// a Storage deleted and recreated, or created after the OAPServer, then heals on its own.
	if err := r.InjectStorage(ctx, &oapServer); err != nil {
		log.Error(err, "cannot resolve the storage this OAPServer names; leaving the current "+
			"deployment in place and retrying")
		r.Recorder.Eventf(&oapServer, nil, core.EventTypeWarning, "StorageUnresolved", "Reconcile",
			"storage %q could not be read: %v", oapServer.Spec.StorageConfig.Name, err)
		return ctrl.Result{RequeueAfter: schedDuration}, nil
	}

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
		return errCol.Error()
	}

	if err := r.updateStatus(ctx, oapServer, overlay, errCol); err != nil {
		errCol.Collect(fmt.Errorf("failed to update status of oapServer: %w", err))
	}

	log.Info("updated Status sub resource")

	return errCol.Error()
}

func (r *OAPServerReconciler) updateStatus(ctx context.Context, oapServer *operatorv1alpha1.OAPServer,
	overlay operatorv1alpha1.OAPServerStatus, errCol *kubernetes.ErrorCollector,
) error {
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

// InjectStorage Inject Storage
// InjectStorage resolves the Storage an OAPServer names and folds its configuration into the
// OAPServer's environment. An OAPServer that names no storage is left alone -- the webhook
// already refuses one that has neither a name nor SW_STORAGE in spec.config.
func (r *OAPServerReconciler) InjectStorage(ctx context.Context, oapServer *operatorv1alpha1.OAPServer) error {
	if oapServer.Spec.StorageConfig == nil || oapServer.Spec.StorageConfig.Name == "" {
		return nil
	}
	storage := &operatorv1alpha1.Storage{}
	key := client.ObjectKey{Namespace: oapServer.Namespace, Name: oapServer.Spec.StorageConfig.Name}
	if err := r.Client.Get(ctx, key, storage); err != nil {
		return fmt.Errorf("reading storage %s/%s: %w", key.Namespace, key.Name, err)
	}
	r.ConfigStorage(storage, oapServer)

	// Record which version of the credential Secret this render is against, so a rotation reaches
	// the running pods. A missing Secret is not an error here -- the references are optional and
	// the OAP will report the auth failure -- so the version is simply left empty.
	if name := storage.Spec.Security.User.SecretName; name != "" && name != "default" {
		secret := &core.Secret{}
		key := client.ObjectKey{Namespace: storage.Namespace, Name: name}
		if err := r.Client.Get(ctx, key, secret); err == nil {
			oapServer.Spec.StorageConfig.CredentialsVersion = secret.ResourceVersion
		}
	}
	return nil
}

// The gRPC port a BanyanDB deployed by the BanyanDB resource listens on
// (operator/pkg/operator/manifests/banyandb/templates/grpc_service.yaml).
const (
	banyanDBGRPCPort = "17912"
	// Where the deployment template mounts the Secret named by Storage.spec.security.tlsSecretName.
	banyanDBTLSMountPath = "/skywalking/bydb-tls"
)

func (r *OAPServerReconciler) ConfigStorage(s *operatorv1alpha1.Storage, o *operatorv1alpha1.OAPServer) {
	// BanyanDB is the OAP's default storage and, H2 having been removed in 10.2.0, the only one that
	// needs no external system to stand up. It is configured through bydb.yml rather than
	// application.yml,
	// and the one setting that matters here is the target list.
	if s.Spec.Type == operatorv1alpha1.StorageTypeBanyanDB {
		r.configBanyanDBStorage(s, o)
		return
	}
	user, tls := s.Spec.Security.User, s.Spec.Security.TLS
	SwStorageEsHTTPProtocol := "http"
	SwStorageEsSslJksPath := ""
	SwStorageEsSslJksPass := "skywalking"
	SwStorageEsClusterNodes := ""
	o.Spec.StorageConfig.Storage = s
	if tls {
		SwStorageEsHTTPProtocol = "https"
		SwStorageEsSslJksPath = "/skywalking/p12/storage.p12"
		SwStorageEsClusterNodes = "skywalking-storage"
	} else {
		SwStorageEsClusterNodes = s.Name + "-" + s.Spec.Type
	}

	o.Spec.Config = append(o.Spec.Config, core.EnvVar{Name: "SW_STORAGE", Value: s.Spec.Type})
	if user.SecretName != "" {
		if user.SecretName == "default" {
			// The well-known Elasticsearch starter pair, documented rather than stored -- there is
			// no Secret named "default" to reference.
			o.Spec.Config = append(o.Spec.Config,
				core.EnvVar{Name: "SW_ES_USER", Value: "elastic"},
				core.EnvVar{Name: "SW_ES_PASSWORD", Value: "changeme"},
			)
		} else {
			o.Spec.Config = append(o.Spec.Config,
				secretEnv("SW_ES_USER", user.SecretName, "username"),
				secretEnv("SW_ES_PASSWORD", user.SecretName, "password"),
			)
		}
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

// configBanyanDBStorage points the OAP at a BanyanDB.
//
// SW_STORAGE_BANYANDB_TARGETS is a host:port list; there is no separate host setting. The
// SW_STORAGE_BANYANDB_HOST that SkyWalking 9.x took was removed in 11.0.0 along with the rest of
// the banyandb block in application.yml, so an OAPServer that still carries it silently falls back
// to 127.0.0.1:17912 and never becomes ready.
func (r *OAPServerReconciler) configBanyanDBStorage(s *operatorv1alpha1.Storage, o *operatorv1alpha1.OAPServer) {
	o.Spec.StorageConfig.Storage = s

	targets := s.Spec.ConnectAddress
	if targets == "" {
		// An internal BanyanDB is one the BanyanDB resource of the same name deployed beside it;
		// its gRPC Service is <name>-banyandb-grpc.
		targets = fmt.Sprintf("%s-banyandb-grpc.%s:%s", s.Name, s.Namespace, banyanDBGRPCPort)
	} else if !strings.Contains(targets, ":") {
		targets = targets + ":" + banyanDBGRPCPort
	}

	o.Spec.Config = append(o.Spec.Config,
		core.EnvVar{Name: "SW_STORAGE", Value: operatorv1alpha1.StorageTypeBanyanDB},
		core.EnvVar{Name: "SW_STORAGE_BANYANDB_TARGETS", Value: targets},
	)

	// TLS to BanyanDB. The CA comes from the named Secret, mounted by the deployment template at
	// banyanDBTLSMountPath; the OAP wants a path, not the certificate itself.
	if s.Spec.Security.TLS {
		o.Spec.Config = append(o.Spec.Config, core.EnvVar{
			Name:  "SW_STORAGE_BANYANDB_SSL_TRUST_CA_PATH",
			Value: banyanDBTLSMountPath + "/ca.crt",
		})
	}

	// BanyanDB authentication, if the Storage names a secret holding it. Referenced, not read:
	// see secretEnv.
	if s.Spec.Security.User.SecretName != "" {
		o.Spec.Config = append(o.Spec.Config,
			secretEnv("SW_STORAGE_BANYANDB_USER", s.Spec.Security.User.SecretName, "username"),
			secretEnv("SW_STORAGE_BANYANDB_PASSWORD", s.Spec.Security.User.SecretName, "password"),
		)
	}
}

// secretEnv points an environment variable at a key in a Secret, so the value is read by the
// kubelet when the pod starts.
//
// The alternative -- fetching the Secret here and appending the value -- writes the credential into
// the OAPServer's spec.config and from there into the Deployment's pod template, where any of the
// many subjects with read access on those objects can see it in `kubectl get -o yaml`. That defeats
// the Secret the user went to the trouble of creating.
//
// Optional, so a missing key leaves the variable unset and the OAP reports an auth failure, rather
// than the pod refusing to start with a message about the Secret.
func secretEnv(name, secretName, key string) core.EnvVar {
	optional := true
	return core.EnvVar{
		Name: name,
		ValueFrom: &core.EnvVarSource{
			SecretKeyRef: &core.SecretKeySelector{
				LocalObjectReference: core.LocalObjectReference{Name: secretName},
				Key:                  key,
				Optional:             &optional,
			},
		},
	}
}

func (r *OAPServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.OAPServer{}).
		Owns(&apps.Deployment{}).
		Owns(&core.Service{}).
		// Credential Secrets are referenced, not owned, so nothing here would otherwise notice one
		// being rotated. Without this the pods keep the value they resolved at start until the
		// periodic requeue happens to re-render them.
		Watches(&core.Secret{}, handler.EnqueueRequestsFromMapFunc(r.oapServersUsingSecret)).
		Complete(r)
}

// oapServersUsingSecret maps a Secret to the OAPServers whose Storage takes credentials from it.
func (r *OAPServerReconciler) oapServersUsingSecret(ctx context.Context, obj client.Object) []reconcile.Request {
	storages := &operatorv1alpha1.StorageList{}
	if err := r.Client.List(ctx, storages, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	named := make(map[string]bool)
	for i := range storages.Items {
		if storages.Items[i].Spec.Security.User.SecretName == obj.GetName() {
			named[storages.Items[i].Name] = true
		}
	}
	if len(named) == 0 {
		return nil
	}

	oapServers := &operatorv1alpha1.OAPServerList{}
	if err := r.Client.List(ctx, oapServers, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	var requests []reconcile.Request
	for i := range oapServers.Items {
		o := &oapServers.Items[i]
		if o.Spec.StorageConfig != nil && named[o.Spec.StorageConfig.Name] {
			requests = append(requests, reconcile.Request{
				NamespacedName: client.ObjectKey{Namespace: o.Namespace, Name: o.Name},
			})
		}
	}
	return requests
}
