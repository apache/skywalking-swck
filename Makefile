# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

# Image URL to use all building/pushing image targets
OPERATOR_IMG ?= controller:latest
ADAPTER_IMG ?= metrics-adapter:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

ARCH := $(shell uname)
OSNAME := $(if $(findstring Darwin,$(ARCH)),darwin,linux)
GOBINDATA_VERSION := v3.21.0

# import local settings
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

all: operator adapter

clean:
	rm -rf bin/
	rm -rf build/bin
	rm -rf build/release
	rm -rf *.out
	rm -rf *.test

# Run tests
test: generate operator-manifests
	go test ./... -coverprofile cover.out

# Build manager binary
operator: generate
	go build -o bin/manager cmd/manager/manager.go

# Install dev CRDs into a cluster
operator-dev-install: operator-manifests operator-dev-uninstall
	kustomize build config/dev/operator/crd | kubectl apply -f -

# Uninstall dev CRDs from a cluster
operator-dev-uninstall: operator-manifests operator-undeploy

# Install CRDs into a cluster
operator-install: operator-manifests
	kustomize build config/operator/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
operator-uninstall: operator-manifests
	kustomize build config/operator/crd | kubectl delete --ignore-not-found=true -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
operator-deploy: operator-manifests
	@echo "Deploy operator"
	-hack/operator-deploy.sh d

# Undeploy controller in the configured Kubernetes cluster in ~/.kube/config
operator-undeploy: operator-manifests
	@echo "Undeploy operator"
	-hack/operator-deploy.sh u

# Generate manifests e.g. CRD, RBAC etc.
operator-manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) webhook paths="./apis/..." output:crd:artifacts:config=config/operator/crd/bases \
		output:webhook:artifacts:config=config/operator/webhook \
		&& $(CONTROLLER_GEN) $(CRD_OPTIONS) paths="./apis/..." output:crd:artifacts:config=config/dev/operator/crd/bases \
		&& $(CONTROLLER_GEN) rbac:roleName=manager-role paths="./controllers/..."  output:rbac:dir=config/operator/rbac \
		&& go run github.com/apache/skywalking-swck/cmd/build license insert config
 
# Build adapter binary
adapter:
	go build -o bin/adapter cmd/adapter/adapter.go

# Deploy adapter in the configured Kubernetes cluster in ~/.kube/config
adapter-deploy:
	kind load docker-image ${ADAPTER_IMG}
	kustomize build config/dev/adapter | kubectl apply -f -

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./apis/..."
	$(MAKE) format

GO_LICENSER := $(GOBIN)/go-licenser
$(GO_LICENSER):
	GO111MODULE=off go get -u github.com/elastic/go-licenser
license: $(GO_LICENSER)
	$(GO_LICENSER) -d -licensor='Apache Software Foundation (ASF)' -exclude=apis/operator/v1alpha1/zz_generated* .
	go run github.com/apache/skywalking-swck/cmd/build license check config
	go run github.com/apache/skywalking-swck/cmd/build license check pkg/operator/manifests

.PHONY: license

# Build the docker image
operator-docker-build:
	docker build . -f build/images/Dockerfile.operator -t ${OPERATOR_IMG}

# Push the docker image
operator-docker-push:
	docker push ${OPERATOR_IMG}

# Build the docker image
adapter-docker-build:
	docker build . -f build/images/Dockerfile.adapter -t ${ADAPTER_IMG}

# Push the docker image
adapter-docker-push:
	docker push ${ADAPTER_IMG}

docker-build: operator-docker-build adapter-docker-build

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.5 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

GOIMPORTS := $(GOBIN)/goimports
$(GOIMPORTS):
	GO111MODULE=off go get -u golang.org/x/tools/cmd/goimports
# The goimports tool does not arrange imports in 3 blocks if there are already more than three blocks.
# To avoid that, before running it, we collapse all imports in one block, then run the formatter.
format: $(GOIMPORTS) ## Format all Go code
	@for f in `find . -name '*.go'`; do \
	    awk '/^import \($$/,/^\)$$/{if($$0=="")next}{print}' $$f > /tmp/fmt; \
	    mv /tmp/fmt $$f; \
	done
	$(GOIMPORTS) -w -local github.com/apache/skywalking-swck .

## Check that the status is consistent with CI.
check: generate operator-manifests update-templates license
	$(MAKE) format
	mkdir -p /tmp/artifacts
	git diff >/tmp/artifacts/check.diff 2>&1
	@go mod tidy &> /dev/null
	@if [ ! -z "`git status -s`" ]; then \
		echo "Following files are not consistent with CI:"; \
		git status -s; \
		exit 1; \
	fi

## Code quality and integrity

LINTER := $(GOBIN)/golangci-lint
$(LINTER):
	wget -O - -q https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | BINDIR=$(GOBIN) sh -s v1.33.0
	
lint: $(LINTER)
	$(LINTER) run --config ./golangci.yml

.PHONY: lint

GO_BINDATA := $(GOBIN)/go-bindata
$(GO_BINDATA):
	curl --location --output $(GO_BINDATA) https://github.com/kevinburke/go-bindata/releases/download/v3.21.0/go-bindata-$(OSNAME)-amd64 \
		&& chmod +x $(GO_BINDATA)
		
update-templates: $(GO_BINDATA)
	@echo updating charts
	-hack/run_update_templates.sh
	-hack/build-header.sh pkg/operator/repo/assets.gen.go

release-operator: generate
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o build/bin/manager-linux-amd64 cmd/manager/manager.go

release-adapter: generate
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o build/bin/adapter-linux-amd64 cmd/adapter/adapter.go

RELEASE_SCRIPTS := ./build/package/release.sh

release-binary: release-operator release-adapter
	${RELEASE_SCRIPTS} -b
	
release-source:
	${RELEASE_SCRIPTS} -s
	
release-sign:
	${RELEASE_SCRIPTS} -k bin
	${RELEASE_SCRIPTS} -k src

release: release-binary release-source release-sign

.PHONY: release-manager release-binary release-source release
