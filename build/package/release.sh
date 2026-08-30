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
    # Prefer the version-pinned kustomize the Makefile installs (tools/build/module.mk) over
    # whatever happens to be on the release manager's PATH. Only the binary package needs it, so
    # it is resolved here rather than at the top of the script.
    KUSTOMIZE=${ROOTDIR}/bin/kustomize
    if [ ! -x "${KUSTOMIZE}" ]; then
        KUSTOMIZE=$(command -v kustomize) || { echo "kustomize not found; run 'make -C operator kustomize'" >&2; exit 1; }
    fi

    bindir=${BUILDDIR}/release/binary
    rm -rf ${bindir}
    mkdir -p ${bindir}/config
    # Copy relevant files
    cp -Rfv ${BUILDDIR}/bin ${bindir}
    cp -Rfv ${ROOTDIR}/docs/en/changes/changes.md ${bindir}/CHANGES.md
    cp -Rfv ${ROOTDIR}/docs/binary-readme.md ${bindir}/README.md
    cp -Rfv ${ROOTDIR}/dist/* ${bindir}
    cp -Rfv ${ROOTDIR}/operator/dist/* ${bindir}
    mkdir -p ${bindir}/licenses/adapter-licenses
    cp -Rfv ${ROOTDIR}/adapter/dist/licenses/* ${bindir}/licenses/adapter-licenses
	cat ${ROOTDIR}/adapter/dist/LICENSE >> ${bindir}/LICENSE
    # Docker
    cp -Rfv ${ROOTDIR}/build/images/Dockerfile.release-bin ${bindir}/Dockerfile
    echo -e "build:" > ${bindir}/Makefile
    echo -e "\tdocker build . -t apache/skywalking-swck:${RELEASE_VERSION}" >> ${bindir}/Makefile
    # Generates CRDs and deployment manifests
    pushd ${ROOTDIR}/operator/config/manager
    ${KUSTOMIZE} edit set image controller=apache/skywalking-swck:${RELEASE_VERSION}
    popd
    ${KUSTOMIZE} build operator/config/default > ${bindir}/config/operator-bundle.yaml
    pushd ${ROOTDIR}/adapter/config/namespaced/adapter
    ${KUSTOMIZE} edit set image metrics-adapter=apache/skywalking-swck:${RELEASE_VERSION}
    popd
    ${KUSTOMIZE} build adapter/config > ${bindir}/config/adapter-bundle.yaml
    # Package
    tar -czf ${BUILDDIR}/release/skywalking-swck-${RELEASE_VERSION}-bin.tgz -C ${bindir} .
    rm -rf ${bindir}
}

source(){
    # Package
    rm -rf ${BUILDDIR}/release/skywalking-swck-${RELEASE_VERSION}-src.tgz
    pushd ${ROOTDIR}
    tar \
        --exclude=".DS_Store" \
        --exclude=".git" \
        --exclude=".github" \
        --exclude=".gitignore" \
        --exclude=".asf.yaml" \
        --exclude=".idea" \
        --exclude="bin"  \
        --exclude="operator/bin"  \
        --exclude="adapter/bin"  \
        --exclude="build/release"  \
        --exclude="*.test"  \
        --exclude="*.out"  \
        -czf ./build/release/skywalking-swck-${RELEASE_VERSION}-src.tgz \
        .
    popd
}

# The Helm chart is a release artifact like the source and binary tarballs: it is
# signed, uploaded to dist.apache.org and voted on. The copies pushed to the two
# registries after the vote are packaged again by
# .github/workflows/publish-docker.yml and differ from this one only in the
# version helm stamps into Chart.yaml.
chart(){
    chartdir=${ROOTDIR}/chart/skywalking-swck
    if ! command -v helm > /dev/null; then
        echo "helm is required to package the chart, see https://helm.sh/docs/intro/install/" >&2
        exit 1
    fi
    # The chart version is bumped by hand as part of preparing the release
    # (docs/en/guides/release.md). Catch a forgotten bump here rather than shipping a chart
    # whose Chart.yaml disagrees with the tarball it is in.
    chart_version=$(awk '/^version:/{print $2; exit}' ${chartdir}/Chart.yaml)
    if [ "${chart_version}" != "${RELEASE_VERSION}" ]; then
        echo "chart/skywalking-swck/Chart.yaml says version ${chart_version}, but this is release ${RELEASE_VERSION}" >&2
        exit 1
    fi
    # LICENSE and NOTICE have to be INSIDE the tarball that gets voted on.
    cp -f ${ROOTDIR}/LICENSE ${ROOTDIR}/NOTICE ${chartdir}/
    trap "rm -f ${chartdir}/LICENSE ${chartdir}/NOTICE" EXIT
    helm dependency update ${chartdir}
    helm package ${chartdir} \
        --version ${RELEASE_VERSION} \
        --app-version ${RELEASE_VERSION} \
        --destination ${BUILDDIR}/release
    rm -f ${chartdir}/LICENSE ${chartdir}/NOTICE
    trap - EXIT
}

sign(){
    type=$1
    # The chart tarball is named skywalking-swck-$VERSION.tgz, with no -bin/-src
    # suffix, because helm derives that name from the chart name and version.
    case "${type}" in
        chart) file=skywalking-swck-${RELEASE_VERSION}.tgz ;;
        *)     file=skywalking-swck-${RELEASE_VERSION}-${type}.tgz ;;
    esac
    # Sign with the key the release manager chose, not gpg's default. A machine
    # with several secret keys would otherwise sign with whichever gpg picks,
    # and an artifact signed by a key that is not in the project KEYS file fails
    # every voter's `gpg --verify`. tools/releasing/release.sh exports this after
    # checking the key's own UID is an @apache.org address.
    local_user=""
    if [ -n "${GPG_KEY_ID:-}" ]; then
        local_user="--local-user ${GPG_KEY_ID}"
    fi
    pushd ${BUILDDIR}/release/
    gpg ${local_user} --batch --yes --armor --detach-sig ${file}
	shasum -a 512 ${file} > ${file}.sha512
	popd
}

parseCmdLine(){
    ARGS=$1
    if [ $# -eq 0 ]; then
        echo "Exactly one argument required."
        usage
    fi
    while getopts  "bsck:h" FLAG; do
        case "${FLAG}" in
            b) binary ;;
            s) source ;;
            c) chart ;;
            k) sign ${OPTARG} ;;
            h) usage ;;
            \?) usage ;;
        esac
    done
	return 0
}



usage() {
cat <<EOF
Usage:
    ${0} -[bsch]

Parameters:
    -b  Build and assemble the binary package
    -s  Assemble the source package
    -c  Package the Helm chart
    -k  Sign an artifact: -k bin, -k src or -k chart
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
echo "Done release [RELEASE_VERSION] (exit $ret)"
