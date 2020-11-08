#!/usr/bin/env bash

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

set -ex
SCRIPTDIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
BUILDDIR=${SCRIPTDIR}/..
ROOTDIR=${BUILDDIR}/..

RELEASE_TAG=$(git describe --tags $(git rev-list --tags --max-count=1))
RELEASE_VERSION=${RELEASE_TAG#"v"}

binary(){
    bindir=${BUILDDIR}/release/binary
    rm -rf ${bindir}
    mkdir -p ${bindir}/config
    # Copy relevant files
    cp -Rfv ${BUILDDIR}/bin ${bindir}
    cp -Rfv ${ROOTDIR}/CHANGES.md ${bindir}
    cp -Rfv ${ROOTDIR}/README.md ${bindir}
    cp -Rfv ${ROOTDIR}/dist/* ${bindir}
    # Generates CRDs and deployment manifests
    kustomize build config/crd > ${bindir}/config/crds.yaml
    pushd ${ROOTDIR}/config/manager
    kustomize edit set image controller=apache/skywalking-swck:${RELEASE_VERSION}
    popd
    kustomize build config/default > ${bindir}/config/deploy.yaml
    # Package
    tar -czf ${BUILDDIR}/release/skywalking-swck-${RELEASE_VERSION}-bin.tgz -C ${bindir} .
}

source(){
    # Package
    rm -rf ${BUILDDIR}/release/skywalking-swck-${RELEASE_VERSION}-src.tgz
    pushd ${ROOTDIR}
    tar \
        --exclude="bin"  \
        --exclude="build"  \
        --exclude=".*" \
        --exclude="*.test"  \
        --exclude="*.out"  \
        -czf ${BUILDDIR}/release/skywalking-swck-${RELEASE_VERSION}-src.tgz \
        .
    popd
}

parseCmdLine(){
    ARGS=$1
    if [ $# -eq 0 ]; then
        echo "Exactly one argument required."
        usage
    fi
    while getopts  "bsh" FLAG; do
        case "${FLAG}" in
            b) binary ;;
            s) source ;;
            h) usage ;;
            \?) usage ;;
        esac
    done
	return 0
}



usage() {
cat <<EOF
Usage:
    ${0} -[bsh]

Parameters:
    -b  Build and assemble the binary package
    -s  Assemble the source package
    -h  Show this help.
EOF
exit 1
}

#
# main
#

ret=0

parseCmdLine "$@"
ret=$?
[ $ret -ne 0 ] && exit $ret
echo "Done release [$RELEASE_TAG] (exit $ret)"

