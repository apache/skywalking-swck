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

package injector

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_needInject(t *testing.T) {
	type args struct {
		pod *corev1.Pod
	}

	// set a pod with a wrong label
	unlabeled_pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"swck-agent-injected": "false",
			},
		},
	}

	// set a pod with a right label
	labeled_pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"swck-agent-injected": "true",
			},
		},
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "don't need injected",
			args: args{
				pod: unlabeled_pod,
			},
			// false means we don't need injected
			want: false,
		},
		{
			name: "need injected",
			args: args{
				pod: labeled_pod,
			},
			// true means we need injected
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := needInject(tt.args.pod); got != tt.want {
				t.Errorf("needInject() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_addAgent(t *testing.T) {
	type args struct {
		pod *corev1.Pod
	}

	Pod := &corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				corev1.Container{
					Name: "app",
				},
			},
		},
	}

	// injected InitContainer's VolumeMount
	injectedICVM := corev1.VolumeMount{
		MountPath: "/sky/agent",
		Name:      "sky-agent",
	}

	// injected InitContainer
	injectedIC := corev1.Container{
		Name:         "inject-sky-agent",
		Image:        "apache/skywalking-java-agent:8.6.0-jdk8",
		Command:      []string{"sh"},
		Args:         []string{"-c", "mkdir -p /sky/agent && cp -r /skywalking/agent/* /sky/agent"},
		VolumeMounts: []corev1.VolumeMount{injectedICVM},
	}

	// injected Volumes
	injectedV := corev1.Volume{
		Name:         "sky-agent",
		VolumeSource: corev1.VolumeSource{EmptyDir: nil},
	}

	// injected Container's VolumeMount
	injectedVM := corev1.VolumeMount{
		MountPath: "/sky/agent",
		Name:      "sky-agent",
	}

	// injected Container's EnvVar
	injectedEV := corev1.EnvVar{
		Name:  "AGENT_OPTS",
		Value: " -javaagent:/sky/agent/skywalking-agent.jar",
	}

	injectedPod := &corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				corev1.Container{
					Name: "app",
					VolumeMounts: []corev1.VolumeMount{
						injectedVM,
					},
					Env: []corev1.EnvVar{
						injectedEV,
					},
				},
			},
			InitContainers: []corev1.Container{
				injectedIC,
			},
			Volumes: []corev1.Volume{
				injectedV,
			},
		},
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "test injected",
			args: args{Pod},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addAgent(tt.args.pod)
			if !reflect.DeepEqual(Pod, tt.args.pod) {
				t.Errorf("needInject() = %v, want %v end", Pod, injectedPod)
			}
		})
	}
}
