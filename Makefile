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
mk_path  := $(abspath $(lastword $(MAKEFILE_LIST)))
mk_dir   := $(dir $(mk_path))
tool_bin := $(mk_dir)/bin

include $(mk_dir)/tools/build/base.mk

##@ Development

.PHONY: all
all: build docker-build

.PHONY: build
build: ## Build the binary
	$(MAKE) -C operator build
	$(MAKE) -C adapter build

.PHONY: docker-build
docker-build: ## Build docker images
	$(MAKE) -C operator docker-build
	$(MAKE) -C adapter docker-build

.PHONY: test
test: ## Run unit test cases
	$(MAKE) -C operator test
	$(MAKE) -C adapter test

.PHONY: clean
clean: ## Clean project
	rm -rf bin/
	rm -rf operator/bin/
	rm -rf adapter/bin/
	rm -rf build/bin
	rm -rf build/release
	rm -rf *.out
	rm -rf *.test

##@ End to End Test

# One target per directory under test/e2e, so the list cannot go stale: `make e2e-chart` runs
# test/e2e/chart/e2e.yaml. See docs/en/guides/e2e.md for what each case covers.
E2E_CASES := \
	banyandb \
	chart \
	oap-agent-adapter-hpa \
	oap-eventexporter \
	oap-satellite-adapter-hpa \
	oap-ui-agent \
	oap-ui-agent-external-storage \
	oap-ui-agent-internal-storage \
	oap-ui-agent-oapserverconfig-oapserverdynamicconfig \
	oap-ui-agent-satellite \
	oap-ui-swagent \
	oap-ui-swagent-configmap

E2E_TARGETS := $(addprefix e2e-,$(E2E_CASES))

.PHONY: e2e-test
e2e-test: $(E2E_TARGETS) ## Run every End to End test case

.PHONY: $(E2E_TARGETS)
$(E2E_TARGETS): e2e-%: e2e
	@echo "Run e2e case $*..."
	$(E2E) run -c test/e2e/$*/e2e.yaml

E2E = $(tool_bin)/cmd
.PHONY: e2e
e2e: ## Download e2e-setup locally if necessary.
	$(call go-get-tool,$(E2E),github.com/apache/skywalking-infra-e2e/cmd@v1.1.0)

##@ Helm chart

CHART_DIR := $(mk_dir)chart/skywalking-swck

# The CRDs, the manager ClusterRole and the admission webhook configurations the chart ships are
# GENERATED from the operator sources -- they are never hand-edited. That is the point of the chart
# living in this repository rather than in apache/skywalking-helm, where they were a hand-copied
# snapshot that nothing checked.
.PHONY: chart-manifests
chart-manifests: yq ## Regenerate the CRDs, RBAC and webhooks the Helm chart ships
	$(MAKE) -C operator manifests
	$(MAKE) -C operator kustomize
	KUSTOMIZE=$(tool_bin)/kustomize YQ=$(YQ) $(mk_dir)tools/generate-chart.sh

# Regenerates into a scratch directory and diffs, rather than regenerating in place and asking git
# what changed -- so the result does not depend on whether the working tree happens to be clean.
.PHONY: chart-check
chart-check: yq ## Fail if the generated parts of the chart are out of date
	$(MAKE) -C operator manifests
	$(MAKE) -C operator kustomize
	@scratch=`mktemp -d`; \
	OUTPUT_DIR=$$scratch KUSTOMIZE=$(tool_bin)/kustomize YQ=$(YQ) $(mk_dir)tools/generate-chart.sh > /dev/null; \
	status=0; \
	diff -r $$scratch/crds $(CHART_DIR)/crds || status=1; \
	diff $$scratch/templates/operator-manager-role.yaml $(CHART_DIR)/templates/operator-manager-role.yaml || status=1; \
	diff $$scratch/templates/operator-webhook.yaml $(CHART_DIR)/templates/operator-webhook.yaml || status=1; \
	rm -rf $$scratch; \
	if [ $$status -ne 0 ]; then \
		echo ""; \
		echo "The generated parts of the chart are out of date."; \
		echo "Run 'make chart-manifests' and commit the result."; \
		exit 1; \
	fi; \
	echo "chart manifests are in sync with the operator sources"

# Depends on yq because test/tools/check-chart-render.sh reads names back out of the rendered output;
# without it the target only works after chart-check has happened to install the tool.
.PHONY: chart-lint
chart-lint: yq ## Lint the chart and check that it renders what it is supposed to
	helm lint $(CHART_DIR)
	CHART_DIR=$(CHART_DIR) $(mk_dir)test/tools/check-chart-render.sh

##@ Code quality and integrity

.PHONY: check
check: ## Check that the status
	$(MAKE) -C operator lint
	$(MAKE) -C operator dependency-check
	$(MAKE) -C operator check
	$(MAKE) -C adapter format
	$(MAKE) -C adapter lint
	$(MAKE) -C adapter dependency-check
	$(MAKE) -C adapter check
	$(MAKE) license-check


##@ release

RELEASE_SCRIPTS := ./build/package/release.sh

release-binary: ## Package binary archive
	$(MAKE) -C operator kustomize
	$(MAKE) -C operator release-build
	$(MAKE) -C adapter release-build
	${RELEASE_SCRIPTS} -b

release-source: ## Package source archive
	${RELEASE_SCRIPTS} -s

release-chart: ## Package the Helm chart archive
	${RELEASE_SCRIPTS} -c

release-sign: ## Sign artifacts
	${RELEASE_SCRIPTS} -k bin
	${RELEASE_SCRIPTS} -k src
	${RELEASE_SCRIPTS} -k chart

release: release-binary release-source release-chart release-sign ## Generate release package
