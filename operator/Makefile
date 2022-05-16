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
OPERATOR_IMG ?= controller:latest
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.22

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

ARCH := $(shell uname)
OSNAME := $(if $(findstring Darwin,$(ARCH)),darwin,linux)

module_path  := $(abspath $(lastword $(MAKEFILE_LIST)))
module_dir   := $(dir $(module_path))

include ../hack/build/module.mk

.PHONY: all
all: build docker-build

##@ Development

generate: generate-deep manifests ## Generate codes and manifests
	$(MAKE) format

.PHONY: manifests
manifests: controller-gen licenseeye ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases \
	  && $(LICENSEEYE) -c .manifests.licenserc.yaml header fix
    		
.PHONY: generate
generate-deep: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="../hack/boilerplate.go.txt" paths="./..."

.PHONY: test
test: generate format envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ./... -coverprofile cover.out

##@ Build

.PHONY: build
build: generate ## Build manager binary.
	go build -o bin/manager main.go

.PHONY: docker-build
docker-build: ## Build docker image with the manager.
	docker build . -t ${OPERATOR_IMG}

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${OPERATOR_IMG}

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy operator controller to the K8s cluster specified in ~/.kube/config.
	@echo "Deploy operator"
	-MOD=operator DIR=default IMG_PATH=manager IMG=controller NEW_IMG=${OPERATOR_IMG} $(root_dir)/hack/deploy.sh d

.PHONY: undeploy
undeploy: manifests ## Undeploy operator controller from the K8s cluster specified in ~/.kube/config.
	@echo "Undeploy operator"
	-MOD=operator DIR=default IMG_PATH=manager IMG=controller NEW_IMG=${OPERATOR_IMG} $(root_dir)/hack/deploy.sh u

##@ Release

.PHONY: release-build
release-build: generate ## Build binary for release
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -ldflags="-s -w" -o $(root_dir)/build/bin/manager-linux-amd64
