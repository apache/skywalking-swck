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

package main

import (
	"flag"
	"os"
	"time"

	"github.com/kubernetes-sigs/custom-metrics-apiserver/pkg/apiserver"
	basecmd "github.com/kubernetes-sigs/custom-metrics-apiserver/pkg/cmd"
	generatedopenapi "github.com/kubernetes-sigs/custom-metrics-apiserver/test-adapter/generated/openapi"
	"k8s.io/apimachinery/pkg/util/wait"
	openapinamer "k8s.io/apiserver/pkg/endpoints/openapi"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/component-base/logs"
	"k8s.io/klog/v2"

	swckprov "github.com/apache/skywalking-swck/pkg/provider"
)

type Adapter struct {
	basecmd.AdapterBase
	// BaseURL is the address of OAP cluster
	BaseURL string
	// MetricRegex is a regular expression to filter metrics retrieved from OAP cluster
	MetricRegex string
	// RefreshRegistryInterval is the refresh window for syncing metrics with OAP cluster
	RefreshRegistryInterval time.Duration
	// Message is printed on successful startup
	Message string
	// Namespace groups metrics into a single set in case of duplicated metric name
	Namespace string
}

func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	cmd := &Adapter{}

	cmd.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(generatedopenapi.GetOpenAPIDefinitions, openapinamer.NewDefinitionNamer(apiserver.Scheme))
	cmd.OpenAPIConfig.Info.Title = "apache-skywalking-adapter"
	cmd.OpenAPIConfig.Info.Version = "1.0.0"

	cmd.Flags().StringVar(&cmd.Message, "msg", "starting adapter...", "startup message")
	cmd.Flags().StringVar(&cmd.BaseURL, "oap-addr", "http://oap:12800/graphql", "the address of OAP cluster")
	cmd.Flags().StringVar(&cmd.MetricRegex, "metric-filter-regex", "", "a regular expression to filter metrics retrieved from OAP cluster")
	cmd.Flags().StringVar(&cmd.Namespace, "namespace", "skywalking.apache.org", "a prefix to which metrics are appended. The format is 'namespace|metric_name'")
	cmd.Flags().DurationVar(&cmd.RefreshRegistryInterval, "refresh-interval", 10*time.Second,
		"the interval at which to update the cache of available metrics from OAP cluster")
	cmd.Flags().AddGoFlagSet(flag.CommandLine) // make sure we get the klog flags
	if err := cmd.Flags().Parse(os.Args); err != nil {
		klog.Fatalf("failed to parse arguments: %v", err)
	}

	p, err := swckprov.NewProvider(cmd.BaseURL, cmd.MetricRegex, cmd.RefreshRegistryInterval, cmd.Namespace)
	if err != nil {
		klog.Fatalf("unable to build p: %v", err)
	}
	cmd.WithExternalMetrics(p)

	klog.Infof(cmd.Message)
	if err := cmd.Run(wait.NeverStop); err != nil {
		klog.Fatalf("unable to run custom metrics adapter: %v", err)
	}
}
