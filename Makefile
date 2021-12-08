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

.PHONY: docker-build
docker-build: ## Build docker images
	$(MAKE) -C operator docker-build

.PHONY: test
test: ## Run unit test cases
	$(MAKE) -C operator test

##@ End to End Test

.PHONY:e2e-test
e2e-test: e2e-oap-ui-agent e2e-oap-ui-agent-storage-internal## Run End to End tests.

.PHONY:e2e-oap-ui-agent
e2e-oap-ui-agent: e2e ## Run oap+ui+agent test
	@echo "Run oap+ui+agent e2e..."
	$(E2E) run -c test/e2e/oap-ui-agent/e2e.yaml

.PHONY:e2e-oap-ui-agent-storage-internal
e2e-oap-ui-agent-internal-storage: e2e ## Run oap+ui+agent test
	@echo "Run oap+ui+agent e2e..."
	$(E2E) run -c test/e2e/oap-ui-agent-internal-storage/e2e.yaml 

E2E = $(tool_bin)/cmd
.PHONY: e2e
e2e: ## Download e2e-setup locally if necessary.
	$(call go-get-tool,$(E2E),github.com/apache/skywalking-infra-e2e/cmd@v1.1.0)

##@ Code quality and integrity

.PHONY: check
check: ## Check that the status
	$(MAKE) -C operator generate
	$(MAKE) -C operator lint
	$(MAKE) -C operator check
	$(MAKE) license-check
