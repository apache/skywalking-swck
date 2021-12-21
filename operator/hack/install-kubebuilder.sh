#!/usr/bin/env bash

#
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
set -ex

os=$(go env GOOS)
arch=$(go env GOARCH)
export K8S_VERSION=1.19.2
export PATH=$PATH:/usr/local/kubebuilder/bin
sudo mkdir -p /usr/local/kubebuilder/bin
curl -sSLo kubebuilder https://github.com/kubernetes-sigs/kubebuilder/releases/download/v3.2.0/kubebuilder_${os}_${arch}
sudo mv ./kubebuilder /usr/local/bin/

curl -sSLo envtest-bins.tar.gz "https://go.kubebuilder.io/test-tools/${K8S_VERSION}/${os}/${arch}"
sudo  tar -C /usr/local/kubebuilder --strip-components=1 -zvxf envtest-bins.tar.gz