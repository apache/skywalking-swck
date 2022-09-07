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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	runtimelog "sigs.k8s.io/controller-runtime/pkg/log"

	operatorv1alpha1 "github.com/apache/skywalking-swck/operator/apis/operator/v1alpha1"
)

// SwAgentReconciler reconciles a SwAgent object
type SwAgentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=swagents,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=swagents/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.skywalking.apache.org,resources=swagents/finalizers,verbs=update

func (r *SwAgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := runtimelog.FromContext(ctx)
	log.Info("===================== SwAgent Reconcile (do nothing) ================================\n")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SwAgentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.SwAgent{}).
		Complete(r)
}
