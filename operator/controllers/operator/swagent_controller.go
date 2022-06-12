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
	"regexp"
	"strings"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
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

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the SwAgent object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *SwAgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := runtimelog.FromContext(ctx)
	log.Info("===================== SwAgent started================================\n")

	swAgent := &operatorv1alpha1.SwAgent{}
	if err := r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, swAgent); err != nil {
		log.Error(err, "unable to get SwAgent.")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	swAgentJson, _ := json.Marshal(swAgent)
	log.Info(fmt.Sprintf("sw agent cr: %s\n", string(swAgentJson)))

	// TODO: check namespace annotation "swck-injection=true"

	// TODO: move to validate webhook
	// setup default values
	r.setDefault(swAgent)

	// select pods
	currentPodList := &corev1.PodList{}
	if len(swAgent.Spec.Selector) > 0 {
		var matchingLabelsSelector client.MatchingLabels = swAgent.Spec.Selector
		r.List(ctx, currentPodList, client.InNamespace(req.Namespace), matchingLabelsSelector)
	} else {
		r.List(ctx, currentPodList, client.InNamespace(req.Namespace))
	}

	for _, currentPod := range currentPodList.Items {
		if err := r.patchPod(ctx, &currentPod, swAgent); err != nil {
			log.Error(err, "failed to patch pod.")
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
	}

	return ctrl.Result{}, nil
}

func (r *SwAgentReconciler) clientPatch(ctx context.Context, currentPod *corev1.Pod, desiredPod *corev1.Pod) error {
	patch := client.MergeFrom(currentPod)
	if err := r.Client.Patch(ctx, desiredPod, patch); err != nil {
		return fmt.Errorf("failed to patch pod. error: %w", err)
	}
	return nil
}

func (r *SwAgentReconciler) patchPod(ctx context.Context, currentPod *corev1.Pod, swAgent *operatorv1alpha1.SwAgent) error {
	desiredPod := currentPod.DeepCopy()

	// filter target containers
	targetContainers := r.getMatchedContainers(desiredPod, swAgent)

	if len(targetContainers) > 0 {
		// deal with shared volume and sw agent configmap
		r.setVolume(desiredPod, swAgent)

		// deal with initContainer
		r.setInitContainer(desiredPod, swAgent)

		// deal with target containers
		r.setTargetContainers(desiredPod, swAgent)

		// inform api server to patch pod.
		if err := r.clientPatch(ctx, currentPod, desiredPod); err != nil {
			return errors.Wrapf(err, "client patch failed.")
		}
	}

	return nil
}

func (r *SwAgentReconciler) getMatchedContainers(pod *corev1.Pod, swAgent *operatorv1alpha1.SwAgent) []*corev1.Container {
	var targetContainers []*corev1.Container
	for _, c := range pod.Spec.Containers {
		var containerRegExp = regexp.MustCompile(swAgent.Spec.ContainerMatcher)
		if containerRegExp.MatchString(c.Name) {
			targetContainers = append(targetContainers, &c)
		}
	}
	return targetContainers
}

func (r *SwAgentReconciler) setInitContainer(desiredPod *corev1.Pod, swAgent *operatorv1alpha1.SwAgent) {
	var initContainer *corev1.Container
	for _, ic := range desiredPod.Spec.InitContainers {
		if strings.EqualFold(swAgent.Spec.JavaSidecar.Name, ic.Name) {
			initContainer = &ic
		}
	}

	if nil == initContainer {
		initContainer = &corev1.Container{
			Name:    swAgent.Spec.JavaSidecar.Name,
			Image:   swAgent.Spec.JavaSidecar.Image,
			Args:    swAgent.Spec.JavaSidecar.Args,
			Command: swAgent.Spec.JavaSidecar.Command,
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      swAgent.Spec.SharedVolume.Name,
					MountPath: swAgent.Spec.SharedVolume.MountPath,
				},
			},
		}
		// todo: deal with plugins, args and command.
		desiredPod.Spec.InitContainers = append(desiredPod.Spec.InitContainers, *initContainer)
	}
}

func (r *SwAgentReconciler) setVolume(desiredPod *corev1.Pod, swAgent *operatorv1alpha1.SwAgent) {
	var sharedVolume *corev1.Volume
	var swConfigMap *corev1.Volume
	for _, v := range desiredPod.Spec.Volumes {
		if strings.EqualFold(swAgent.Spec.SharedVolume.Name, v.Name) {
			sharedVolume = &v
			break
		}
		if strings.EqualFold(swAgent.Spec.SwConfigMap.Name, v.Name) {
			swConfigMap = &v
			break
		}
	}
	if nil == sharedVolume {
		sharedVolume = &corev1.Volume{
			Name: swAgent.Spec.SharedVolume.Name,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}
		desiredPod.Spec.Volumes = append(desiredPod.Spec.Volumes, *sharedVolume)
	}
	if nil == swConfigMap {
		swConfigMap = &corev1.Volume{
			Name: swAgent.Spec.SwConfigMap.Name,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: swAgent.Spec.SwConfigMap.VolumeName,
					},
				},
			},
		}
		desiredPod.Spec.Volumes = append(desiredPod.Spec.Volumes, *swConfigMap)
	}
}

func (r *SwAgentReconciler) setTargetContainers(desiredPod *corev1.Pod, swAgent *operatorv1alpha1.SwAgent) {
	containers := r.getMatchedContainers(desiredPod, swAgent)
	for _, c := range containers {
		// deal with shared volume
		var sharedVolumeMounts *corev1.VolumeMount
		for _, vm := range c.VolumeMounts {
			if strings.EqualFold(vm.Name, swAgent.Spec.SharedVolume.Name) {
				sharedVolumeMounts = &vm
			}
		}
		if nil == sharedVolumeMounts {
			sharedVolumeMounts = &corev1.VolumeMount{
				Name:      swAgent.Spec.SharedVolume.Name,
				MountPath: swAgent.Spec.SharedVolume.MountPath,
			}
			c.VolumeMounts = append(c.VolumeMounts, *sharedVolumeMounts)
		}

		// deal with env
		desiredEnvs := swAgent.Spec.JavaSidecar.Env
		if !r.setEnvIfExists(&desiredEnvs, "SW_AGENT_COLLECTOR_BACKEND_SERVICES", c.Name) {
			swAgent.Spec.JavaSidecar.Env = append(swAgent.Spec.JavaSidecar.Env, corev1.EnvVar{
				Name:  "SW_AGENT_COLLECTOR_BACKEND_SERVICES",
				Value: c.Name,
			})
		}

		var addEnvs []corev1.EnvVar
		for j, desiredEnv := range desiredEnvs {
			envExists := false
			for k, curEnv := range c.Env {
				if strings.EqualFold(desiredEnv.Name, curEnv.Name) {
					c.Env[k].Value = desiredEnv.Value
					envExists = true
					continue
				}
			}
			if !envExists {
				addEnvs = append(addEnvs, desiredEnvs[j])
			}
		}
		if len(addEnvs) > 0 {
			c.Env = append(c.Env, addEnvs...)
		}

	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *SwAgentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.SwAgent{}).
		Complete(r)
}

func (r *SwAgentReconciler) setDefault(swAgent *operatorv1alpha1.SwAgent) {
	if nil != swAgent {
		if len(swAgent.Spec.Selector) == 0 {
			if swAgent.Spec.Selector == nil {
				swAgent.Spec.Selector = make(map[string]string)
			}
			swAgent.Spec.Selector[LabelAutoUpdate] = "true"
			swAgent.Spec.Selector[LabelJavaAgent] = "true"
		}
		if len(swAgent.Spec.ContainerMatcher) == 0 {
			swAgent.Spec.ContainerMatcher = ".*"
		}

		// default values for java sidecar
		if len(swAgent.Spec.JavaSidecar.Name) == 0 {
			swAgent.Spec.JavaSidecar.Name = "inject-skywalking-agent"
		}
		if len(swAgent.Spec.JavaSidecar.Image) == 0 {
			swAgent.Spec.JavaSidecar.Image = "apache/skywalking-java-agent:8.8.0-java8"
		}
		if len(swAgent.Spec.JavaSidecar.Command) == 0 {
			if swAgent.Spec.JavaSidecar.Command == nil {
				swAgent.Spec.JavaSidecar.Command = []string{}
			}
			swAgent.Spec.JavaSidecar.Command = append(swAgent.Spec.JavaSidecar.Command, "sh")
		}
		if len(swAgent.Spec.JavaSidecar.Args) == 0 {
			if swAgent.Spec.JavaSidecar.Args == nil {
				swAgent.Spec.JavaSidecar.Args = []string{}
			}
			swAgent.Spec.JavaSidecar.Args = append(swAgent.Spec.JavaSidecar.Args, "-c")
			swAgent.Spec.JavaSidecar.Args = append(swAgent.Spec.JavaSidecar.Args, "mkdir -p /sky/agent && cp -r /skywalking/agent/* /sky/agent")
		}
		r.setOrAddEnv(swAgent, "JAVA_TOOL_OPTIONS", " -javaagent:/sky/agent/skywalking-agent.jar")

		// default values for shared volume
		if len(swAgent.Spec.SharedVolume.Name) == 0 {
			swAgent.Spec.SharedVolume.Name = "sky-agent"
		}
		if len(swAgent.Spec.SharedVolume.MountPath) == 0 {
			swAgent.Spec.SharedVolume.MountPath = "/sky/agent"
		}

		// default values for agent configmap
		if len(swAgent.Spec.SwConfigMap.Name) == 0 {
			swAgent.Spec.SwConfigMap.Name = "java-agent-configmap-volume"
		}
		if len(swAgent.Spec.SwConfigMap.VolumeName) == 0 {
			swAgent.Spec.SwConfigMap.VolumeName = "skywalking-swck-java-agent-configmap"
		}
		if len(swAgent.Spec.SwConfigMap.VolumeMountPath) == 0 {
			swAgent.Spec.SwConfigMap.VolumeMountPath = "/sky/agent/config"
		}

	}

	// todo: deal with plugins
}

func (r *SwAgentReconciler) setOrAddEnv(swAgent *operatorv1alpha1.SwAgent, envKey string, envValue string) {
	if !r.setEnvIfExists(&swAgent.Spec.JavaSidecar.Env, envKey, envValue) {
		javaToolOptionsEnv := corev1.EnvVar{
			Name:  envKey,
			Value: envValue,
		}
		swAgent.Spec.JavaSidecar.Env = append(swAgent.Spec.JavaSidecar.Env, javaToolOptionsEnv)
	}
}

func (r *SwAgentReconciler) setEnvIfExists(envs *[]corev1.EnvVar, envKey string, envValue string) bool {
	for _, env := range *envs {
		if strings.EqualFold(env.Name, envKey) {
			env.Value = envValue
			return true
		}
	}
	return false
}

const (
	LabelAutoUpdate = "skywalking-swck-auto-update"
	LabelJavaAgent  = "swck-java-agent-injected"
)
