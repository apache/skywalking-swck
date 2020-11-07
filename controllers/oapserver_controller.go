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

package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	apiequal "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1alpha1 "github.com/skywalking-swck/api/v1alpha1"
	"github.com/skywalking-swck/pkg/kubernetes"
)

const annotationKeyIstioSetup = "istio-setup-command"

var schedDuration, _ = time.ParseDuration("1m")
var rushModeSchedDuration, _ = time.ParseDuration("5s")

// OAPServerReconciler reconciles a OAPServer object
type OAPServerReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=oapservers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=oapservers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=,resources=services,verbs=get;list;watch;create;update;patch;delete

func (r *OAPServerReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("oapserver", req.NamespacedName)
	log.Info("=====================reconcile started================================")

	oapServer := operatorv1alpha1.OAPServer{}
	if err := r.Client.Get(ctx, req.NamespacedName, &oapServer); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	serviceName := oapServer.Name
	if err := r.service(ctx, log, serviceName, &oapServer); err != nil {
		return ctrl.Result{}, err
	}

	deploymentName := oapServer.Name
	if err := r.deployment(ctx, log, deploymentName, &oapServer); err != nil {
		return ctrl.Result{}, err
	}

	r.istio(ctx, log, serviceName, &oapServer)

	return ctrl.Result{RequeueAfter: r.checkState(ctx, log, &oapServer, serviceName, deploymentName)}, nil
}

func (r *OAPServerReconciler) deployment(ctx context.Context, log logr.Logger, deploymentName string, oapServer *operatorv1alpha1.OAPServer) error {
	currentDeploy := apps.Deployment{}
	err := r.Client.Get(ctx, client.ObjectKey{Namespace: oapServer.Namespace, Name: deploymentName}, &currentDeploy)
	if apierrors.IsNotFound(err) {
		log.Info("could not find existing Deployment, creating one...")

		currentDeploy = *buildDeployment(oapServer, deploymentName)
		if err = r.Client.Create(ctx, &currentDeploy); err != nil {
			return err
		}

		log.Info("created Deployment resource")
		return nil
	}
	if err != nil {
		return err
	}

	deployment := buildDeployment(oapServer, deploymentName)
	if apiequal.Semantic.DeepDerivative(deployment.Spec, currentDeploy.Spec) {
		log.Info("Deployment keeps the same as before")
		return nil
	}
	if err := r.Client.Update(ctx, deployment); err != nil {
		return err
	}
	log.Info("updated Deployment resource")
	return nil
}

func (r *OAPServerReconciler) checkState(ctx context.Context, log logr.Logger, oapServer *operatorv1alpha1.OAPServer, serviceName, deploymentName string) time.Duration {
	overlay := operatorv1alpha1.OAPServerStatus{}
	deployment := apps.Deployment{}
	nextSchedule := schedDuration
	if err := r.Client.Get(ctx, client.ObjectKey{Namespace: oapServer.Namespace, Name: deploymentName}, &deployment); err != nil {
		nextSchedule = rushModeSchedDuration
	} else {
		overlay.Conditions = deployment.Status.Conditions
		overlay.AvailableReplicas = deployment.Status.AvailableReplicas
		if oapServer.Spec.Instances != overlay.AvailableReplicas {
			nextSchedule = rushModeSchedDuration
		}
		if oapServer.Spec.Image == "" {
			oapServer.Spec.Image = deployment.Spec.Template.Spec.Containers[0].Image
			if err := r.Update(ctx, oapServer); err != nil {
				log.Error(err, "failed to update OAPServer Image field")
			}
			log.Info("updated OAPServer Image field")
			return rushModeSchedDuration
		}
	}
	service := core.Service{}
	if err := r.Client.Get(ctx, client.ObjectKey{Namespace: oapServer.Namespace, Name: serviceName}, &service); err != nil {
		nextSchedule = rushModeSchedDuration
	} else {
		overlay.Address = fmt.Sprintf("%s.%s", service.Name, service.Namespace)
	}
	if apiequal.Semantic.DeepDerivative(overlay, oapServer.Status) {
		log.Info("Status keeps the same as before")
		return nextSchedule
	}
	oapServer.Status = overlay
	if err := kubernetes.ApplyOverlay(oapServer, &operatorv1alpha1.OAPServer{Status: overlay}); err != nil {
		log.Error(err, "failed to overlay OAPServer")
		return rushModeSchedDuration
	}
	if err := r.Status().Update(ctx, oapServer); err != nil {
		return rushModeSchedDuration
	}
	log.Info("updated Status sub resource")
	return nextSchedule
}

func (r *OAPServerReconciler) istio(ctx context.Context, log logr.Logger, serviceName string, oapServer *operatorv1alpha1.OAPServer) {
	for _, envVar := range oapServer.Spec.Config {
		if envVar.Name == "SW_ENVOY_METRIC_ALS_HTTP_ANALYSIS" &&
			oapServer.ObjectMeta.Annotations[annotationKeyIstioSetup] == "" {
			oapServer.Annotations[annotationKeyIstioSetup] = fmt.Sprintf("istioctl install --set profile=demo "+
				"--set meshConfig.defaultConfig.envoyAccessLogService.address=%s.%s:11800 "+
				"--set meshConfig.enableEnvoyAccessLogService=true", serviceName, oapServer.Namespace)
			if err := r.Update(ctx, oapServer); err != nil {
				log.Error(err, "unable to patch Istio setup command to annotation")
				return
			}
			log.Info("patched Istio annotation")
			return
		}
	}
}

func buildDeployment(oapServer *operatorv1alpha1.OAPServer, deploymentName string) *apps.Deployment {
	podSpec := &core.PodTemplateSpec{}

	if &podSpec.ObjectMeta == nil {
		podSpec.ObjectMeta = metav1.ObjectMeta{Labels: make(map[string]string)}
	}
	if podSpec.Labels == nil {
		podSpec.Labels = make(map[string]string)
	}
	podSpec.ObjectMeta.Labels = labelSelector(oapServer)

	if &podSpec.Spec == nil {
		podSpec.Spec = core.PodSpec{}
	}
	pod := &podSpec.Spec
	image := oapServer.Spec.Image
	if image == "" {
		v := oapServer.Spec.Version
		vTmpl := "apache/skywalking-oap-server:%s-%s"
		vES := "es6"
		for _, e := range oapServer.Spec.Config {
			if e.Name != "SW_STORAGE" {
				continue
			}
			if e.Value == "elasticsearch7" {
				vES = "es7"
			}
		}
		image = fmt.Sprintf(vTmpl, v, vES)
	}
	probe := &core.Probe{
		Handler: core.Handler{
			Exec: &core.ExecAction{
				Command: []string{"/skywalking/bin/swctl", "ch"},
			},
		},
		InitialDelaySeconds: 40,
		TimeoutSeconds:      10,
		PeriodSeconds:       30,
		SuccessThreshold:    1,
		FailureThreshold:    3,
	}
	cc := oapServer.Spec.Config
	if cc == nil {
		cc = []core.EnvVar{}
	}
	cc = append(cc, core.EnvVar{Name: "SW_TELEMETRY", Value: "prometheus"})
	cc = append(cc, core.EnvVar{Name: "SW_HEALTH_CHECKER", Value: "default"})
	pod.Containers = []core.Container{
		{
			Name:  "oap",
			Image: image,
			Env:   cc,
			Ports: []core.ContainerPort{
				{
					Name:          "grpc",
					ContainerPort: 11800,
					Protocol:      core.ProtocolTCP,
				},
				{
					Name:          "graphql",
					ContainerPort: 12800,
					Protocol:      core.ProtocolTCP,
				},
				{
					Name:          "http-monitoring",
					ContainerPort: 1234,
					Protocol:      core.ProtocolTCP,
				},
			},
			LivenessProbe:  probe,
			ReadinessProbe: probe,
		},
	}

	deployment := apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:            deploymentName,
			Namespace:       oapServer.Namespace,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(oapServer, operatorv1alpha1.GroupVersion.WithKind("OAPServer"))},
		},
		Spec: apps.DeploymentSpec{
			Replicas: &oapServer.Spec.Instances,
			Selector: &metav1.LabelSelector{
				MatchLabels: labelSelector(oapServer),
			},
			MinReadySeconds: 5,
			Template:        *podSpec,
		},
	}
	return &deployment
}

func (r *OAPServerReconciler) service(ctx context.Context, log logr.Logger, serviceName string, oapServer *operatorv1alpha1.OAPServer) error {
	currentService := core.Service{}
	err := r.Client.Get(ctx, client.ObjectKey{Namespace: oapServer.Namespace, Name: serviceName}, &currentService)
	if apierrors.IsNotFound(err) {
		log.Info("could not find existing Service, creating one...")
		currentService = buildService(serviceName, nil, oapServer)
		if err = r.Client.Create(ctx, &currentService); err != nil {
			return err
		}
		return nil
	}
	if err != nil {
		return err
	}

	service := buildService(serviceName, currentService.DeepCopy(), oapServer)

	if apiequal.Semantic.DeepDerivative(service.Spec, currentService.Spec) {
		log.Info("Service keeps the same as before")
		return nil
	}
	if err := r.Client.Update(ctx, &service); err != nil {
		return err
	}
	log.Info("updated Service resource")
	return nil
}

func buildService(serviceName string, base *core.Service, oapServer *operatorv1alpha1.OAPServer) core.Service {
	s := core.Service{}
	s.Name = serviceName
	s.Namespace = oapServer.Namespace
	s.ObjectMeta = metav1.ObjectMeta{
		Name:            serviceName,
		Namespace:       oapServer.Namespace,
		OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(oapServer, operatorv1alpha1.GroupVersion.WithKind("OAPServer"))},
	}
	s.Spec = core.ServiceSpec{
		Type:     core.ServiceTypeClusterIP,
		Selector: labelSelector(oapServer),
		Ports: []core.ServicePort{
			{
				Name:     "grpc",
				Port:     11800,
				Protocol: core.ProtocolTCP,
				TargetPort: intstr.IntOrString{
					Type:   intstr.String,
					StrVal: "grpc",
				},
			},
			{
				Name:     "graphql",
				Port:     12800,
				Protocol: core.ProtocolTCP,
				TargetPort: intstr.IntOrString{
					Type:   intstr.String,
					StrVal: "graphql",
				},
			},
		},
	}
	if base == nil {
		return s
	}
	if err := kubernetes.ApplyOverlay(base, &s); err != nil {
		return s
	}
	return *base
}

func labelSelector(oapServer *operatorv1alpha1.OAPServer) map[string]string {
	return map[string]string{fmt.Sprintf("%s/oap-server-name", operatorv1alpha1.GroupVersion.Group): oapServer.Name}
}

func (r *OAPServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.OAPServer{}).
		Complete(r)
}
