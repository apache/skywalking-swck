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

include $(mk_dir)/hack/build/base.mk

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

.PHONY:e2e-test
e2e-test: e2e-oap-ui-agent e2e-oap-ui-swagent e2e-oap-ui-agent-storage-internal e2e-oap-agent-adapter-hpa e2e-oap-ui-agent-satellite e2e-oap-agent-satellite-adapter-hpa e2e-oap-ui-agent-oapserverconfig-oapserverdynamicconfig ## Run End to End tests.

.PHONY:e2e-oap-ui-agent
e2e-oap-ui-agent: e2e ## Run oap+ui+agent test
	@echo "Run oap+ui+agent e2e..."
	$(E2E) run -c test/e2e/oap-ui-agent/e2e.yaml

.PHONY:e2e-oap-ui-swagent
e2e-oap-ui-swagent: e2e ## Run oap+ui+swagent test
	@echo "Run oap+ui+swagent e2e..."
	$(E2E) run -c test/e2e/oap-ui-swagent/e2e.yaml

.PHONY:e2e-oap-ui-agent-storage-internal
e2e-oap-ui-agent-internal-storage: e2e ## Run oap+ui+agent test
	@echo "Run oap+ui+agent e2e..."
	$(E2E) run -c test/e2e/oap-ui-agent-internal-storage/e2e.yaml 

.PHONY:e2e-oap-agent-adapter-hpa
e2e-oap-agent-adapter-hpa: e2e ## Run oap+agent+adapter HPA test
	@echo "Run HPA e2e..."
	$(E2E) run -c test/e2e/oap-agent-adapter-hpa/e2e.yaml

.PHONY:e2e-oap-ui-agent-satellite
e2e-oap-ui-agent-satellite: e2e ## Run oap+ui+agent+satellite test
	@echo "Run oap+ui+agent+satellite e2e..."
	$(E2E) run -c test/e2e/oap-ui-agent-satellite/e2e.yaml

.PHONY:e2e-oap-agent-satellite-adapter-hpa
e2e-oap-agent-satellite-adapter-hpa: e2e ## Run oap+agent+satellite+adapter HPA test
	@echo "Run Satellite HPA e2e..."
	$(E2E) run -c test/e2e/oap-satellite-adapter-hpa/e2e.yaml

.PHONY:e2e-oap-ui-agent-oapserverconfig-oapserverdynamicconfig
e2e-oap-ui-agent-oapserverconfig-oapserverdynamicconfig: e2e ## Run e2e-oap-ui-agent-oapserverconfig-oapserverdynamicconfig test
	@echo "Run e2e-oap-ui-agent-oapserverconfig-oapserverdynamicconfig e2e..."
	$(E2E) run -c test/e2e/oap-ui-agent-oapserverconfig/e2e.yaml

E2E = $(tool_bin)/cmd
.PHONY: e2e
e2e: ## Download e2e-setup locally if necessary.
	$(call go-get-tool,$(E2E),github.com/apache/skywalking-infra-e2e/cmd@v1.1.0)

##@ Code quality and integrity

.PHONY: check
check: ## Check that the status
	$(MAKE) -C operator lint
	$(MAKE) -C operator dependency-resolve
	$(MAKE) -C operator check
	$(MAKE) -C adapter format
	$(MAKE) -C adapter lint
	$(MAKE) -C adapter dependency-resolve
	$(MAKE) -C adapter check
	$(MAKE) license-check


##@ release

RELEASE_SCRIPTS := ./build/package/release.sh

release-binary: ## Package binary archive
	$(MAKE) -C operator release-build
	$(MAKE) -C adapter release-build
	${RELEASE_SCRIPTS} -b

release-source: ## Package source archive
	${RELEASE_SCRIPTS} -s

release-sign: ## Sign artifacts
	${RELEASE_SCRIPTS} -k bin
	${RELEASE_SCRIPTS} -k src

release: release-binary release-source release-sign ## Generate release package