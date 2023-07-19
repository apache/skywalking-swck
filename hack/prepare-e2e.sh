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

OS=$(go env GOOS)
ARCH=$(go env GOHOSTARCH)

INSTALL_DIR=/usr/local/bin

KUBECTL_VERSION=v1.21.10
SWCTL_VERSION=0.12.0
YQ_VERSION=v4.11.1

prepare_ok=true
# install kubectl
function install_kubectl()
{
    if ! command -v kubectl &> /dev/null; then
      curl -LO https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/${OS}/${ARCH}/kubectl && chmod +x ./kubectl && mv ./kubectl ${INSTALL_DIR}
      if [ $? -ne 0 ]; then
        echo "install kubectl error, please check"
        prepare_ok=false
      fi
    fi
}
# install swctl
function install_swctl()
{
    if ! command -v swctl &> /dev/null; then
      wget https://github.com/apache/skywalking-cli/archive/${SWCTL_VERSION}.tar.gz -O - |\
      tar xz && cd skywalking-cli-${SWCTL_VERSION} && make install DESTDIR=${INSTALL_DIR}  \
      && cd .. && rm -r skywalking-cli-${SWCTL_VERSION}
      if [ $? -ne 0 ]; then
        echo "install swctl error, please check"
        prepare_ok=false
      fi
    fi
}
# install yq
function install_yq()
{
    if ! command -v yq &> /dev/null; then
      echo "install yq..."
      wget https://github.com/mikefarah/yq/releases/download/${YQ_VERSION}/yq_${OS}_${ARCH}.tar.gz -O - |\
      tar xz && mv yq_${OS}_${ARCH} ${INSTALL_DIR}/yq
      if [ $? -ne 0 ]; then
        echo "install yq error, please check"
        prepare_ok=false
      fi
    fi
}

function install_all()
{
    echo "check e2e dependencies..."
    install_kubectl
    install_swctl
    install_yq
    if [ "$prepare_ok" = false ]; then
        echo "check e2e dependencies failed"
        exit
    else
        echo "check e2e dependencies successfully"
    fi
}

install_all