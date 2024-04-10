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

// EventExporterConfigReconciler reconciles a EventExporterConfig object
type EventExporterConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=eventexporterconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=eventexporterconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=eventexporterconfigs/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=eventexporters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=eventexporters/status,verbs=get;update;patch

func (r *EventExporterConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := runtimelog.FromContext(ctx)
	log.Info("=====================eventexporterconfig reconcile started================================")

	eventExporterConfig := operatorv1alpha1.EventExporterConfig{}
	if err := r.Client.Get(ctx, req.NamespacedName, &eventExporterConfig); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	eventExporterList := operatorv1alpha1.EventExporterList{}
	opts := []client.ListOption{
		client.InNamespace(req.Namespace),
	}

	if err := r.List(ctx, &eventExporterList, opts...); err != nil && !apierrors.IsNotFound(err) {
		return ctrl.Result{}, fmt.Errorf("failed to list eventexporter: %w", err)
	}

	for _, eventExporter := range eventExporterList.Items {
		if eventExporter.Spec.Version != eventExporterConfig.Spec.Version {
			continue
		}
		deployment := apps.Deployment{}
		if err := r.Client.Get(ctx, client.ObjectKey{Namespace: eventExporter.Namespace, Name: eventExporter.Name + "-eventexporter"}, &deployment); err != nil && !apierrors.IsNotFound(err) {
			return ctrl.Result{}, fmt.Errorf("failed to get the deployment of eventexporter: %w", err)
		}

		changed, err := r.OverlayData(ctx, log, &eventExporterConfig, &deployment)
		if err != nil {
			log.Error(err, "failed to overlay the data")
		}

		if !changed {
			continue
		}

		if err := r.Client.Update(ctx, &deployment); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update the deployment of eventexporter: %w", err)
		}
	}

	if err := r.checkState(ctx, log, &eventExporterConfig, eventExporterList); err != nil {
		log.Error(err, "failed to update EventExporterConfig's status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *EventExporterConfigReconciler) OverlayData(ctx context.Context, log logr.Logger,
	config *operatorv1alpha1.EventExporterConfig, deployment *apps.Deployment) (bool, error) {
	newMd5Hash := MD5Hash(config.Spec.Data)

	configmap := core.ConfigMap{}
	err := r.Client.Get(ctx, client.ObjectKey{Namespace: config.Namespace, Name: config.Name}, &configmap)
	if err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, "failed to get the eventexporter's configmap")
		return false, err
	}

	changed := false

	// delete old config map if it exists and is outdated
	if !apierrors.IsNotFound(err) {
		oldMd5Hash := configmap.Labels["md5-data"]
		if oldMd5Hash != newMd5Hash {
			changed = true
			if err := r.Client.Delete(ctx, &configmap); err != nil {
				log.Error(err, "failed to delete the eventexporter's configmap")
			}
		} else {
			log.Info("eventexporter configuration keeps the same as before")
			return false, nil
		}
	}

	configmap = core.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.Name,
			Namespace: config.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: config.APIVersion,
					Kind:       config.Kind,
					Name:       config.Name,
					UID:        config.UID,
				},
			},
			Labels: map[string]string{
				"version":  config.Spec.Version,
				"md5-data": newMd5Hash,
			},
		},
		Data: map[string]string{"config.yaml": config.Spec.Data},
	}
	if err := r.Client.Create(ctx, &configmap); err != nil {
		log.Error(err, "failed to create static configuration configmap")
		return true, err
	}

	// will trigger restart
	deployment.Spec.Template.ObjectMeta.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

	return changed, nil
}

func (r *EventExporterConfigReconciler) checkState(ctx context.Context, log logr.Logger,
	config *operatorv1alpha1.EventExporterConfig, eventExporterList operatorv1alpha1.EventExporterList) error {
	errCol := new(kubernetes.ErrorCollector)

	nilTime := metav1.Time{}
	now := metav1.NewTime(time.Now())
	overlay := operatorv1alpha1.EventExporterConfigStatus{}

	// get Instances and AvailableReplicas
	for _, exporter := range eventExporterList.Items {
		if exporter.Spec.Version != config.Spec.Version {
			continue
		}
		overlay.Desired += int(exporter.Spec.Instances)
		overlay.Ready += int(exporter.Status.AvailableReplicas)
	}

	if config.Status.CreationTime == nilTime {
		overlay.CreationTime = now
		overlay.LastUpdateTime = now
	} else {
		overlay.CreationTime = config.Status.CreationTime
		overlay.LastUpdateTime = now
	}

	config.Status = overlay
	config.Kind = "EventExporterConfig"
	if err := kubernetes.ApplyOverlay(config, &operatorv1alpha1.EventExporterConfig{Status: overlay}); err != nil {
		errCol.Collect(fmt.Errorf("failed to apply overlay: %w", err))
		return errCol.Error()
	}

	if err := r.updateStatus(ctx, config, overlay, errCol); err != nil {
		errCol.Collect(fmt.Errorf("failed to update status of EventExporterConfig: %w", err))
	}

	log.Info("updated EventExporterConfig sub resource")
	return errCol.Error()
}

func (r *EventExporterConfigReconciler) updateStatus(ctx context.Context, config *operatorv1alpha1.EventExporterConfig,
	overlay operatorv1alpha1.EventExporterConfigStatus, errCol *kubernetes.ErrorCollector) error {
	// avoid resource conflict
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		if err := r.Client.Get(ctx, client.ObjectKey{Name: config.Name, Namespace: config.Namespace}, config); err != nil {
			errCol.Collect(fmt.Errorf("failed to get eventExporterConfig: %w", err))
		}
		config.Status = overlay
		config.Kind = "EventExporterConfig"
		if err := kubernetes.ApplyOverlay(config, &operatorv1alpha1.EventExporterConfig{Status: overlay}); err != nil {
			errCol.Collect(fmt.Errorf("failed to apply overlay: %w", err))
		}
		if err := r.Status().Update(ctx, config); err != nil {
			errCol.Collect(fmt.Errorf("failed to update status: %w", err))
		}
		return errCol.Error()
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *EventExporterConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.EventExporterConfig{}).
		Owns(&apps.Deployment{}).
		Complete(r)
}
