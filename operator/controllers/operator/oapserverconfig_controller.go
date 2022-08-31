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
	"sort"
	"time"

	"github.com/go-logr/logr"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	runtimelog "sigs.k8s.io/controller-runtime/pkg/log"

	operatorv1alpha1 "github.com/apache/skywalking-swck/operator/apis/operator/v1alpha1"
	"github.com/apache/skywalking-swck/operator/pkg/kubernetes"
)

// OAPServerConfigReconciler reconciles a OAPServerConfig object
type OAPServerConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

type SortByFileName []operatorv1alpha1.FileConfig

func (a SortByFileName) Len() int {
	return len(a)
}
func (a SortByFileName) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a SortByFileName) Less(i, j int) bool {
	return a[i].Name < a[j].Name
}

type SortByEnvName []core.EnvVar

func (a SortByEnvName) Len() int {
	return len(a)
}
func (a SortByEnvName) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a SortByEnvName) Less(i, j int) bool {
	return a[i].Name < a[j].Name
}

// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=oapserverconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=oapserverconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=oapservers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=oapservers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

func (r *OAPServerConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := runtimelog.FromContext(ctx)
	log.Info("=====================oapserverconfig reconcile started================================")

	oapServerConfig := operatorv1alpha1.OAPServerConfig{}
	if err := r.Client.Get(ctx, req.NamespacedName, &oapServerConfig); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	oapList := operatorv1alpha1.OAPServerList{}
	opts := []client.ListOption{
		client.InNamespace(req.Namespace),
	}

	if err := r.List(ctx, &oapList, opts...); err != nil && !apierrors.IsNotFound(err) {
		return ctrl.Result{}, fmt.Errorf("failed to list oapserver: %w", err)
	}

	// get the specific version's oapserver
	for i := range oapList.Items {
		if oapList.Items[i].Spec.Version == oapServerConfig.Spec.Version {
			oapServer := oapList.Items[i]
			deployment := apps.Deployment{}
			if err := r.Client.Get(ctx, client.ObjectKey{Namespace: oapServer.Namespace, Name: oapServer.Name + "-oap"}, &deployment); err != nil && !apierrors.IsNotFound(err) {
				return ctrl.Result{}, fmt.Errorf("failed to get the deployment of OAPServer: %w", err)
			}
			// overlay the env configuration
			envChanged, err := r.OverlayEnv(ctx, log, &oapServer, &oapServerConfig, &deployment)
			if err != nil {
				log.Error(err, "failed to overlay the env configuration")
			}
			// overlay the file configuration
			fileChanged, err := r.OverlayStaticFile(ctx, log, &oapServerConfig, &deployment)
			if err != nil {
				log.Error(err, "failed to overlay the file configuration")
			}
			// update the deployment
			if envChanged || fileChanged {
				if err := r.Client.Update(ctx, &deployment); err != nil {
					return ctrl.Result{}, fmt.Errorf("failed to update the deployment of OAPServer: %w", err)
				}
			}
		}
	}

	if err := r.checkState(ctx, log, &oapServerConfig, oapList); err != nil {
		log.Error(err, "failed to update OAPServerConfig's status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: schedDuration}, nil
}

func (r *OAPServerConfigReconciler) OverlayEnv(ctx context.Context, log logr.Logger,
	oapServer *operatorv1alpha1.OAPServer, oapServerConfig *operatorv1alpha1.OAPServerConfig, deployment *apps.Deployment) (bool, error) {
	changed := false

	sort.Sort(SortByEnvName(oapServerConfig.Spec.Env))
	newMd5Hash := MD5Hash(oapServerConfig.Spec.Env)

	oldMd5Hash, ok := deployment.Spec.Template.Labels["md5-env"]
	if !ok || oldMd5Hash != newMd5Hash {
		changed = true
	}

	if changed {
		deployment.Spec.Template.Spec.Containers[0].Env = oapServerConfig.Spec.Env
		deployment.Spec.Template.Labels["md5-env"] = newMd5Hash
	} else {
		log.Info("env configuration keeps the same as before")
		return changed, nil
	}

	log.Info("successfully overlay the env configuration")
	return changed, nil
}

func (r *OAPServerConfigReconciler) OverlayStaticFile(ctx context.Context, log logr.Logger,
	oapServerConfig *operatorv1alpha1.OAPServerConfig, deployment *apps.Deployment) (bool, error) {
	changed := false

	sort.Sort(SortByFileName(oapServerConfig.Spec.File))
	newMd5Hash := MD5Hash(oapServerConfig.Spec.File)
	configmap := core.ConfigMap{}
	err := r.Client.Get(ctx, client.ObjectKey{Namespace: oapServerConfig.Namespace,
		Name: oapServerConfig.Name}, &configmap)
	if err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, "failed to get the static file configuration's configmap")
		return changed, err
	}
	// if the configmap exist and the static configuration changed, then delete it
	if !apierrors.IsNotFound(err) {
		oldMd5Hash := configmap.Labels["md5-file"]
		if oldMd5Hash != newMd5Hash {
			changed = true
			if err := r.Client.Delete(ctx, &configmap); err != nil {
				log.Error(err, "faled to delete the static file configuration's configmap")
			}
		} else {
			log.Info("file configuration keeps the same as before")
			return changed, nil
		}
	}

	data := make(map[string]string)
	mounts := []core.VolumeMount{}
	volume := core.Volume{
		Name: oapServerConfig.Name,
		VolumeSource: core.VolumeSource{
			ConfigMap: &core.ConfigMapVolumeSource{
				LocalObjectReference: core.LocalObjectReference{
					Name: oapServerConfig.Name,
				},
			},
		},
	}
	for _, f := range oapServerConfig.Spec.File {
		mounts = append(mounts, core.VolumeMount{
			MountPath: f.Path + "/" + f.Name,
			Name:      oapServerConfig.Name,
			SubPath:   f.Name,
		})
		data[f.Name] = f.Data
	}

	labels := make(map[string]string)
	// set the version label
	labels["version"] = oapServerConfig.Spec.Version
	// set the configuration type
	labels["oapServerConfig"] = "static"
	// set the md5 value of the data
	labels["md5-file"] = newMd5Hash
	// create configmap for static files
	configmap = core.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      oapServerConfig.Name,
			Namespace: oapServerConfig.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					APIVersion: oapServerConfig.APIVersion,
					Kind:       oapServerConfig.Kind,
					Name:       oapServerConfig.Name,
					UID:        oapServerConfig.UID,
				},
			},
			Labels: labels,
		},
		Data: data,
	}

	if err := r.Client.Create(ctx, &configmap); err != nil {
		log.Error(err, "failed to create static configuration configmap")
		return changed, err
	}

	overlayDeployment := deployment
	overlayDeployment.Spec.Template.Spec.Containers[0].VolumeMounts = mounts
	overlayDeployment.Spec.Template.Spec.Volumes = []core.Volume{volume}
	if err := kubernetes.ApplyOverlay(deployment, overlayDeployment); err != nil {
		log.Error(err, "failed to apply overlay deployment")
	}

	log.Info("successfully overlay the file configuration")
	return changed, nil
}

func (r *OAPServerConfigReconciler) checkState(ctx context.Context, log logr.Logger,
	oapServerConfig *operatorv1alpha1.OAPServerConfig, oapList operatorv1alpha1.OAPServerList) error {
	errCol := new(kubernetes.ErrorCollector)

	nilTime := metav1.Time{}
	now := metav1.NewTime(time.Now())
	overlay := operatorv1alpha1.OAPServerConfigStatus{}

	// get Instances and AvailableReplicas
	for i := range oapList.Items {
		if oapList.Items[i].Spec.Version == oapServerConfig.Spec.Version {
			overlay.Desired += int(oapList.Items[i].Spec.Instances)
			overlay.Ready += int(oapList.Items[i].Status.AvailableReplicas)
		}
	}

	if oapServerConfig.Status.CreationTime == nilTime {
		overlay.CreationTime = now
		overlay.LastUpdateTime = now
	} else {
		overlay.CreationTime = oapServerConfig.Status.CreationTime
		overlay.LastUpdateTime = now
	}

	oapServerConfig.Status = overlay
	oapServerConfig.Kind = "OAPServerConfig"
	if err := kubernetes.ApplyOverlay(oapServerConfig, &operatorv1alpha1.OAPServerConfig{Status: overlay}); err != nil {
		errCol.Collect(fmt.Errorf("failed to apply overlay: %w", err))
		return errCol.Error()
	}

	if err := r.updateStatus(ctx, oapServerConfig, overlay, errCol); err != nil {
		errCol.Collect(fmt.Errorf("failed to update status of OAPServerConfig: %w", err))
	}

	log.Info("updated OAPServerConfig sub resource")
	return errCol.Error()
}

func (r *OAPServerConfigReconciler) updateStatus(ctx context.Context, oapServerConfig *operatorv1alpha1.OAPServerConfig,
	overlay operatorv1alpha1.OAPServerConfigStatus, errCol *kubernetes.ErrorCollector) error {
	// avoid resource conflict
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		if err := r.Client.Get(ctx, client.ObjectKey{Name: oapServerConfig.Name, Namespace: oapServerConfig.Namespace}, oapServerConfig); err != nil {
			errCol.Collect(fmt.Errorf("failed to get oapServerConfig: %w", err))
		}
		oapServerConfig.Status = overlay
		oapServerConfig.Kind = "OAPServerConfig"
		if err := kubernetes.ApplyOverlay(oapServerConfig, &operatorv1alpha1.OAPServerConfig{Status: overlay}); err != nil {
			errCol.Collect(fmt.Errorf("failed to apply overlay: %w", err))
		}
		if err := r.Status().Update(ctx, oapServerConfig); err != nil {
			errCol.Collect(fmt.Errorf("failed to update status: %w", err))
		}
		return errCol.Error()
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *OAPServerConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.OAPServerConfig{}).
		Owns(&apps.Deployment{}).
		Complete(r)
}
