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
	"text/template"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

var useImmutableConfigMap = true

// EventExporterReconciler reconciles a EventExporter object
type EventExporterReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	FileRepo kubernetes.Repo
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=eventexporters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=eventexporters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=eventexporters/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *EventExporterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := runtimelog.FromContext(ctx)
	log.Info(fmt.Sprintf("===============eventexporter reconcile started (ns: %s, name: %s)===============", req.Namespace, req.Name))

	eventExporter := operatorv1alpha1.EventExporter{}
	if err := r.Client.Get(ctx, req.NamespacedName, &eventExporter); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	newConfigMapName := configMapName(&eventExporter)
	if _, err := r.overlayData(ctx, log, &eventExporter, newConfigMapName); err != nil {
		log.Error(err, "failed to delete eventexporter's configMap")
		return ctrl.Result{}, err
	}

	ff, err := r.FileRepo.GetFilesRecursive("templates")
	if err != nil {
		log.Error(err, "failed to load resource templates")
		return ctrl.Result{}, err
	}

	app := kubernetes.Application{
		Client:   r.Client,
		FileRepo: r.FileRepo,
		CR:       &eventExporter,
		GVK:      operatorv1alpha1.GroupVersion.WithKind("EventExporter"),
		Recorder: r.Recorder,
		TmplFunc: template.FuncMap{
			"configMapName": func() string { return newConfigMapName },
		},
	}

	if err := app.ApplyAll(ctx, ff, log); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.checkState(ctx, log, &eventExporter); err != nil {
		l.Error(err, "failed to check sub resources state")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: schedDuration}, nil

}

func (r *EventExporterReconciler) overlayData(ctx context.Context, log logr.Logger,
	eventExporter *operatorv1alpha1.EventExporter, newConfigMapName string) (changed bool, err error) {

	oldConfigMapName := eventExporter.Status.ConfigMapName
	if oldConfigMapName == newConfigMapName {
		log.Info("eventexporter configuration keeps the same as before")
		return false, nil
	}

	configmap := core.ConfigMap{}
	err = r.Client.Get(ctx, client.ObjectKey{Namespace: eventExporter.Namespace, Name: oldConfigMapName}, &configmap)
	if err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, "failed to get the eventexporter's configmap")
		return true, err
	}

	if !apierrors.IsNotFound(err) {
		if err = r.Client.Delete(ctx, &configmap); err != nil {
			log.Error(err, "failed to delete eventexporter's configmap")
			return true, err
		}
	}

	configmap = core.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: eventExporter.Namespace,
			Name:      newConfigMapName,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: eventExporter.APIVersion,
					Kind:       eventExporter.Kind,
					Name:       eventExporter.Name,
					UID:        eventExporter.UID,
				},
			},
			Labels: map[string]string{
				"app": "eventexporter",
				"operator.skywalking.apache.org/eventexporter-name": eventExporter.Name,
				"operator.skywalking.apache.org/application":        "eventexporter",
				"operator.skywalking.apache.org/component":          "configmap",
			},
		},
		Immutable: &useImmutableConfigMap,
		Data:      map[string]string{"config.yaml": eventExporter.Spec.Config},
	}

	if err = r.Client.Create(ctx, &configmap); err != nil {
		log.Error(err, "failed to create eventexporter's configmap")
		return true, err
	}

	return true, nil
}

func (r *EventExporterReconciler) checkState(ctx context.Context, log logr.Logger, eventExporter *operatorv1alpha1.EventExporter) error {
	overlay := operatorv1alpha1.EventExporterStatus{}
	deployment := apps.Deployment{}
	errCol := new(kubernetes.ErrorCollector)

	if err := r.Client.Get(ctx, client.ObjectKey{Namespace: eventExporter.Namespace,
		Name: eventExporter.Name + "-eventexporter"}, &deployment); err != nil && !apierrors.IsNotFound(err) {
		errCol.Collect(fmt.Errorf("failed to get deployment: %w", err))
	} else {
		overlay.Conditions = deployment.Status.Conditions
		overlay.AvailableReplicas = deployment.Status.AvailableReplicas
		overlay.ConfigMapName = configMapName(eventExporter)
	}

	if apiequal.Semantic.DeepDerivative(overlay, eventExporter.Status) {
		log.Info("Status keeps the same as before")
		return errCol.Error()
	}

	if err := r.updateStatus(ctx, eventExporter, overlay, errCol); err != nil {
		errCol.Collect(fmt.Errorf("failed to update status of EventExporter: %w", err))
	}

	log.Info("updated Status sub resource")

	return errCol.Error()
}

func (r *EventExporterReconciler) updateStatus(ctx context.Context, eventExporter *operatorv1alpha1.EventExporter,
	overlay operatorv1alpha1.EventExporterStatus, errCol *kubernetes.ErrorCollector) error {
	// avoid resource conflict
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		if err := r.Client.Get(ctx, client.ObjectKey{Name: eventExporter.Name, Namespace: eventExporter.Namespace}, eventExporter); err != nil {
			errCol.Collect(fmt.Errorf("failed to get EventExporter: %w", err))
		}

		eventExporter.Status = overlay
		eventExporter.Kind = "EventExporter"
		if err := kubernetes.ApplyOverlay(eventExporter, &operatorv1alpha1.EventExporter{Status: overlay}); err != nil {
			errCol.Collect(fmt.Errorf("failed to apply overlay: %w", err))
		}

		if err := r.Status().Update(ctx, eventExporter); err != nil {
			errCol.Collect(fmt.Errorf("failed to update status of EventExporter: %w", err))
		}
		return errCol.Error()
	})
}

func configMapName(eventExporter *operatorv1alpha1.EventExporter) string {
	return fmt.Sprintf("%s-eventexporter-cm-%s", eventExporter.Name, MD5Hash(eventExporter.Spec.Config))
}

// SetupWithManager sets up the controller with the Manager.
func (r *EventExporterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.EventExporter{}).
		Complete(r)
}
