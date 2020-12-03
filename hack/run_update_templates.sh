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
#

set -ue

GOBINDATA="${LOCAL_GOBINDATA-go-bindata}"

OUT_DIR=$(mktemp -d -t swck-templates.XXXXXXXXXX) || { echo "Failed to create temp file"; exit 1; }

SCRIPTPATH="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
ROOTDIR="${SCRIPTPATH}/.."

OPERATOR_DIR="${ROOTDIR}"/pkg/operator
INSTALLER_DIR="${OPERATOR_DIR}"/manifests
GEN_PATH="${OPERATOR_DIR}"/repo/assets.gen.go

rm -f "${GEN_PATH}"
mkdir -p "${OUT_DIR}"

for comp in $(ls ${INSTALLER_DIR});
do
    cp -Rf "${INSTALLER_DIR}"/"${comp}" "${OUT_DIR}"
done

set +u

cd "${OUT_DIR}"
set +e
rm -f *.bak
set -e
"${GOBINDATA}" --nocompress --nometadata --pkg repo -o "${GEN_PATH}" ./...

rm -Rf "${OUT_DIR}"
