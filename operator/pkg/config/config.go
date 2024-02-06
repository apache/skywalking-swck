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

package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

type Config struct {
	ApiVersion     string               `yaml:"apiVersion"`
	Kind           string               `yaml:"kind"`
	Health         HealthConfig         `yaml:"health"`
	Metrics        MetricsConfig        `yaml:"metrics"`
	Webhook        WebhookConfig        `yaml:"webhook"`
	LeaderElection LeaderElectionConfig `yaml:"leaderElection"`
}

type HealthConfig struct {
	HealthProbeBindAddress string `yaml:"healthProbeBindAddress"`
}

type MetricsConfig struct {
	MetricsBindAddress string `yaml:"bindAddress"`
}

type WebhookConfig struct {
	Port int `yaml:"port"`
}

type LeaderElectionConfig struct {
	Enabled    bool   `yaml:"leaderElect"`
	ResourceID string `yaml:"resourceName"`
}

func ParseFile(path string) (*Config, error) {
	fd, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open the configuration file: %v", err)
	}
	defer fd.Close()

	cfg := Config{}

	if err = yaml.NewDecoder(fd).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("could not decode configuration file: %v", err)
	}

	return &cfg, nil
}

func (c *Config) ManagerOptions() *manager.Options {
	return &manager.Options{
		HealthProbeBindAddress: c.Health.HealthProbeBindAddress,
		LeaderElection:         c.LeaderElection.Enabled,
		LeaderElectionID:       c.LeaderElection.ResourceID,
		Metrics: metricsserver.Options{
			BindAddress: c.Metrics.MetricsBindAddress,
		},
		WebhookServer: webhook.NewServer(webhook.Options{Port: c.Webhook.Port}),
	}
}
