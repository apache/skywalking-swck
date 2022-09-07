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
	"crypto/md5"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/go-logr/logr"
	core "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	runtimelog "sigs.k8s.io/controller-runtime/pkg/log"

	operatorv1alpha1 "github.com/apache/skywalking-swck/operator/apis/operator/v1alpha1"
	"github.com/apache/skywalking-swck/operator/pkg/kubernetes"
)

// OAPServerDynamicConfigReconciler reconciles a OAPServerDynamicConfig object
type OAPServerDynamicConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

type SortByConfigName []operatorv1alpha1.Config

func (a SortByConfigName) Len() int {
	return len(a)
}
func (a SortByConfigName) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a SortByConfigName) Less(i, j int) bool {
	return a[i].Name < a[j].Name
}

// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=oapserverdynamicconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=oapserverdynamicconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=oapservers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=oapservers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

func (r *OAPServerDynamicConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	log := runtimelog.FromContext(ctx)
	log.Info("=====================oapserverdynamicconfig reconcile started================================")

	config := operatorv1alpha1.OAPServerDynamicConfig{}
	if err := r.Client.Get(ctx, req.NamespacedName, &config); err != nil {
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
		if oapList.Items[i].Spec.Version == config.Spec.Version {
			oapServer := oapList.Items[i]
			// Update the dynamic configuration
			err := r.UpdateDynamicConfig(ctx, log, &oapServer, &config)
			if err != nil {
				log.Error(err, "failed to update the dynamic configuration")
				return ctrl.Result{}, err
			}

		}
	}

	if err := r.checkState(ctx, log, &config); err != nil {
		log.Error(err, "failed to update OAPServerDynamicConfig's status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: schedDuration}, nil
}

func (r *OAPServerDynamicConfigReconciler) UpdateDynamicConfig(ctx context.Context, log logr.Logger,
	oapServer *operatorv1alpha1.OAPServer, config *operatorv1alpha1.OAPServerDynamicConfig) error {
	changed := false
	exsitedConfiguration := map[string]bool{}

	dynamicConfigList := operatorv1alpha1.OAPServerDynamicConfigList{}
	opts := []client.ListOption{
		client.InNamespace(config.Namespace),
	}
	if err := r.List(ctx, &dynamicConfigList, opts...); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to list OAPServerDynamicConfig: %w", err)
	}
	for _, i := range dynamicConfigList.Items {
		if i.Name != config.Name {
			for _, n := range i.Spec.Data {
				exsitedConfiguration[n.Name] = true
			}
		}
	}

	configuration := config.Spec.Data
	for i := range configuration {
		_, ok := exsitedConfiguration[configuration[i].Name]
		if ok {
			return fmt.Errorf("the configuration %s already exist", configuration[i].Name)
		}
	}
	sort.Sort(SortByConfigName(configuration))
	newMd5Hash := MD5Hash(configuration)

	labelSelector, err := labels.Parse(config.Spec.LabelSelector)
	if err != nil {
		log.Error(err, "failed to parse string to labelselector")
		return err
	}
	label, err := labels.ConvertSelectorToLabelsMap(config.Spec.LabelSelector)
	if err != nil {
		log.Error(err, "failed to convert labelselector to map")
		return err
	}

	configmap := core.ConfigMap{}
	err = r.Client.Get(ctx, client.ObjectKey{Namespace: config.Namespace,
		Name: config.Name}, &configmap)
	if err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, "failed to get the dynamic configuration configmap")
		return err
	}
	// if the configmap exist and the dynamic configuration or labelselector changed, then delete it
	if !apierrors.IsNotFound(err) {
		oldMd5Hash := configmap.Labels["md5"]
		// check dynamic configuration
		if oldMd5Hash != newMd5Hash {
			changed = true
		}

		// check labelselector
		if !labelSelector.Matches(labels.Set(configmap.Labels)) {
			changed = true
		}
		if changed {
			if err := r.Client.Delete(ctx, &configmap); err != nil {
				log.Error(err, "faled to delete the dynamic configuration configmap")
			}
		} else {
			log.Info("dynamic configuration keeps the same as before")
			return nil
		}
	}

	// set the version label
	label["version"] = config.Spec.Version
	// set the configuration type
	label["OAPServerConfig"] = "dynamic"
	// set the md5 value
	label["md5"] = newMd5Hash
	// set the data
	data := map[string]string{}
	for _, v := range configuration {
		data[v.Name] = v.Value
	}
	configmap = core.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.Name,
			Namespace: oapServer.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				metav1.OwnerReference{
					APIVersion: config.APIVersion,
					Kind:       config.Kind,
					Name:       config.Name,
					UID:        config.UID,
				},
			},
			Labels: label,
		},
		Data: data,
	}

	// create the dynamic configuration configmap
	if err := r.Client.Create(ctx, &configmap); err != nil {
		log.Error(err, "failed to create dynamic configuration configmap")
		return err
	}

	log.Info("successfully setup the dynamic configuration")
	return nil
}

func (r *OAPServerDynamicConfigReconciler) checkState(ctx context.Context, log logr.Logger,
	config *operatorv1alpha1.OAPServerDynamicConfig) error {
	errCol := new(kubernetes.ErrorCollector)

	nilTime := metav1.Time{}
	now := metav1.NewTime(time.Now())
	overlay := operatorv1alpha1.OAPServerDynamicConfigStatus{
		State: "Stopped",
	}

	// get dynamic configuration's state
	configmapList := core.ConfigMapList{}
	opts := []client.ListOption{
		client.InNamespace(config.Namespace),
		client.MatchingLabels{
			"version":         config.Spec.Version,
			"OAPServerConfig": "dynamic",
		},
	}

	if err := r.List(ctx, &configmapList, opts...); err != nil && !apierrors.IsNotFound(err) {
		errCol.Collect(fmt.Errorf("failed to list configmap: %w", err))
	}

	for i := range configmapList.Items {
		if strings.Contains(configmapList.Items[i].Name, config.Name) {
			overlay.State = "Running"
		}
	}

	if config.Status.CreationTime == nilTime {
		overlay.CreationTime = now
		overlay.LastUpdateTime = now
	} else {
		overlay.CreationTime = config.Status.CreationTime
		overlay.LastUpdateTime = now
	}

	config.Status = overlay
	config.Kind = "OAPServerDynamicConfig"
	if err := kubernetes.ApplyOverlay(config, &operatorv1alpha1.OAPServerDynamicConfig{Status: overlay}); err != nil {
		errCol.Collect(fmt.Errorf("failed to apply overlay: %w", err))
		return errCol.Error()
	}

	if err := r.updateStatus(ctx, config, overlay, errCol); err != nil {
		errCol.Collect(fmt.Errorf("failed to update status of OAPServerDynamicConfig: %w", err))
	}

	log.Info("updated OAPServerDynamicConfig sub resource")
	return errCol.Error()
}

func (r *OAPServerDynamicConfigReconciler) updateStatus(ctx context.Context, config *operatorv1alpha1.OAPServerDynamicConfig,
	overlay operatorv1alpha1.OAPServerDynamicConfigStatus, errCol *kubernetes.ErrorCollector) error {
	// avoid resource conflict
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		if err := r.Client.Get(ctx, client.ObjectKey{Name: config.Name, Namespace: config.Namespace}, config); err != nil {
			errCol.Collect(fmt.Errorf("failed to get OAPServerDynamicConfig: %w", err))
		}
		config.Status = overlay
		config.Kind = "OAPServerDynamicConfig"
		if err := kubernetes.ApplyOverlay(config, &operatorv1alpha1.OAPServerDynamicConfig{Status: overlay}); err != nil {
			errCol.Collect(fmt.Errorf("failed to apply overlay: %w", err))
		}
		if err := r.Status().Update(ctx, config); err != nil {
			errCol.Collect(fmt.Errorf("failed to update status: %w", err))
		}
		return errCol.Error()
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *OAPServerDynamicConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.OAPServerDynamicConfig{}).
		Complete(r)
}

func MD5Hash(a interface{}) string {
	h := md5.New()
	h.Write([]byte(fmt.Sprintf("%v", a)))

	return fmt.Sprintf("%x", h.Sum(nil))
}
