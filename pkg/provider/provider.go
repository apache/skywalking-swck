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
	"encoding/hex"
	"flag"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	swctlapi "github.com/apache/skywalking-cli/api"
	"github.com/apache/skywalking-cli/pkg/graphql/metrics"
	apiprovider "github.com/kubernetes-sigs/custom-metrics-apiserver/pkg/provider"
	"github.com/urfave/cli"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	apischema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

const labelValueTypeStr string = "str"
const labelValueTypeByte string = "byte"
const stepMinute string = "2006-01-02 1504"

var (
	NsGroupResource = apischema.GroupResource{Resource: "namespaces"}
)

// externalMetricsProvider is a implementation of provider.MetricsProvider which provides metrics from OAP
type externalMetricsProvider struct {
	metricDefines           []*swctlapi.MetricDefinition
	lock                    sync.RWMutex
	ctx                     *cli.Context
	regex                   string
	refreshRegistryInterval time.Duration
	namespace               string
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
func NewProvider(baseURL string, metricRegex string, refreshRegistryInterval time.Duration, namespace string) (apiprovider.ExternalMetricsProvider, error) {
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
		namespace:               namespace,
	}
	provider.sync()

	return provider, nil
}

type paramValue struct {
	key string
	val *string
}

func (pv *paramValue) extractValue(requirements labels.Requirements) error {
	vv := make([]string, 10)
	for _, r := range requirements {
		if !strings.HasPrefix(r.Key(), pv.key) {
			continue
		}
		kk := strings.Split(r.Key(), ".")
		if len(kk) == 1 {
			vv, _ = bufferEntity(vv, 0, r, nil)
			pv.val = &vv[0]
			return nil
		} else if len(kk) < 3 {
			return fmt.Errorf("invalid label key:%s", r.Key())
		}
		i, err := strconv.ParseInt(kk[2], 10, 8)
		if err != nil {
			return fmt.Errorf("failed to parse index string to int: %v", err)
		}

		index := int(i)
		switch kk[1] {
		case labelValueTypeStr:
			vv, _ = bufferEntity(vv, index, r, nil)
		case labelValueTypeByte:
			vv, err = bufferEntity(vv, index, r, func(encoded string) (string, error) {
				var bytes []byte
				bytes, err = hex.DecodeString(encoded)
				if err != nil {
					return "", fmt.Errorf("failed to decode hex to string: %v", err)
				}
				return string(bytes), nil
			})
			if err != nil {
				return err
			}
		}
	}
	val := strings.Join(vv, "")
	pv.val = &val
	return nil
}

type decoder func(string) (string, error)

func bufferEntity(buff []string, index int, requirement labels.Requirement, dec decoder) ([]string, error) {
	if cap(buff) <= index {
		old := buff
		buff = make([]string, index+1)
		copy(buff, old)
	}
	if v, exist := requirement.Values().PopAny(); exist {
		if dec != nil {
			var err error
			buff[index], err = dec(v)
			if err != nil {
				return nil, err
			}
		} else {
			buff[index] = v
		}
	}
	return buff, nil
}

func (p *externalMetricsProvider) GetExternalMetric(namespace string, metricSelector labels.Selector,
	info apiprovider.ExternalMetricInfo) (*external_metrics.ExternalMetricValueList, error) {
	var md *swctlapi.MetricDefinition
	for _, m := range p.metricDefines {
		if p.getMetricNameWithNamespace(m.Name) == info.Metric {
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
	step := swctlapi.StepMinute
	duration := swctlapi.Duration{
		Start: startTime.Format(stepMinute),
		End:   endTime.Format(stepMinute),
		Step:  step,
	}

	normal := true
	empty := ""
	entity := &swctlapi.Entity{
		ServiceName:             svc.val,
		ServiceInstanceName:     instance.val,
		EndpointName:            endpoint.val,
		Normal:                  &normal,
		DestServiceName:         &empty,
		DestNormal:              &normal,
		DestServiceInstanceName: &empty,
		DestEndpointName:        &empty,
	}
	entity.Scope = parseScope(entity)
	condition := swctlapi.MetricsCondition{
		Name:   md.Name,
		Entity: entity,
	}
	var metricsValues swctlapi.MetricsValues
	if md.Type == swctlapi.MetricsTypeRegularValue {
		var err error
		metricsValues, err = metrics.LinearIntValues(p.ctx, condition, duration)
		if err != nil {
			return nil, apierr.NewInternalError(fmt.Errorf("unable to fetch metrics: %v", err))
		}
		klog.V(4).Infof("Linear request{condition:%s, duration:%s}  response %s", display(condition), display(duration), display(metricsValues))
	} else if md.Type == swctlapi.MetricsTypeLabeledValue {
		if *label.val == "" {
			klog.Errorf("%s is lack of required label 'label'", md.Name)
			return nil, apierr.NewBadRequest(fmt.Sprintf("%s is lack of required label 'label'", md.Name))
		}
		result, err := metrics.MultipleLinearIntValues(p.ctx, condition, []string{*label.val}, duration)
		if err != nil {
			return nil, apierr.NewInternalError(fmt.Errorf("unable to fetch metrics: %v", err))
		}

		klog.V(4).Infof("Labeled request{condition:%s, duration:%s, labels:%s}  response %s",
			display(condition), display(duration), *label.val, display(result))

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
	klog.V(4).Infof("metric value: %d, timestamp: %s", sValue, sTime.Format(stepMinute))

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
	for _, pv := range paramValues {
		err := pv.extractValue(requirement)
		if err != nil {
			klog.Errorf("failed to parse label %s: %v ", pv.key, err)
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

func (p *externalMetricsProvider) getMetricNameWithNamespace(metricName string) string {
	return strings.Join([]string{p.namespace, metricName}, "|")
}

// TODO: remove this function once cli move it from internal module to pkg
func parseScope(entity *swctlapi.Entity) swctlapi.Scope {
	scope := swctlapi.ScopeAll

	if *entity.DestEndpointName != "" {
		scope = swctlapi.ScopeEndpointRelation
	} else if *entity.DestServiceInstanceName != "" {
		scope = swctlapi.ScopeServiceInstanceRelation
	} else if *entity.DestServiceName != "" {
		scope = swctlapi.ScopeServiceRelation
	} else if *entity.EndpointName != "" {
		scope = swctlapi.ScopeEndpoint
	} else if *entity.ServiceInstanceName != "" {
		scope = swctlapi.ScopeServiceInstance
	} else if *entity.ServiceName != "" {
		scope = swctlapi.ScopeService
	}

	return scope
}
