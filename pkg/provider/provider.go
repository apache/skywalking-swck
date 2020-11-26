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
	"flag"
	"fmt"
	"sync"
	"time"

	"github.com/apache/skywalking-cli/assets"
	"github.com/apache/skywalking-cli/commands/interceptor"
	"github.com/apache/skywalking-cli/graphql/client"
	swctlschema "github.com/apache/skywalking-cli/graphql/schema"
	"github.com/apache/skywalking-cli/graphql/utils"
	apiprovider "github.com/kubernetes-sigs/custom-metrics-apiserver/pkg/provider"
	"github.com/machinebox/graphql"
	"github.com/urfave/cli"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	apischema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/klog/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

var (
	NsGroupResource = apischema.GroupResource{Resource: "namespaces"}
)

// externalMetricsProvider is a implementation of provider.MetricsProvider which provides metrics from OAP
type externalMetricsProvider struct {
	metricDefines           []*swctlschema.MetricDefinition
	lock                    sync.RWMutex
	ctx                     *cli.Context
	regex                   string
	refreshRegistryInterval time.Duration
}

type stringValue string

func (sv *stringValue) String() string {
	return string(*sv)
}

func (sv *stringValue) Set(s string) error {
	*sv = stringValue(s)
	return nil
}

// NewProvider returns an instance of externalMetricsProvider
func NewProvider(baseURL string, metricRegex string, refreshRegistryInterval time.Duration) (apiprovider.ExternalMetricsProvider, error) {
	fs := flag.NewFlagSet("mock", flag.ContinueOnError)
	var k stringValue
	if err := k.Set(baseURL); err != nil {
		return nil, fmt.Errorf("failed to set OAP address: %v", err)
	}
	fs.Var(&k, "base-url", "")
	ctx := cli.NewContext(nil, fs, nil)
	if url := ctx.GlobalString("base-url"); url != k.String() {
		return nil, fmt.Errorf("failed to set base-url: %s", url)
	}
	provider := &externalMetricsProvider{
		ctx:                     ctx,
		regex:                   metricRegex,
		refreshRegistryInterval: refreshRegistryInterval,
	}
	provider.sync()

	return provider, nil
}

type paramValue struct {
	key string
	val *string
}

func (p *externalMetricsProvider) GetExternalMetric(namespace string, metricSelector labels.Selector,
	info apiprovider.ExternalMetricInfo) (*external_metrics.ExternalMetricValueList, error) {
	var md *swctlschema.MetricDefinition
	for _, m := range p.metricDefines {
		if m.Name == info.Metric {
			md = m
		}
	}
	if md == nil {
		klog.Errorf("%s is missing in OAP", info.Metric)
		return nil, apierr.NewBadRequest(fmt.Sprintf("%s is defined in OAP", info.Metric))
	}
	requirement, selector := metricSelector.Requirements()
	if !selector {
		klog.Errorf("no selector for metric: %s", md.Name)
		return nil, apierr.NewBadRequest(fmt.Sprintf("no selector for metric: %s", md.Name))
	}
	svc := &paramValue{key: "service"}
	label := &paramValue{key: "label"}
	instance := &paramValue{key: "instance"}
	endpoint := &paramValue{key: "endpoint"}
	extractValue(requirement, svc, label, instance, endpoint)
	if *svc.val == "" {
		klog.Errorf("%s is lack of required label 'service'", md.Name)
		return nil, apierr.NewBadRequest(fmt.Sprintf("%s is lack of required label 'service'", md.Name))
	}

	now := time.Now()
	startTime := now.Add(-3 * time.Minute)
	endTime := now
	step := swctlschema.StepMinute
	duration := swctlschema.Duration{
		Start: startTime.Format(utils.StepFormats[step]),
		End:   endTime.Format(utils.StepFormats[step]),
		Step:  step,
	}

	normal := true
	condition := swctlschema.MetricsCondition{
		Name: md.Name,
		Entity: &swctlschema.Entity{
			Scope:               interceptor.ParseScope(md.Name),
			ServiceName:         svc.val,
			ServiceInstanceName: instance.val,
			EndpointName:        endpoint.val,
			Normal:              &normal,
		},
	}
	var metricsValues swctlschema.MetricsValues
	if md.Type == swctlschema.MetricsTypeRegularValue {
		var response map[string]swctlschema.MetricsValues

		request := graphql.NewRequest(assets.Read("graphqls/metrics/MetricsValues.graphql"))

		request.Var("condition", condition)
		request.Var("duration", duration)

		if err := client.ExecuteQuery(p.ctx, request, &response); err != nil {
			return nil, apierr.NewInternalError(fmt.Errorf("unable to fetch metrics: %v", err))
		}

		klog.V(4).Infof("Linear request{condition:%s, duration:%s}  response %s", display(condition), display(duration), display(response))

		metricsValues = response["result"]
	} else if md.Type == swctlschema.MetricsTypeLabeledValue {
		if *label.val == "" {
			klog.Errorf("%s is lack of required label 'label'", md.Name)
			return nil, apierr.NewBadRequest(fmt.Sprintf("%s is lack of required label 'label'", md.Name))
		}
		var response map[string][]swctlschema.MetricsValues

		request := graphql.NewRequest(assets.Read("graphqls/metrics/LabeledMetricsValues.graphql"))

		request.Var("duration", duration)
		request.Var("condition", condition)
		request.Var("labels", []string{*label.val})

		if err := client.ExecuteQuery(p.ctx, request, &response); err != nil {
			return nil, apierr.NewInternalError(fmt.Errorf("unable to fetch metrics: %v", err))
		}

		klog.V(4).Infof("Labeled request{condition:%s, duration:%s, labels:%s}  response %s",
			display(condition), display(duration), *label.val, display(response))

		result := response["result"]

		for _, r := range result {
			if *r.Label == *label.val {
				metricsValues = r
			}
		}
	}
	if len(metricsValues.Values.Values) < 1 {
		return nil, apiprovider.NewMetricNotFoundError(p.selectGroupResource(namespace), info.Metric)
	}
	var sTime time.Time
	var sValue int64
	l := len(metricsValues.Values.Values)
	if l < 2 {
		return nil, apiprovider.NewMetricNotFoundError(p.selectGroupResource(namespace), info.Metric)
	}
	kv := metricsValues.Values.Values[l-2]
	sTime = endTime.Add(time.Minute * time.Duration(-1))
	if kv.Value > 0 {
		sValue = kv.Value
	}
	if sValue == 0 {
		sTime = endTime
	}
	klog.V(4).Infof("metric value: %d, timestamp: %s", sValue, sTime.Format(utils.StepFormats[step]))

	return &external_metrics.ExternalMetricValueList{
		Items: []external_metrics.ExternalMetricValue{
			{
				MetricName: info.Metric,
				Timestamp: metav1.Time{
					Time: sTime,
				},
				Value: *resource.NewQuantity(sValue, resource.DecimalSI),
			},
		},
	}, nil
}

func extractValue(requirement labels.Requirements, paramValues ...*paramValue) {
	for _, r := range requirement {
		for _, pv := range paramValues {
			if r.Key() == pv.key && r.Operator() == selection.Equals {
				if v, exist := r.Values().PopAny(); exist {
					pv.val = &v
				}
			}
		}
	}
}

func (p *externalMetricsProvider) selectGroupResource(namespace string) apischema.GroupResource {
	if namespace == "default" {
		return NsGroupResource
	}

	return apischema.GroupResource{
		Group:    "",
		Resource: "",
	}
}
