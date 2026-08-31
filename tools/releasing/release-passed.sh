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

# Apache SkyWalking Cloud on Kubernetes release automation, part 2 of 2: everything after the vote
# passes. Run tools/releasing/release.sh first.
#
# Usage: bash tools/releasing/release-passed.sh [version]
#
# What it does, in the order the ASF release policy requires:
#
#   1. moves the voted artifacts from dist/dev to dist/release  -- this is the act of releasing
#   2. removes the previous release from dist/release           -- only the current release is mirrored
#   3. waits for the artifacts to reach archive.apache.org      -- the image build downloads them
#   4. publishes the GitHub release                             -- which triggers publish-docker.yml
#   5. waits for that workflow and verifies what it pushed
#   6. prints the announcement emails
#
# The images and the chart in the registries are convenience binaries, not the release. They are
# built from the voted binary tarball and must not exist before step 1.

set -e -o pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
PROJECT_DIR=$(cd "${SCRIPT_DIR}/../.." && pwd)
PRODUCT_NAME="skywalking-swck"
REPO="apache/skywalking-swck"
SVN_DEV_URL="https://dist.apache.org/repos/dist/dev/skywalking/swck"
SVN_RELEASE_URL="https://dist.apache.org/repos/dist/release/skywalking/swck"
DOWNLOAD_URL="https://downloads.apache.org/skywalking/swck"

# ========================== Shared functions ==========================

confirm() {
    local prompt="$1"
    read -r -p "${prompt} [y/N] " reply
    if [[ "$reply" != "y" && "$reply" != "Y" ]]; then
        echo "Aborted."
        exit 1
    fi
}

move_to_release() {
    # Check first, ask second. Asking whether the vote passed and only then discovering the move
    # already happened makes a resume look like it is about to redo something irreversible.
    if svn ls "${SVN_RELEASE_URL}/${RELEASE_VERSION}" >/dev/null 2>&1; then
        echo "${SVN_RELEASE_URL}/${RELEASE_VERSION} already exists, skipping the move."
        return
    fi

    echo "Moving ${SVN_DEV_URL}/${RELEASE_VERSION} to ${SVN_RELEASE_URL}/..."
    echo "You need to be a PMC member to do this, and you will be asked for your Apache password."
    confirm "Has the vote passed with at least 3 binding +1 and more +1 than -1?"

    svn mv -m "Release Apache SkyWalking Cloud on Kubernetes ${RELEASE_VERSION}" \
        "${SVN_DEV_URL}/${RELEASE_VERSION}" "${SVN_RELEASE_URL}/${RELEASE_VERSION}"
    echo "Released to ${SVN_RELEASE_URL}/${RELEASE_VERSION}"
}

# dist/release is mirrored to every ASF mirror, so it carries only the current release; older ones
# live on archive.apache.org, which is exactly where Dockerfile.release reads from.
remove_previous_release() {
    local previous
    previous=$(svn ls "${SVN_RELEASE_URL}" 2>/dev/null | tr -d '/' | grep -v "^${RELEASE_VERSION}$" || true)
    if [ -z "${previous}" ]; then
        echo "No previous release to remove."
        return
    fi
    echo "Previous release(s) still in dist/release:"
    echo "${previous}"
    confirm "Remove them? (they stay available on archive.apache.org)"
    for version in ${previous}; do
        svn rm -m "Remove Apache SkyWalking Cloud on Kubernetes ${version}, superseded by ${RELEASE_VERSION}" \
            "${SVN_RELEASE_URL}/${version}"
    done
}

# build/images/Dockerfile.release downloads the binary tarball from archive.apache.org and verifies
# its signature, so the image cannot be built until the archive has it. The publish workflow waits
# too, but failing here costs a minute instead of a workflow run.
# The move committed to dist/release. That is the release, and it is what the image build reads,
# so there is nothing else to confirm.
verify_published() {
    local svn_url="${SVN_RELEASE_URL}/${RELEASE_VERSION}"
    if ! svn ls "${svn_url}" >/dev/null 2>&1; then
        echo "ERROR: ${svn_url} does not exist -- the move to dist/release did not take effect."
        exit 1
    fi
    echo "Published to ${svn_url}"
}

# Publishing the GitHub release is what triggers .github/workflows/publish-docker.yml, which builds
# and pushes the two GHCR images, the combined Docker Hub image and both copies of the chart.
publish_github_release() {
    local tag="v${RELEASE_VERSION}"

    # A release that exists is not necessarily a release that is published: `gh release view`
    # resolves drafts too. Treating a draft as "already done" returned success here while leaving
    # the release unpublished, so the publish workflow -- which triggers on publication -- never
    # ran and the next step went looking for convenience binaries nobody had built.
    if state=$(gh release view "${tag}" --repo "${REPO}" \
        --json isDraft,publishedAt --jq '[.isDraft, (.publishedAt // "")] | @tsv' 2>/dev/null); then
        IFS=$'\t' read -r is_draft published_at <<< "${state}"
        if [ "${is_draft}" = "false" ] && [ -n "${published_at}" ]; then
            echo "GitHub release ${tag} is already published."
            return
        fi
        echo "GitHub release ${tag} exists as a draft; publishing it..."
        gh release edit "${tag}" --repo "${REPO}" --draft=false
        return
    fi

    echo "Creating the GitHub release ${tag}..."
    local notes_file
    notes_file=$(mktemp)

    # The release page carries the changelog itself, not a link to it. A link is worse than it
    # looks: by the time anyone follows it, the next-version PR has moved that section into
    # docs/en/changes/changes-<version>.md and put an empty template at changes.md, so the link
    # lands on the NEXT version's empty page.
    #
    # Read it from the tag rather than the working tree, for the same reason and because the tag is
    # what the PMC actually voted on. The leading "## <version>" heading is dropped: GitHub already
    # shows the version as the release title.
    local changelog
    changelog=$(git -C "${PROJECT_DIR}" show "${tag}:docs/en/changes/changes.md" 2>/dev/null \
        | sed '1{/^## /d;}')
    if [ -z "${changelog}" ]; then
        echo "WARNING: could not read the changelog from ${tag}; falling back to a link." >&2
        changelog="Changes: https://github.com/${REPO}/blob/${tag}/docs/en/changes/changes.md"
    fi

    cat > "${notes_file}" <<EOF
Apache SkyWalking Cloud on Kubernetes ${RELEASE_VERSION}.

${changelog}

---

Source and binary distributions, and the Helm chart, are on the ASF mirrors:
${DOWNLOAD_URL}/${RELEASE_VERSION}

Docker images and the Helm chart are convenience binaries; the official release is the source
tarball above.
EOF
    # --verify-tag or gh creates the tag itself, at whatever master points to now. That would
    # publish convenience binaries built from code the PMC never voted on.
    gh release create "${tag}" --repo "${REPO}" --verify-tag \
        --title "${RELEASE_VERSION}" --notes-file "${notes_file}"
    rm -f "${notes_file}"
    echo "Published. .github/workflows/publish-docker.yml is now building the convenience binaries."
}

wait_for_publish_workflow() {
    local tag="v${RELEASE_VERSION}"
    local head_sha
    # Ask the remote, not the working copy. release.sh pushes this tag from a throwaway clone it
    # wipes on the next run, and this script is meant to be run from the release manager's own
    # checkout days later -- which has never seen the tag. Resolving it locally yielded an empty
    # sha, and the poll below then silently skipped every query and reported "could not find a
    # publish run" on the happy path.
    head_sha=$(gh api "repos/${REPO}/git/ref/tags/${tag}" --jq '.object.sha' 2>/dev/null || true)
    if [ -z "${head_sha}" ]; then
        echo "ERROR: ${tag} does not exist on ${REPO}." >&2
        echo "The release tag is created by tools/releasing/release.sh before the vote." >&2
        exit 1
    fi
    # An annotated tag's ref points at the tag object; the run is keyed on the commit.
    local object_type
    object_type=$(gh api "repos/${REPO}/git/ref/tags/${tag}" --jq '.object.type' 2>/dev/null || echo commit)
    if [ "${object_type}" = "tag" ]; then
        head_sha=$(gh api "repos/${REPO}/git/tags/${head_sha}" --jq '.object.sha' 2>/dev/null || echo "${head_sha}")
    fi

    echo "Waiting for the publish workflow for ${tag}..."
    # Correlate on the commit the release tag points at rather than taking the
    # newest run. A master push, or someone else's dispatch, would otherwise be
    # watched instead -- and its result reported as this release's. GitHub does
    # not register the run instantly either, so poll rather than sleep once.
    local run_id=""
    for _ in $(seq 1 30); do
        run_id=$(gh run list --repo "${REPO}" --workflow publish-docker.yml --limit 30 \
            --json databaseId,headSha \
            --jq "[.[] | select(.headSha == \"${head_sha}\")] | .[0].databaseId" 2>/dev/null || true)
        if [ -n "${run_id}" ] && [ "${run_id}" != "null" ]; then
            break
        fi
        sleep 10
    done
    if [ -z "${run_id}" ] || [ "${run_id}" = "null" ]; then
        # Stop rather than return: the next step verifies the images this workflow publishes, and
        # reporting them as missing is a far more confusing failure than saying the run never
        # appeared.
        echo "ERROR: no publish run for ${tag} (${head_sha}) after 5 minutes." >&2
        echo "Check https://github.com/${REPO}/actions and re-run this script once it has finished." >&2
        exit 1
    fi
    echo "Watching run ${run_id}: https://github.com/${REPO}/actions/runs/${run_id}"
    gh run watch "${run_id}" --repo "${REPO}" --exit-status || {
        echo ""
        echo "ERROR: the publish workflow failed."
        echo "Fix the cause and re-run it with:"
        echo "  gh workflow run publish-docker.yml --repo ${REPO} --ref master -f tag=v${RELEASE_VERSION}"
        exit 1
    }
}

verify_published_artifacts() {
    local failed=0
    echo ""
    echo "Verifying the published artifacts..."

    for image in \
        "ghcr.io/apache/skywalking-swck/operator:${RELEASE_VERSION}" \
        "ghcr.io/apache/skywalking-swck/metrics-adapter:${RELEASE_VERSION}" \
        "docker.io/apache/skywalking-swck:${RELEASE_VERSION}"; do
        if docker buildx imagetools inspect "${image}" >/dev/null 2>&1; then
            echo "  ok   ${image}"
        else
            echo "  FAIL ${image}"
            failed=1
        fi
    done

    # A chart pushed over an image tag replaces the manifest and `docker pull` stops working. The
    # workflow checks this too; check it again from outside, with a plain pull.
    if docker pull -q "docker.io/apache/skywalking-swck:${RELEASE_VERSION}" >/dev/null 2>&1; then
        echo "  ok   docker pull apache/skywalking-swck:${RELEASE_VERSION}"
    else
        echo "  FAIL docker pull apache/skywalking-swck:${RELEASE_VERSION} -- did a chart push clobber the image tag?"
        failed=1
    fi

    if helm show chart "oci://ghcr.io/apache/skywalking-swck/helm/${PRODUCT_NAME}" \
            --version "${RELEASE_VERSION}" >/dev/null 2>&1; then
        echo "  ok   oci://ghcr.io/apache/skywalking-swck/helm/${PRODUCT_NAME} --version ${RELEASE_VERSION}"
    else
        echo "  FAIL oci://ghcr.io/apache/skywalking-swck/helm/${PRODUCT_NAME} --version ${RELEASE_VERSION}"
        failed=1
    fi

    # On Docker Hub the chart shares a repository with the image, so it carries a -helm suffix. It
    # is a SemVer pre-release and only an exact --version resolves it.
    if helm show chart "oci://registry-1.docker.io/apache/${PRODUCT_NAME}" \
            --version "${RELEASE_VERSION}-helm" >/dev/null 2>&1; then
        echo "  ok   oci://registry-1.docker.io/apache/${PRODUCT_NAME} --version ${RELEASE_VERSION}-helm"
    else
        echo "  FAIL oci://registry-1.docker.io/apache/${PRODUCT_NAME} --version ${RELEASE_VERSION}-helm"
        failed=1
    fi

    if [ "${failed}" -ne 0 ]; then
        echo ""
        echo "ERROR: some artifacts are missing. See docs/en/guides/release.md for how to push them by hand."
        exit 1
    fi
    echo "All five release artifacts are published."
}

generate_announcement() {
    cat <<EOF

========================================================================
Vote result email - Copy and send to dev@skywalking.apache.org
========================================================================

Mail title: [RESULT][VOTE] Release Apache SkyWalking Cloud on Kubernetes version ${RELEASE_VERSION}

Mail content:
Hi All,

3 days passed, we've got (N) +1 bindings:
(list the names of the binding voters)

and (N) +1 non-bindings:
(list the names of the non-binding voters)

I'll continue the release process.

========================================================================
Announcement email - Copy and send to announce@apache.org and dev@skywalking.apache.org
========================================================================

Mail title: [ANNOUNCE] Apache SkyWalking Cloud on Kubernetes ${RELEASE_VERSION} released

Mail content:
Hi the SkyWalking Community,

On behalf of the SkyWalking Team, I'm glad to announce that SkyWalking Cloud on
Kubernetes ${RELEASE_VERSION} is now released.

SkyWalking Cloud on Kubernetes (SWCK) is a platform for the SkyWalking user that
provisions, upgrades, maintains SkyWalking relevant components, and makes them
work natively on Kubernetes.

Download Links: https://skywalking.apache.org/downloads/

Release Notes: https://github.com/${REPO}/blob/v${RELEASE_VERSION}/docs/en/changes/changes.md

Website: https://skywalking.apache.org/

SkyWalking Cloud on Kubernetes Resources:
- Issue: https://github.com/${REPO}/issues
- Mailing list: dev@skywalking.apache.org
- Documents: https://skywalking.apache.org/docs/skywalking-swck/latest/readme/

The Apache SkyWalking Team

========================================================================

Remaining manual steps:
  1. Send both emails above.
  2. Update the website: add ${RELEASE_VERSION} to data/docs.yml and data/releases.yml in
     apache/skywalking-website, following a previous release PR.
  3. Update the GitHub release page if you want richer notes than the generated ones.
========================================================================

EOF
}

# ========================== Main flow ==========================

RELEASE_VERSION="${1:-}"
if [ -z "${RELEASE_VERSION}" ]; then
    # Ask dist/dev what is waiting to be published, rather than the working tree.
    #
    # Chart.yaml is the wrong source here: by the time the vote passes, release.sh has already
    # opened the next-version PR and it has usually been merged, so Chart.yaml holds the version
    # AFTER the one being released. Defaulting to it meant pressing Enter tried to publish a
    # candidate that does not exist -- which failed, but only several prompts later and with an
    # svn path error rather than an explanation.
    CANDIDATES=$(svn ls "${SVN_DEV_URL}" 2>/dev/null | tr -d '/' | grep -E '^[0-9]+\.[0-9]+\.[0-9]+$' || true)
    CANDIDATE_COUNT=$(echo "${CANDIDATES}" | grep -c . || true)

    if [ "${CANDIDATE_COUNT}" -eq 1 ]; then
        RELEASE_VERSION="${CANDIDATES}"
        echo "dist/dev holds one candidate: ${RELEASE_VERSION}"
    elif [ "${CANDIDATE_COUNT}" -gt 1 ]; then
        echo "ERROR: ${SVN_DEV_URL} holds more than one candidate:"
        echo "${CANDIDATES}" | sed 's/^/  /'
        echo "Pass the one you mean: bash $(basename "$0") <version>"
        exit 1
    else
        # dist/dev is empty, which is what a RESUME looks like: the move already happened and
        # everything after it -- publishing the GitHub release, the images, the chart -- has not.
        # Fall back to the newest thing in dist/release so re-running picks up where it stopped
        # instead of refusing to start.
        RELEASE_VERSION=$(svn ls "${SVN_RELEASE_URL}" 2>/dev/null | tr -d '/' \
            | grep -E '^[0-9]+\.[0-9]+\.[0-9]+$' | sort -V | tail -1 || true)
        if [ -z "${RELEASE_VERSION}" ]; then
            echo "ERROR: neither ${SVN_DEV_URL} nor ${SVN_RELEASE_URL} holds a release."
            exit 1
        fi
        echo "dist/dev is empty and dist/release holds ${RELEASE_VERSION}."
        echo "Continuing a release that was already moved; the steps below skip what is done."
    fi

    read -r -p "Version to publish [${RELEASE_VERSION}]: " answer
    RELEASE_VERSION="${answer:-${RELEASE_VERSION}}"
fi

if [[ ! "${RELEASE_VERSION}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "ERROR: '${RELEASE_VERSION}' is not a release version like 0.11.0"
    exit 1
fi

# Fail here, before the vote question and before anything is moved, rather than partway through
# with an svn path error.
if ! svn ls "${SVN_DEV_URL}/${RELEASE_VERSION}" >/dev/null 2>&1 \
    && ! svn ls "${SVN_RELEASE_URL}/${RELEASE_VERSION}" >/dev/null 2>&1; then
    echo "ERROR: ${RELEASE_VERSION} is in neither dist/dev nor dist/release."
    echo "  dist/dev:     $(svn ls "${SVN_DEV_URL}" 2>/dev/null | tr -d '/' | tr '\n' ' ')"
    echo "  dist/release: $(svn ls "${SVN_RELEASE_URL}" 2>/dev/null | tr -d '/' | tr '\n' ' ')"
    exit 1
fi

echo "=== Step 1: Checking required tools ==="
MISSING_TOOLS=()
for tool in svn curl gh docker helm; do
    if ! command -v "$tool" &>/dev/null; then
        MISSING_TOOLS+=("$tool")
    fi
done
if [ ${#MISSING_TOOLS[@]} -gt 0 ]; then
    echo "ERROR: Missing required tools: ${MISSING_TOOLS[*]}"
    exit 1
fi
echo "All required tools are available."

echo ""
echo "=== Step 2: Publishing the voted artifacts to dist/release ==="
move_to_release

echo ""
echo "=== Step 3: Removing the previous release from dist/release ==="
remove_previous_release

echo ""
echo "=== Step 4: Confirming the release is published ==="
verify_published

echo ""
echo "=== Step 5: Publishing the GitHub release ==="
publish_github_release

echo ""
echo "=== Step 6: Waiting for the publish workflow ==="
wait_for_publish_workflow

echo ""
echo "=== Step 7: Verifying the published artifacts ==="
verify_published_artifacts

echo ""
echo "=== Step 8: Announcement ==="
generate_announcement
