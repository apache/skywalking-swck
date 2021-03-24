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

package provider

import (
	"encoding/json"

	"github.com/apache/skywalking-cli/assets"
	"github.com/apache/skywalking-cli/graphql/client"
	swctlschema "github.com/apache/skywalking-cli/graphql/schema"
	apiprovider "github.com/kubernetes-sigs/custom-metrics-apiserver/pkg/provider"
	"github.com/machinebox/graphql"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
)

func (p *externalMetricsProvider) ListAllExternalMetrics() (externalMetricsInfo []apiprovider.ExternalMetricInfo) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	for _, md := range p.metricDefines {
		info := apiprovider.ExternalMetricInfo{
			Metric: p.getMetricNameWithNamespace(md.Name),
		}
		externalMetricsInfo = append(externalMetricsInfo, info)
	}
	return
}

func (p *externalMetricsProvider) sync() {
	go wait.Until(func() {
		if err := p.updateMetrics(); err != nil {
			klog.Errorf("failed to update metrics: %v", err)
		}
	}, p.refreshRegistryInterval, wait.NeverStop)
}

func (p *externalMetricsProvider) updateMetrics() error {
	var response map[string][]*swctlschema.MetricDefinition
	request := graphql.NewRequest(assets.Read("graphqls/metrics/ListMetrics.graphql"))
	request.Var("regex", p.regex)

	err := client.ExecuteQuery(p.ctx, request, &response)
	if err != nil {
		return err
	}

	mdd := response["result"]
	klog.Infof("Get service metrics: %s", display(mdd))
	if len(mdd) > 0 {
		p.lock.Lock()
		defer p.lock.Unlock()
		p.metricDefines = mdd
	}
	return nil
}

func display(data interface{}) string {
	bytes, e := json.Marshal(data)
	if e != nil {
		return "Error"
	}
	return string(bytes)

}
