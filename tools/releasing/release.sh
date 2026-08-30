#!/usr/bin/env bash

#
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

# Apache SkyWalking Cloud on Kubernetes release automation, part 1 of 2: everything up to the vote.
#
#   tools/releasing/release.sh         tag, build, sign, upload the candidate, draft the vote email
#   tools/releasing/release-passed.sh  after the vote passes: publish, announce, ship the binaries
#
# Modelled on apache/skywalking's tools/releasing/release.sh. Not to be confused with
# build/package/release.sh, which is the packaging helper `make release` calls -- this script drives
# the process around it.
#
# Usage: bash tools/releasing/release.sh
#
# See docs/en/guides/release.md for the manual equivalent of every step here.

set -e -o pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
PROJECT_DIR=$(cd "${SCRIPT_DIR}/../.." && pwd)
PRODUCT_NAME="skywalking-swck"
SVN_DEV_URL="https://dist.apache.org/repos/dist/dev/skywalking/swck"
CLONE_DIR="${SCRIPT_DIR}/skywalking-swck"
CHART_DIR_REL="chart/skywalking-swck"

# ========================== Shared functions ==========================

clone_repo() {
    cd "${SCRIPT_DIR}"
    if [ -d "${CLONE_DIR}" ]; then
        rm -rf "${CLONE_DIR}"
    fi
    echo "Cloning the repository..."
    git clone --depth 1 https://github.com/apache/skywalking-swck.git
}

# Everything that has to say the release version says it in one of three places. They are set here,
# in the release branch, so that the tag -- which is what the artifacts are built from -- carries
# them.
set_release_version() {
    cd "${CLONE_DIR}"

    echo "Setting the chart version to ${RELEASE_VERSION}..."
    # build/package/release.sh refuses to package a chart whose Chart.yaml disagrees with the
    # release version, and the publish workflow packages the registry copies from this same file.
    sed -i.bak -E "s/^version: .*/version: ${RELEASE_VERSION}/" "${CHART_DIR_REL}/Chart.yaml"
    sed -i.bak -E "s/^appVersion: .*/appVersion: \"${RELEASE_VERSION}\"/" "${CHART_DIR_REL}/Chart.yaml"
    rm -f "${CHART_DIR_REL}/Chart.yaml.bak"

    echo "Setting the image tags in the kustomize bundles to ${RELEASE_VERSION}..."
    # These are what `kubectl apply -f operator-bundle.yaml` from the binary tarball pulls, and
    # what `kubectl apply -k .../config/default` deploys straight from the repository.
    #
    # The same two commands run again inside build/package/release.sh when it builds the bundles.
    # Running them HERE, before the tag, is what makes that second call a no-op -- otherwise the
    # release build would modify the tree it had already tagged, and the source tarball would not
    # match the tag it is named after.
    make -C operator kustomize
    ( cd operator/config/manager && "${PWD%/operator/config/manager}/bin/kustomize" edit set image \
        "controller=apache/skywalking-swck:${RELEASE_VERSION}" )
    ( cd adapter/config/namespaced/adapter && "${PWD%/adapter/config/namespaced/adapter}/bin/kustomize" edit set image \
        "metrics-adapter=apache/skywalking-swck:${RELEASE_VERSION}" )

    grep -E '^(version|appVersion):' "${CHART_DIR_REL}/Chart.yaml"

    git add "${CHART_DIR_REL}/Chart.yaml" operator/config adapter/config
    # --allow-empty, for two reasons.
    #
    # The tree usually ALREADY carries the release version by the time a release is cut: the guide
    # tells you to set Chart.yaml before you start, and the kustomize image tags are set here
    # precisely so the same call inside `make release` is a no-op. `git commit` exits non-zero when
    # there is nothing staged, and this script runs under `set -e`, so the release died right here
    # with "nothing to commit, working tree clean" and no explanation.
    #
    # And the regeneration step below amends this commit. Without a commit of our own to amend, it
    # would rewrite whatever master's HEAD happened to be, folding generated files into somebody
    # else's commit and changing the tree the tag is about to point at.
    git commit --allow-empty -m "Prepare for release ${RELEASE_VERSION}"
}

build_release_artifacts() {
    local TAG=v${RELEASE_VERSION}

    cd "${CLONE_DIR}"

    echo "Checking out the release branch ${RELEASE_VERSION}-release..."
    git checkout -b "${RELEASE_VERSION}-release"

    set_release_version

    # Everything generated has to be in the tree BEFORE the tag, because the
    # source tarball is an archive of the working tree at build time and
    # `make release` runs the generators on the way through. Generating after
    # tagging would let the voted tarball differ from the tag it claims to be.
    echo "Regenerating everything the build would regenerate..."
    make -C operator generate
    make chart-manifests
    if [ -n "$(git status --porcelain)" ]; then
        echo "Generated files changed; folding them into the release commit:"
        git status --porcelain
        git add -A
        git commit --amend --no-edit
    fi
    if [ -n "$(git status --porcelain)" ]; then
        echo "ERROR: the tree is still dirty after regenerating:" >&2
        git status --porcelain >&2
        exit 1
    fi

    echo "Creating the release tag ${TAG}..."
    git tag "${TAG}"

    echo "Pushing the release tag ${TAG} to the remote repository..."
    # build/package/release.sh derives the version from the newest tag, so the tag has to exist and
    # be reachable before anything is built. That ordering means a build failure leaves a tag
    # behind on the remote, so the failure path below prints how to take it back.
    git push origin "${TAG}"

    log_file=$(mktemp)
    echo "Building and signing the release artifacts, log file: ${log_file}"
    echo "(you may be prompted for your GPG passphrase)"
    if ! make release > "${log_file}" 2>&1; then
        echo ""
        echo "ERROR: 'make release' failed. Last 40 lines of log:"
        tail -40 "${log_file}"
        echo ""
        echo "Full log: ${log_file}"
        echo ""
        echo "The tag ${TAG} is already on the remote, and re-running this script will fail to"
        echo "create it again. Take it back before retrying:"
        echo ""
        echo "    git push --delete origin ${TAG}"
        echo "    git tag --delete ${TAG}"
        exit 1
    fi

    # `make release` must not have changed anything: if it did, the tarball just
    # built no longer matches the tag it is named after.
    if [ -n "$(git status --porcelain)" ]; then
        echo "ERROR: 'make release' modified the tree, so the artifacts do not match ${TAG}:" >&2
        git status --porcelain >&2
        exit 1
    fi

    echo "Release artifacts built successfully."
}

verify_artifacts() {
    cd "${CLONE_DIR}/build/release"

    echo "Verifying the artifacts..."
    for f in "${SRC_TGZ}" "${BIN_TGZ}" "${CHART_TGZ}"; do
        for suffix in "" ".asc" ".sha512"; do
            if [ ! -f "${f}${suffix}" ]; then
                echo "ERROR: expected artifact not found: ${f}${suffix}"
                exit 1
            fi
        done
        shasum -a 512 -c "${f}.sha512"
        gpg --batch --verify "${f}.asc" "${f}"
    done

    # LICENSE and NOTICE have to be inside every tarball -- it is on the vote checklist, and it is
    # the one thing about these tarballs that a script can check better than a human.
    for f in "${SRC_TGZ}" "${BIN_TGZ}" "${CHART_TGZ}"; do
        for required in LICENSE NOTICE; do
            if ! tar tzf "${f}" | grep -qE "(^|/)${required}$"; then
                echo "ERROR: ${required} is missing from ${f}"
                exit 1
            fi
        done
    done

    # The chart tarball is only useful if it actually renders the operator.
    local rendered
    rendered=$(helm template swck "${CHART_TGZ}")
    if ! grep -q '^kind: Deployment$' <<<"${rendered}"; then
        echo "ERROR: ${CHART_TGZ} renders no Deployment"
        exit 1
    fi
    local crd_count
    crd_count=$(tar tzf "${CHART_TGZ}" | grep -c "${PRODUCT_NAME}/crds/.*\.yaml" || true)
    if [ "${crd_count}" -eq 0 ]; then
        echo "ERROR: ${CHART_TGZ} ships no CRDs; the operator it installs could not reconcile anything"
        exit 1
    fi
    echo "Chart renders a Deployment and ships ${crd_count} CRDs."

    echo "All artifacts verified:"
    ls -lh "${CLONE_DIR}"/build/release/
}

upload_to_svn() {
    local SVN_DIR="${SCRIPT_DIR}/svn-staging"
    local SVN_VERSION_DIR="${SVN_DIR}/${RELEASE_VERSION}"

    if [ -d "${SVN_DIR}" ]; then
        rm -rf "${SVN_DIR}"
    fi

    echo "Checking out the SVN dev directory..."
    svn co --depth immediates "${SVN_DEV_URL}" "${SVN_DIR}"

    if [ -d "${SVN_VERSION_DIR}" ]; then
        echo "Version folder ${RELEASE_VERSION} already exists in SVN, updating artifacts..."
        svn update "${SVN_VERSION_DIR}"
    else
        mkdir -p "${SVN_VERSION_DIR}"
        cd "${SVN_DIR}"
        svn add "${RELEASE_VERSION}"
    fi

    cd "${CLONE_DIR}/build/release"
    for f in "${SRC_TGZ}" "${BIN_TGZ}" "${CHART_TGZ}"; do
        cp "${f}" "${f}.asc" "${f}.sha512" "${SVN_VERSION_DIR}/"
    done

    cd "${SVN_DIR}"
    svn add --force "${RELEASE_VERSION}"
    svn commit -m "Draft Apache SkyWalking Cloud on Kubernetes release ${RELEASE_VERSION}"

    echo "Artifacts uploaded to: ${SVN_DEV_URL}/${RELEASE_VERSION}"

    rm -rf "${SVN_DIR}"
}

generate_vote_email() {
    local TAG=v${RELEASE_VERSION}
    local VOTE_DATE
    VOTE_DATE=$(date +"%B %d, %Y")

    cd "${CLONE_DIR}/build/release"
    local SRC_SHA512 BIN_SHA512 CHART_SHA512
    SRC_SHA512=$(cat "${SRC_TGZ}.sha512")
    BIN_SHA512=$(cat "${BIN_TGZ}.sha512")
    CHART_SHA512=$(cat "${CHART_TGZ}.sha512")

    cat <<EOF

========================================================================
Vote Email - Copy and send to dev@skywalking.apache.org
========================================================================

Mail title: [VOTE] Release Apache SkyWalking Cloud on Kubernetes version ${RELEASE_VERSION}

Mail content:
Hi All,
This is a call for vote to release Apache SkyWalking Cloud on Kubernetes version ${RELEASE_VERSION}.

Release notes:

 * https://github.com/apache/skywalking-swck/blob/${TAG}/docs/en/changes/changes.md

Release Candidate:

 * ${SVN_DEV_URL}/${RELEASE_VERSION}
 * sha512 checksums
   - ${SRC_SHA512}
   - ${BIN_SHA512}
   - ${CHART_SHA512}

Release Tag :

 * (Git Tag) ${TAG}

Release CommitID :

 * https://github.com/apache/skywalking-swck/tree/${TAG}

Keys to verify the Release Candidate :

 * https://dist.apache.org/repos/dist/release/skywalking/KEYS

Guide to build the release from source :

 * https://github.com/apache/skywalking-swck/blob/${TAG}/docs/en/guides/release.md

Voting will start now (${VOTE_DATE}) and will remain open for at least 72 hours, Request all PMC members to give their vote.
[ ] +1 Release this package.
[ ] +0 No opinion.
[ ] -1 Do not release this package because....

========================================================================

EOF
}

prepare_next_version() {
    cd "${CLONE_DIR}"

    echo "Setting the next version ${NEXT_RELEASE_VERSION} in the chart..."
    sed -i.bak -E "s/^version: .*/version: ${NEXT_RELEASE_VERSION}/" "${CHART_DIR_REL}/Chart.yaml"
    sed -i.bak -E "s/^appVersion: .*/appVersion: \"${NEXT_RELEASE_VERSION}\"/" "${CHART_DIR_REL}/Chart.yaml"
    rm -f "${CHART_DIR_REL}/Chart.yaml.bak"

    echo "Rotating the changelog..."
    mv docs/en/changes/changes.md "docs/en/changes/changes-${RELEASE_VERSION}.md"
    sed "s/NEXT_RELEASE_VERSION/${NEXT_RELEASE_VERSION}/g" docs/en/changes/changes.tpl > docs/en/changes/changes.md

    echo "Adding ${RELEASE_VERSION} to the docs menu..."
    # Inserted after "Current Version" rather than at the front, so the released version does not
    # displace the in-progress changelog at the top of the Changelog menu.
    yq -i '(.catalog[] | select(.name=="Changelog") | .catalog) |= [.[] | select(.name == "Current Version")] + [{ "name": "'"${RELEASE_VERSION}"'", "path": "/en/changes/changes-'"${RELEASE_VERSION}"'" }] + [.[] | select(.name != "Current Version")]' docs/menu.yml

    git add "${CHART_DIR_REL}/Chart.yaml" docs
    # Same reason as the release commit above: nothing staged is not a failure, and `set -e` would
    # otherwise abort the run after the vote candidate is already uploaded.
    git commit --allow-empty -m "Start the next iteration ${NEXT_RELEASE_VERSION}"

    echo "Pushing the changes to the remote repository..."
    git push --set-upstream origin "${RELEASE_VERSION}-release"

    echo "Creating the PR..."
    gh pr create --title "Prepare for next release ${NEXT_RELEASE_VERSION}" \
        --body "Bump the chart version to ${NEXT_RELEASE_VERSION} and rotate the changelog for the ${RELEASE_VERSION} release." \
        --base master
}

# ========================== Main flow ==========================

# Step 1: Preflight
# Everything this script needs, checked before it does anything irreversible -- the tools it shells
# out to, the signing key, gh, the dist URLs, the version, and whether the generated files are in
# sync. tools/releasing/preflight.sh is the same code, runnable on its own beforehand.
#
# It runs FIRST, before the signer prompt: being asked to confirm a key and only then told that
# svn is unreachable or the version is already tagged wastes the release manager's time.
# shellcheck source=tools/releasing/preflight.sh
. "${SCRIPT_DIR}/preflight.sh"
if ! run_preflight; then
    exit 1
fi

# Step 2: Confirm the signer, and prove it can actually sign
# The key itself was resolved and validated by the preflight -- including that it carries an
# @apache.org UID and appears in the published KEYS file. This does not repeat that work; it
# confirms the choice with a human and checks that gpg-agent will hand over the passphrase, which
# is worth finding out now rather than halfway through `make release`.
echo ""
echo "=== Step 2: Confirming the GPG signer ==="
GPG_KEY_ID="${PREFLIGHT_GPG_KEY_ID}"
export GPG_KEY_ID
echo "GPG Key:    ${GPG_KEY_ID}"
echo "GPG Signer: ${PREFLIGHT_GPG_UIDS}"
echo "GPG Email:  ${PREFLIGHT_GPG_EMAIL}"
read -r -p "Is this the correct GPG signer? [y/N] " confirm
if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
    echo "Aborted."
    exit 1
fi

GPG_TTY=$(tty)
export GPG_TTY
echo "Verifying GPG signing works (you may be prompted for your passphrase)..."
TEST_FILE=$(mktemp)
echo "test" > "${TEST_FILE}"
if ! gpg --local-user "${GPG_KEY_ID}" --armor --detach-sig "${TEST_FILE}" 2>/dev/null; then
    rm -f "${TEST_FILE}" "${TEST_FILE}.asc"
    echo ""
    echo "ERROR: GPG signing failed. Common fixes:"
    echo "  1. Run 'export GPG_TTY=\$(tty)' and retry"
    echo "  2. Start gpg-agent: 'gpgconf --launch gpg-agent'"
    echo "  3. Pre-cache passphrase: 'echo test | gpg --clearsign > /dev/null'"
    exit 1
fi
rm -f "${TEST_FILE}" "${TEST_FILE}.asc"
echo "GPG signing verified successfully."

# Step 3: Detect the current version
echo ""
echo "=== Step 3: Detecting the current version ==="

CURRENT_VERSION=$(awk '/^version:/{print $2; exit}' "${PROJECT_DIR}/${CHART_DIR_REL}/Chart.yaml")

if [ -z "$CURRENT_VERSION" ]; then
    echo "ERROR: could not read the version from ${CHART_DIR_REL}/Chart.yaml."
    exit 1
fi

echo "Current version in ${CHART_DIR_REL}/Chart.yaml: ${CURRENT_VERSION}"

# Step 4: Confirm the versions
echo ""
echo "=== Step 4: Confirming the versions ==="

RELEASE_VERSION="${CURRENT_VERSION}"
MAJOR=$(echo "$RELEASE_VERSION" | cut -d. -f1)
MINOR=$(echo "$RELEASE_VERSION" | cut -d. -f2)
NEXT_MINOR=$((MINOR + 1))
NEXT_RELEASE_VERSION="${MAJOR}.${NEXT_MINOR}.0"

echo "Release version:  ${RELEASE_VERSION}"
echo "Next dev version: ${NEXT_RELEASE_VERSION}"
read -r -p "Are these versions correct? [y/N] " confirm
if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
    read -r -p "Enter release version: " RELEASE_VERSION
    read -r -p "Enter next release version: " NEXT_RELEASE_VERSION
fi

# Validate before anything irreversible happens. TAG prepends its own `v`, and the version is
# stamped into the chart, the tarball names and the Docker tags, so a typo here is not caught
# anywhere downstream -- `helm package --version v0.11.0` is accepted, and by the time it shows up
# the tag has already been pushed to apache/skywalking-swck.
for v in "${RELEASE_VERSION}" "${NEXT_RELEASE_VERSION}"; do
    if [[ ! "${v}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        echo "ERROR: '${v}' is not a release version. Expected MAJOR.MINOR.PATCH with no leading 'v'." >&2
        exit 1
    fi
done

SRC_TGZ="${PRODUCT_NAME}-${RELEASE_VERSION}-src.tgz"
BIN_TGZ="${PRODUCT_NAME}-${RELEASE_VERSION}-bin.tgz"
# helm names the chart tarball from the chart name and version, so it has no -src/-bin suffix.
CHART_TGZ="${PRODUCT_NAME}-${RELEASE_VERSION}.tgz"

# Step 5: Clone and build
echo ""
echo "=== Step 5: Cloning the repository and building the release artifacts ==="
clone_repo
build_release_artifacts

# Step 6: Verify
echo ""
echo "=== Step 6: Verifying the artifacts ==="
verify_artifacts

# Step 7: Upload to SVN
echo ""
echo "=== Step 7: Uploading to SVN staging ==="
upload_to_svn

# Step 8: Vote email
echo ""
echo "=== Step 8: Generating the vote email ==="
generate_vote_email

# Step 9: Next version
echo ""
echo "=== Step 9: Starting the next version iteration ==="
prepare_next_version

cat <<EOF

=========================================================================
Release candidate ${RELEASE_VERSION} is staged.
  Release version:  ${RELEASE_VERSION}
  Next dev version: ${NEXT_RELEASE_VERSION}
  SVN staging:      ${SVN_DEV_URL}/${RELEASE_VERSION}
  Git tag:          v${RELEASE_VERSION}

Next steps:
  1. Send the vote email above to dev@skywalking.apache.org
  2. Merge the next-version PR
  3. Once the vote passes, run: bash tools/releasing/release-passed.sh

No image and no chart has been published yet. They are convenience binaries and
release-passed.sh is what publishes them, after the vote.
=========================================================================
EOF
