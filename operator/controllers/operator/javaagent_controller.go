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
	"strings"
	"time"

	"github.com/go-logr/logr"
	core "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	runtimelog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	operatorv1alpha1 "github.com/apache/skywalking-swck/operator/apis/operator/v1alpha1"
	"github.com/apache/skywalking-swck/operator/pkg/kubernetes"
	"github.com/apache/skywalking-swck/operator/pkg/operator/injector"
)

// JavaAgentReconciler reconciles a JavaAgent object
type JavaAgentReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	FileRepo kubernetes.Repo
}

// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=javaagents,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=javaagents/status,verbs=get;update;patch;delete
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch

func (r *JavaAgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := runtimelog.FromContext(ctx)
	log.Info("=====================javaagent reconcile started================================")

	pod := &core.Pod{}
	err := r.Client.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: req.Name}, pod)
	if err != nil && !apierrors.IsNotFound(err) {
		log.Error(err, "failed to get pod")
		return ctrl.Result{}, err
	}

	configmap := &core.ConfigMap{}
	configmapName := ""
	for i := range pod.Spec.Volumes {
		if pod.Spec.Volumes[i].ConfigMap != nil {
			configmapName = pod.Spec.Volumes[i].ConfigMap.Name
		}
	}

	// get pods' OwnerReferences
	if len(pod.OwnerReferences) == 0 {
		log.Error(err, "the pod isn't created by workloads")
		return ctrl.Result{}, err
	}
	ownerReference := pod.OwnerReferences[0]

	// get configmap from the volume of configmap
	if len(configmapName) > 0 {
		err = r.Client.Get(ctx, client.ObjectKey{Namespace: req.Namespace, Name: configmapName}, configmap)
		if err != nil && !apierrors.IsNotFound(err) {
			log.Error(err, "failed to get configmap")
			return ctrl.Result{}, err
		}
	} else {
		log.Info("No configmap mounted.")
	}

	// get configuration from configmap
	config, err := injector.GetConfigmapConfiguration(configmap)
	if err != nil {
		log.Error(err, "failed to get configmap's configuration")
		return ctrl.Result{}, err
	}
	injector.GetInjectedAgentConfig(&pod.Annotations, &config)

	swAgentList := &operatorv1alpha1.SwAgentList{}
	var lastMatchedSwAgent *operatorv1alpha1.SwAgent
	if lastMatchedSwAgent, err = r.getSwAgent(ctx, req, swAgentList, pod); err != nil {
		log.Error(err, "get SwAgent error")
		return ctrl.Result{}, err
	}
	r.injectConfigBySwAgent(lastMatchedSwAgent, config)

	// only get the first selector label from labels as podselector
	labels := pod.Labels
	keys := []string{}
	for k := range labels {
		if !strings.Contains(k, injector.ActiveInjectorLabel) {
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		log.Error(err, "the pod doesn't contain the pod selector")
		return ctrl.Result{}, err
	}
	sort.Strings(keys)
	selectorname := strings.Join([]string{keys[0], labels[keys[0]]}, "-")
	podselector := strings.Join([]string{keys[0], labels[keys[0]]}, "=")

	app := kubernetes.Application{
		Client:   r.Client,
		FileRepo: r.FileRepo,
		CR:       pod,
		GVK:      core.SchemeGroupVersion.WithKind("Pod"),
		TmplFunc: map[string]interface{}{
			"config": func() map[string]string {
				return config
			},
			"ownerReference": func() metav1.OwnerReference {
				return ownerReference
			},
			"SelectorName": func() string {
				return selectorname
			},
			"Namespace": func() string {
				return req.Namespace
			},
			"PodSelector": func() string {
				return podselector
			},
			"ServiceName": func() string {
				return operatorv1alpha1.GetServiceName(&config)
			},
			"BackendService": func() string {
				return operatorv1alpha1.GetBackendService(&config)
			},
		},
	}

	// false means not to compose , such as ownerReferences , as we compose it as template
	_, err = app.Apply(ctx, "injector/templates/javaagent.yaml", log, false)
	if err != nil {
		log.Error(err, "failed to apply javaagent")
		return ctrl.Result{}, err
	}

	if err := r.UpdateStatus(ctx, log, req.Namespace, selectorname, podselector); err != nil {
		log.Error(err, "failed to update javaagent's status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *JavaAgentReconciler) injectConfigBySwAgent(lastMatchedSwAgent *operatorv1alpha1.SwAgent, config map[string]string) {
	if nil != lastMatchedSwAgent && len(lastMatchedSwAgent.Spec.JavaSidecar.Env) > 0 {
		for _, env := range lastMatchedSwAgent.Spec.JavaSidecar.Env {
			if strings.EqualFold(env.Name, "SW_AGENT_COLLECTOR_BACKEND_SERVICES") {
				config[operatorv1alpha1.BackendService] = env.Value
			} else if strings.EqualFold(env.Name, "SW_AGENT_NAME") {
				config[operatorv1alpha1.ServiceName] = env.Value
			}
		}
	}
}

func (r *JavaAgentReconciler) getSwAgent(ctx context.Context, req ctrl.Request,
	swAgentList *operatorv1alpha1.SwAgentList, pod *core.Pod) (*operatorv1alpha1.SwAgent, error) {
	if err := r.Client.List(ctx, swAgentList, client.InNamespace(req.Namespace)); err != nil {
		return nil, err
	}

	// selector
	var lastMatchedSwAgent *operatorv1alpha1.SwAgent
	for _, swAgent := range swAgentList.Items {
		isMatch := true
		if len(swAgent.Spec.Selector) != 0 {
			for k, v := range swAgent.Spec.Selector {
				if !strings.EqualFold(v, pod.Labels[k]) {
					isMatch = false
				}
			}
		}
		if isMatch {
			lastMatchedSwAgent = &swAgent
		}
	}
	return lastMatchedSwAgent, nil
}

func (r *JavaAgentReconciler) UpdateStatus(ctx context.Context, log logr.Logger, namespace, selectorname, podselector string) error {
	errCol := new(kubernetes.ErrorCollector)

	// get javaagent by selectorname
	javaagent := &operatorv1alpha1.JavaAgent{}
	err := r.Client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: selectorname + "-javaagent"}, javaagent)
	if err != nil && !apierrors.IsNotFound(err) {
		errCol.Collect(fmt.Errorf("failed to get javaagent: %w", err))
	}

	// avoid printing error info when the javaagent is creating
	if javaagent.Name == "" {
		log.Info("javaagent is creating...", "name", selectorname+"-javaagent")
		return errCol.Error()
	}

	// return all pods in the request namespace with the podselector
	podList := &core.PodList{}
	label := strings.Split(podselector, "=")
	opts := []client.ListOption{
		client.InNamespace(namespace),
		client.MatchingLabels{label[0]: label[1]},
	}

	if err := r.Client.List(ctx, podList, opts...); err != nil && !apierrors.IsNotFound(err) {
		errCol.Collect(fmt.Errorf("failed to list pod: %w", err))
	}

	// get the pod's number that need to be injected
	expectedInjectedNum := 0
	// get the pod's number that injected successfully
	realInjectedNum := 0
	for i := range podList.Items {
		labels := podList.Items[i].Labels
		annotations := podList.Items[i].Annotations
		if labels != nil && strings.EqualFold(strings.ToLower(labels[injector.ActiveInjectorLabel]), "true") {
			expectedInjectedNum++
		}
		if annotations != nil && strings.EqualFold(strings.ToLower(annotations[injector.SidecarInjectSucceedAnno]), "true") {
			realInjectedNum++
		}
	}

	javaagent.Status.ExpectedInjectedNum = expectedInjectedNum
	javaagent.Status.RealInjectedNum = realInjectedNum

	nilTime := metav1.Time{}
	now := metav1.NewTime(time.Now())
	if javaagent.Status.CreationTime == nilTime {
		javaagent.Status.CreationTime = now
		javaagent.Status.LastUpdateTime = now
	} else {
		javaagent.Status.LastUpdateTime = now
	}

	overlay := javaagent.Status
	if err := r.updateStatus(ctx, javaagent, overlay, errCol); err != nil {
		errCol.Collect(fmt.Errorf("failed to update java status: %w", err))
	}

	log.Info("updated javaagent's status")
	return errCol.Error()
}

func (r *JavaAgentReconciler) updateStatus(ctx context.Context, javaagent *operatorv1alpha1.JavaAgent,
	overlay operatorv1alpha1.JavaAgentStatus, errCol *kubernetes.ErrorCollector) error {
	// avoid resource conflict
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		if err := r.Client.Get(ctx, client.ObjectKey{Name: javaagent.Name, Namespace: javaagent.Namespace}, javaagent); err != nil {
			errCol.Collect(fmt.Errorf("failed to get javaagent: %w", err))
		}
		javaagent.Status = overlay
		javaagent.Kind = "JavaAgent"
		if err := kubernetes.ApplyOverlay(javaagent, &operatorv1alpha1.JavaAgent{Status: overlay}); err != nil {
			errCol.Collect(fmt.Errorf("failed to apply overlay: %w", err))
		}
		if err := r.Status().Update(ctx, javaagent); err != nil {
			errCol.Collect(fmt.Errorf("failed to update status of JavaAgent: %w", err))
		}
		return errCol.Error()
	})
}

func (r *JavaAgentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&core.Pod{}).
		WithEventFilter(predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool {
				annotations := e.Object.GetAnnotations()
				if annotations != nil && strings.ToLower(annotations[injector.SidecarInjectSucceedAnno]) == "true" {
					return true
				}
				return false
			},
			// avoid calling Reconcile when the pod's workload is deleted
			UpdateFunc: func(e event.UpdateEvent) bool {
				annotations := e.ObjectNew.GetAnnotations()
				if annotations != nil && strings.ToLower(annotations[injector.SidecarInjectSucceedAnno]) == "true" {
					return e.ObjectNew.GetDeletionTimestamp() == nil
				}
				return false
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				return false
			},
		}).
		Owns(&operatorv1alpha1.JavaAgent{}).
		Complete(r)
}
