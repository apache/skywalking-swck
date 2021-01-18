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

package controllers_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8sreconcile "sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/apache/skywalking-swck/apis/operator/v1alpha1"
	controllers "github.com/apache/skywalking-swck/controllers/operator"
	"github.com/apache/skywalking-swck/pkg/operator/repo"
)

func TestFetcherNewObjectsOnReconciliation(t *testing.T) {
	// prepare
	nsn := types.NamespacedName{Name: "my-instance", Namespace: "default"}
	reconciler := controllers.FetcherReconciler{
		Client:   k8sClient,
		Log:      logger,
		Scheme:   testScheme,
		FileRepo: repo.NewRepo("fetcher"),
		Recorder: record.NewFakeRecorder(100),
	}
	created := &v1alpha1.Fetcher{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Fetcher",
			APIVersion: v1alpha1.GroupVersion.Version,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      nsn.Name,
			Namespace: nsn.Namespace,
		},
		Spec: v1alpha1.FetcherSpec{
			Type:             []v1alpha1.FetcherType{v1alpha1.FetcherTypePrometheus},
			OAPServerAddress: "oap.skywalking:12800",
		},
	}
	created.Default()
	err := k8sClient.Create(context.Background(), created)
	require.NoError(t, err)

	// test
	req := k8sreconcile.Request{
		NamespacedName: nsn,
	}
	_, err = reconciler.Reconcile(context.Background(), req)

	// verify
	require.NoError(t, err)

	// the base query for the underlying objects
	opts := []client.ListOption{
		client.InNamespace(nsn.Namespace),
		client.MatchingLabels(map[string]string{
			"operator.skywalking.apache.org/fetcher-name": nsn.Name,
			"operator.skywalking.apache.org/application":  "fetcher",
		}),
	}

	// verify that we have at least one object for each of the types we create
	// whether we have the right ones is up to the specific tests for each type
	{
		list := &corev1.ConfigMapList{}
		err = k8sClient.List(context.Background(), list, opts...)
		assert.NoError(t, err)
		assert.NotEmpty(t, list.Items)
	}
	{
		list := &appsv1.DeploymentList{}
		err = k8sClient.List(context.Background(), list, opts...)
		assert.NoError(t, err)
		assert.NotEmpty(t, list.Items)
	}

	// cleanup
	require.NoError(t, k8sClient.Delete(context.Background(), created))

}
