
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
IMAGE_FILE=test/e2e/e2e.yaml

prepare_ok=true
# install kubectl
function install_kubectl()
{
    if ! command -v kubectl &> /dev/null; then
      echo "please refer https://kubernetes.io/docs/tasks/tools/#install-kubectl to install kubectl!"
      $prepare_ok=false
    fi
}
# install swctl
function install_swctl()
{
    if ! command -v swctl &> /dev/null; then
      echo "please refer https://github.com/apache/skywalking-cli to install swctl!"
      $prepare_ok=false
    fi
}
# install yq
function install_yq()
{
    if ! command -v yq &> /dev/null; then
      echo "install yq..."
      wget https://github.com/mikefarah/yq/releases/download/v4.11.1/yq_${OS}_${ARCH}.tar.gz -O - |\
      tar xz && mv yq_${OS}_${ARCH} ${INSTALL_DIR}/yq
      if [ $? -ne 0 ]; then
        echo "install yq error, please check"
        $prepare_ok=false
      fi
    fi
}

# prepare images, please make sure you can pull these images
function install_images()
{
    yq e '.setup.kind.import-images' $IMAGE_FILE | awk -F ' ' '{print $2}' | xargs -I {} docker pull {}
    if [ $? -ne 0 ]; then
      echo "install images error, please check"
      $prepare_ok=false
    fi
}

function install_all()
{
    echo "check e2e dependencies..."
    install_e2e
    install_kind
    install_kubectl
    install_swctl
    install_yq
    install_images
    if [ "$prepare_ok" = false ]; then
        echo "check e2e dependencies failed"
        exit
    else
        echo "check e2e dependencies successfully"
    fi
}

install_all