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

set -u
set -ex

OUT_DIR=$(mktemp -d -t operator-deploy.XXXXXXXXXX) || { echo "Failed to create temp file"; exit 1; }

SCRIPTPATH="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
ROOTDIR="${SCRIPTPATH}/.."

main() {
    [[ $1 -eq 0 ]] && frag="apply" || frag="delete --ignore-not-found=true"
    cp -Rvf "${ROOTDIR}"/config/operator/* "${OUT_DIR}"/.
    cd "${OUT_DIR}"/manager && kustomize edit set image controller=${OPERATOR_IMG}
    kustomize build "${OUT_DIR}"/default | kubectl ${frag} -f -
}

usage() {
cat <<EOF
Usage:
    ${0} -[duh]

Parameters:
    -d  Deploy operator
    -u  Undeploy operator
    -h  Show this help.
EOF
exit 1
}

parseCmdLine(){
    ARG=$1
    if [ $# -eq 0 ]; then
        echo "Exactly one argument required."
        usage
    fi
    case "${ARG}" in
        d) main 0;;
        u) main 1;;
        h) usage ;;
        \?) usage ;;
    esac
	  return 0
}

#
# main
#

ret=0

parseCmdLine "$@"
ret=$?
[ $ret -ne 0 ] && exit $ret
echo "Done deploy [$OPERATOR_IMG] (exit $ret)"