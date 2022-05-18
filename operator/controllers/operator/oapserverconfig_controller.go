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
	"reflect"
	"strings"
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

// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=oapserverconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=oapserverconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=oapservers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=oapservers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

func (r *OAPServerConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := runtimelog.FromContext(ctx)
	log.Info("=====================reconcile started================================")

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
			// Update the static configuration
			labelSelector, err := r.UpdateStaticConfig(ctx, log, &oapServer, &oapServerConfig)
			if err != nil {
				log.Error(err, "failed to update the static configuration")
			}

			if oapServerConfig.Spec.DynamicConfig != nil {
				// Update the dynamic configuration
				err = r.UpdateDynamicConfig(ctx, log, &oapServer, &oapServerConfig, labelSelector)
				if err != nil {
					log.Error(err, "failed to update OAPServerConfig's status")
					return ctrl.Result{}, err
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

func (r *OAPServerConfigReconciler) UpdateStaticConfig(ctx context.Context, log logr.Logger,
	oapServer *operatorv1alpha1.OAPServer, oapServerConfig *operatorv1alpha1.OAPServerConfig) (string, error) {
	labelSelector := ""
	deployment := apps.Deployment{}

	changed := false

	oldConfig := make(map[string]string)
	newConfig := make(map[string]string)

	if err := r.Client.Get(ctx, client.ObjectKey{Namespace: oapServer.Namespace, Name: oapServer.Name + "-oap"}, &deployment); err != nil && !apierrors.IsNotFound(err) {
		return labelSelector, fmt.Errorf("failed to get deployment: %w", err)
	}

	for _, e := range deployment.Spec.Template.Spec.Containers[0].Env {
		if e.Name == "SW_CLUSTER_K8S_LABEL" {
			labelSelector = e.Value
		}
	}
	// setup the new Config
	for _, e := range oapServerConfig.Spec.StaticConfig {
		newConfig[e.Name] = e.Value
	}

	// enable the dynamic configuration
	if oapServerConfig.Spec.DynamicConfig != nil {
		newConfig["SW_CONFIGURATION"] = "k8s-configmap"
	}

	configmap := core.ConfigMap{}
	err := r.Client.Get(ctx, client.ObjectKey{Namespace: oapServer.Namespace,
		Name: oapServer.Name + "-" + oapServerConfig.Spec.Version + "-static-config"}, &configmap)
	if err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, "failed to get static configuration configmap")
		return labelSelector, err
	}

	// if the configmap exist
	if !apierrors.IsNotFound(err) {
		oldConfig = configmap.Data
		if reflect.DeepEqual(oldConfig, newConfig) {
			log.Info("Static configuration keeps the same as before")
			return labelSelector, nil
		}
		// delete the old configmap
		if err := r.Client.Delete(ctx, &configmap); err != nil {
			log.Error(err, "faled to delete the static configuration configmap")
		}
	}

	// overlay the deployment's Env
	configExist := make(map[string]bool)
	overlay := []core.EnvVar{}
	for i, e := range deployment.Spec.Template.Spec.Containers[0].Env {
		v, newok := newConfig[e.Name]
		_, oldok := oldConfig[e.Name]
		if e.Name == "SW_CLUSTER_K8S_LABEL" && newok {
			labelSelector = v
		}
		// the value updated in the new config
		if newok {
			configExist[e.Name] = true
			if v != e.Value {
				overlay = append(overlay, core.EnvVar{Name: e.Name, Value: v})
				changed = true
				continue
			}
		}
		// the value deleted in the new config
		if !newok && oldok {
			changed = true
			continue
		}
		overlay = append(overlay, deployment.Spec.Template.Spec.Containers[0].Env[i])
	}

	for k, v := range newConfig {
		if _, ok := configExist[k]; !ok {
			overlay = append(overlay, core.EnvVar{Name: k, Value: v})
			changed = true
		}
	}

	if changed {
		deployment.Spec.Template.Spec.Containers[0].Env = overlay
		if err := r.Client.Update(ctx, &deployment); err != nil {
			return labelSelector, fmt.Errorf("failed to update the deployment of OAPServer: %w", err)
		}
	}

	// create new static configuration configmap
	configmap = core.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      oapServer.Name + "-" + oapServerConfig.Spec.Version + "-static-config",
			Namespace: oapServer.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					APIVersion: oapServerConfig.APIVersion,
					Kind:       oapServerConfig.Kind,
					Name:       oapServerConfig.Name,
					UID:        oapServerConfig.UID,
				},
			},
		},
		Data: newConfig,
	}

	if err := r.Client.Create(ctx, &configmap); err != nil {
		log.Error(err, "failed to create static configuration configmap")
		return labelSelector, err
	}

	log.Info("successfully setup the static configuration")
	return labelSelector, nil
}

func (r *OAPServerConfigReconciler) UpdateDynamicConfig(ctx context.Context, log logr.Logger,
	oapServer *operatorv1alpha1.OAPServer, oapServerConfig *operatorv1alpha1.OAPServerConfig, labelSelector string) error {
	// if the configmap doesn't exist, then create configmap
	configmap := core.ConfigMap{}
	changed := false
	err := r.Client.Get(ctx, client.ObjectKey{Namespace: oapServer.Namespace,
		Name: oapServer.Name + "-" + oapServerConfig.Spec.Version + "-dynamic-config"}, &configmap)
	if err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, "failed to get dynamic configuration configmap")
		return err
	}

	labels := make(map[string]string)
	str := strings.Split(labelSelector, ",")
	for i := range str {
		s := strings.Split(str[i], "=")
		labels[s[0]] = s[1]
	}

	// check the dynamic configuration
	oldConfig := configmap.Data
	newConfig := oapServerConfig.Spec.DynamicConfig
	if !reflect.DeepEqual(oldConfig, newConfig) {
		changed = true
	}

	// check the k8s label
	if !reflect.DeepEqual(labels, configmap.Labels) {
		changed = true
	}

	// if the configmap exist and the dynamic configuration or k8s label changed, then delete it
	if !apierrors.IsNotFound(err) {
		if changed {
			if err := r.Client.Delete(ctx, &configmap); err != nil {
				log.Error(err, "faled to delete the dynamic configuration configmap")
			}
		} else {
			log.Info("Dynamic configuration keeps the same as before")
			return nil
		}
	}
	configmap = core.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      oapServer.Name + "-" + oapServerConfig.Spec.Version + "-dynamic-config",
			Namespace: oapServer.Namespace,
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
		Data: oapServerConfig.Spec.DynamicConfig,
	}

	// create the dynamic configuration configmap
	if err := r.Client.Create(ctx, &configmap); err != nil {
		log.Error(err, "failed to create dynamic configuration configmap")
		return err
	}

	log.Info("successfully setup the dynamic configuration")
	return nil
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
			overlay.ExpectedConfiguredNum += int(oapList.Items[i].Spec.Instances)
			overlay.RealConfiguredNum += int(oapList.Items[i].Status.AvailableReplicas)
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
		errCol.Collect(fmt.Errorf("failed to update status of oapServerConfig: %w", err))
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
			errCol.Collect(fmt.Errorf("failed to update status of oapServerConfig: %w", err))
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
