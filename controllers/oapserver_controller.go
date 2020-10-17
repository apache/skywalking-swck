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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1alpha1 "github.com/skywalking-swck/api/v1alpha1"
)

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

func (r *OAPServerReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("oapserver", req.NamespacedName)

	log.Info("Fetching OAPServer")
	oapServer := operatorv1alpha1.OAPServer{}
	if err := r.Client.Get(ctx, req.NamespacedName, &oapServer); err != nil {
		log.Error(err, "failed to get OAPServer")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("Checking if an existing Deployment exists")
	deploymentName := "oap-server-deployment"
	deployment := apps.Deployment{}
	err := r.Client.Get(ctx, client.ObjectKey{Namespace: oapServer.Namespace, Name: deploymentName}, &deployment)
	if apierrors.IsNotFound(err) {
		log.Info("Could not find existing Deployment for OAP server, creating one...")

		deployment = *buildDeployment(&oapServer, deploymentName)
		if err := r.Client.Create(ctx, &deployment); err != nil {
			log.Error(err, "failed to create Deployment resource")
			return ctrl.Result{}, err
		}

		oapServer.Status = operatorv1alpha1.OAPServerStatus{
			Phase:    operatorv1alpha1.OAPServerChangingPhase,
			Message:  "Deploying OAP deployment",
			LastTime: metav1.Time{Time: time.Now()},
		}
		if err := r.Status().Update(ctx, &oapServer); err != nil {
			r.Log.Error(err, "unable to update OAPServerOperator status")
		}

		log.Info("created Deployment resource for OAPServer")
		return ctrl.Result{RequeueAfter: rushModeSchedDuration}, nil
	}
	if err != nil {
		log.Error(err, "failed to get Deployment for OAPServer")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: r.checkState(ctx, &oapServer, &deployment)}, nil
}

func (r *OAPServerReconciler) checkState(ctx context.Context, oapServer *operatorv1alpha1.OAPServer, deployment *apps.Deployment) time.Duration {
	r.Log.Info("checking OAP deployment status")
	status := deployment.Status
	lc := status.Conditions[len(status.Conditions)-1]
	p := operatorv1alpha1.OAPServerResourceInvalid
	if lc.Status == core.ConditionTrue {
		switch lc.Type {
		case apps.DeploymentProgressing:
			p = operatorv1alpha1.OAPServerReadyPhase
		case apps.DeploymentAvailable:
			p = operatorv1alpha1.OAPServerReadyPhase
		}
	}
	oapServer.Status = operatorv1alpha1.OAPServerStatus{
		Phase:             p,
		Message:           lc.Message,
		Reason:            lc.Reason,
		LastTime:          metav1.Time{Time: time.Now()},
		AvailableReplicas: status.AvailableReplicas,
	}
	if err := r.Status().Update(ctx, oapServer); err != nil {
		r.Log.Error(err, "unable to update OAPServerOperator status")
		return rushModeSchedDuration
	}

	if p != operatorv1alpha1.OAPServerReadyPhase {
		return rushModeSchedDuration
	}
	return schedDuration
}

func buildDeployment(oapServer *operatorv1alpha1.OAPServer, deploymentName string) *apps.Deployment {

	podSpec := &oapServer.Spec.PodTemplate
	if podSpec != nil {
		podSpec = podSpec.DeepCopy()
	} else {
		podSpec = &core.PodTemplateSpec{}
	}

	if &podSpec.ObjectMeta == nil {
		podSpec.ObjectMeta = metav1.ObjectMeta{Labels: make(map[string]string)}
	}
	if podSpec.Labels == nil {
		podSpec.Labels = make(map[string]string)
	}
	podSpec.ObjectMeta.Labels[fmt.Sprintf("%s/deployment-name", operatorv1alpha1.GroupVersion.Group)] = deploymentName

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
				MatchLabels: map[string]string{
					fmt.Sprintf("%s/deployment-name", operatorv1alpha1.GroupVersion.Group): deploymentName,
				},
			},
			MinReadySeconds: 5,
			Template:        *podSpec,
		},
	}
	return &deployment
}

func (r *OAPServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.OAPServer{}).
		Complete(r)
}
