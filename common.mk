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

##@ Code quality and integrity

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
check: generate manifests update-templates license-check ## Check that the status
	$(MAKE) format
	mkdir -p /tmp/artifacts
	git diff >/tmp/artifacts/check.diff 2>&1
	@go mod tidy &> /dev/null
	@if [ ! -z "`git status -s`" ]; then \
		echo "Following files are not consistent with CI:"; \
		git status -s; \
		cat /tmp/artifacts/check.diff; \
		exit 1; \
	fi

LINTER := $(GOBIN)/golangci-lint
$(LINTER):
	wget -O - -q https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | BINDIR=$(GOBIN) sh -s v1.33.0
	
lint: $(LINTER) ## Lint codes
	$(LINTER) run --config ./golangci.yml

.PHONY: lint

GO_BINDATA := $(shell pwd)/bin/go-bindata
$(GO_BINDATA):
	curl --location --output $(GO_BINDATA) https://github.com/kevinburke/go-bindata/releases/download/v3.21.0/go-bindata-$(OSNAME)-amd64 \
		&& chmod +x $(GO_BINDATA)

update-templates: $(GO_BINDATA) ## Update templates
	@echo updating charts
	-hack/run_update_templates.sh
	-hack/build-header.sh pkg/operator/repo/assets.gen.go


CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
.PHONY: controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.7.0)

KUSTOMIZE = $(shell pwd)/bin/kustomize
.PHONY: kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v4@v4.4.1)

ENVTEST = $(shell pwd)/bin/setup-envtest
.PHONY: envtest
envtest: ## Download envtest-setup locally if necessary.
	$(call go-get-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest@latest)
	
E2E = $(shell pwd)/bin/cmd
.PHONY: e2e
e2e: ## Download e2e-setup locally if necessary.
	$(call go-get-tool,$(E2E),github.com/apache/skywalking-infra-e2e/cmd@v1.1.0)

##@ License targets

LICENSEEYE = $(shell pwd)/bin/license-eye
.PHONY: licenseeye
licenseeye: ## Download skywalking-eye locally if necessary.
	$(call go-get-tool,$(LICENSEEYE),github.com/apache/skywalking-eyes/cmd/license-eye@v0.2.0)

.PHONY: license-check
license-check: licenseeye ## Check license header
	$(LICENSEEYE) header check

.PHONY: license-fix
license-fix: licenseeye ## Fix license header issues
	$(LICENSEEYE) header fix

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef
