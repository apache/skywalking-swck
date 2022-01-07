# Licensed to Apache Software Foundation (ASF) under one or more contributor
# license agreements. See the NOTICE file distributed with
# this work for additional information regarding copyright
# ownership. Apache Software Foundation (ASF) licenses this file to you under
# the Apache License, Version 2.0 (the "License"); you may
# not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing,
# software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied.  See the License for the
# specific language governing permissions and limitations
# under the License.
#

# Image URL to use all building/pushing image targets
ADAPTER_IMG ?= metrics-adapter:latest
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.22

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

module_path  := $(abspath $(lastword $(MAKEFILE_LIST)))
module_dir   := $(dir $(module_path))

include ../hack/build/module.mk

.PHONY: all
all: build docker-build

##@ Development

.PHONY: test
test: format envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ./... -coverprofile cover.out

##@ Build

.PHONY: build
build: ## Build adapter binary.
	go build -o bin/adapter main.go

.PHONY: docker-build
docker-build: ## Build docker image with the adapter.
	docker build . -t ${ADAPTER_IMG}

.PHONY: docker-push
docker-push: ## Push docker image with the adapter.
	docker push ${ADAPTER_IMG}

##@ Deployment

.PHONY: deploy
deploy: kustomize ## Deploy adapter controller to the K8s cluster specified in ~/.kube/config.
	@echo "Deploy adapter"
	-MOD=adapter DIR="./" IMG_PATH=namespaced/adapter IMG=metrics-adapter NEW_IMG=${ADAPTER_IMG} $(root_dir)/hack/deploy.sh d

.PHONY: release-build
release-build: ## Build binary for release
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -ldflags="-s -w" -o $(root_dir)/build/bin/adapter-linux-amd64
	