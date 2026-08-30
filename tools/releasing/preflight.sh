#!/usr/bin/env bash
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

# Everything release.sh needs, checked before it starts.
#
# release.sh pushes a tag partway through, so a prerequisite discovered late is not merely
# inconvenient: recovering means deleting a tag from the remote. This runs the same checks first,
# reports all of them rather than stopping at the first, and exits non-zero if any BLOCK.
#
# Three checks cannot be made from here and say so instead of guessing: svn commit access (testing
# it means writing to the ASF dist area), the Docker Hub secrets on the repository (listing them
# needs an admin token), and whether the Docker Hub publish path works at all (it has never run).
#
#   bash tools/releasing/preflight.sh
#
# release.sh sources this file and calls run_preflight, so the two cannot drift apart.

REQUIRED_TOOLS=(gpg svn shasum git go helm yq gh tar)
SVN_DEV_URL="https://dist.apache.org/repos/dist/dev/skywalking/swck"
SVN_RELEASE_URL="https://dist.apache.org/repos/dist/release/skywalking/swck"
KEYS_URL="https://downloads.apache.org/skywalking/KEYS"

PREFLIGHT_BLOCKERS=0

_ok()    { printf '  \033[32m✓\033[0m %s\n' "$1"; }
_warn()  { printf '  \033[33m!\033[0m %s\n' "$1"; }
_block() { printf '  \033[31m✗\033[0m %s\n' "$1"; PREFLIGHT_BLOCKERS=$((PREFLIGHT_BLOCKERS + 1)); }

# The tools release.sh and `make release` shell out to.
check_tools() {
    echo "Tools"
    local missing=()
    for tool in "${REQUIRED_TOOLS[@]}"; do
        command -v "$tool" &>/dev/null || missing+=("$tool")
    done
    if [ ${#missing[@]} -eq 0 ]; then
        _ok "all present: ${REQUIRED_TOOLS[*]}"
    else
        _block "missing: ${missing[*]}"
    fi
}

# The key that signs the artifacts every voter will verify. Checked the same way release.sh
# chooses it, so this reports on the key that would actually be used.
check_gpg() {
    echo "GPG signing key"
    local key uids email
    key=$(git config user.signingkey 2>/dev/null || true)
    if [ -z "$key" ]; then
        key=$(gpg --list-secret-keys --keyid-format LONG 2>/dev/null | grep -A1 '^sec' | tail -1 | awk '{print $1}' || true)
        [ -n "$key" ] && _warn "git config user.signingkey is unset; falling back to the first secret key. Set it explicitly if you hold more than one."
    fi
    if [ -z "$key" ]; then
        _block "no GPG secret key on this machine"
        return
    fi

    uids=$(gpg --list-secret-keys --with-colons "$key" 2>/dev/null | awk -F: '$1 == "uid" { print $10 }')
    if [ -z "$uids" ]; then
        _block "no secret key matching ${key}"
        return
    fi
    email=$(echo "$uids" | grep -oE '[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}' | grep '@apache.org$' | head -1)
    if [ -z "$email" ]; then
        _block "key ${key} has no @apache.org UID (${uids//$'\n'/, }); Apache releases must be signed with one"
        return
    fi
    _ok "would sign as ${email} (${key})"
    # release.sh signs with this, and build/package/release.sh passes it to gpg as --local-user.
    # Resolved once, here, so the two cannot pick different keys.
    PREFLIGHT_GPG_KEY_ID="${key}"
    PREFLIGHT_GPG_UIDS="${uids}"
    PREFLIGHT_GPG_EMAIL="${email}"

    # Voters run `gpg --verify` against the published KEYS file. A key that is not in it produces a
    # candidate nobody can verify, which is found out on the mailing list rather than here.
    local keys
    keys=$(mktemp)
    if curl -sSL --max-time 60 "${KEYS_URL}" -o "${keys}" 2>/dev/null && [ -s "${keys}" ]; then
        if gpg --dry-run --import-options show-only --import "${keys}" 2>/dev/null | grep -qi "${key}"; then
            _ok "that key is in the published KEYS file"
        else
            _block "that key is NOT in ${KEYS_URL}; add it before releasing or no voter can verify the signature"
        fi
    else
        _warn "could not fetch KEYS; check by hand that ${key} is in ${KEYS_URL}"
    fi
    rm -f "${keys}"
}

# gh publishes the GitHub release and opens the next-version PR.
check_github() {
    echo "GitHub"
    if gh auth status &>/dev/null; then
        _ok "gh is authenticated as $(gh api user -q .login 2>/dev/null || echo 'unknown')"
    else
        _block "gh is not authenticated; run 'gh auth login'"
    fi
}

# Read access proves the URLs resolve and shows what is already there. Commit access cannot be
# tested without writing to the ASF dist area, so it is not tested.
check_svn() {
    echo "Apache dist (svn)"
    local out
    for url in "${SVN_DEV_URL}" "${SVN_RELEASE_URL}"; do
        if out=$(svn ls "${url}" 2>&1); then
            _ok "${url##*/dist/} readable$([ -n "${out}" ] && echo " (holds: $(echo ${out} | tr '\n' ' '))")"
        else
            _block "${url} is not readable: $(echo "${out}" | head -1)"
        fi
    done
    _warn "commit access is NOT checked here -- testing it means writing to the ASF dist area. release.sh step 7 is the first real test."
}

# The version has to agree in three places, and release.sh derives everything from it.
check_version() {
    echo "Version"
    local chart_version changelog_version last_tag
    chart_version=$(awk '/^version:/{print $2; exit}' "${PROJECT_DIR}/${CHART_DIR_REL}/Chart.yaml" 2>/dev/null)
    changelog_version=$(awk '/^## /{gsub(/^## /,""); print; exit}' "${PROJECT_DIR}/docs/en/changes/changes.md" 2>/dev/null)
    last_tag=$(git -C "${PROJECT_DIR}" describe --tags --abbrev=0 2>/dev/null || echo "none")

    if [[ ! "${chart_version}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        _block "chart version '${chart_version}' is not MAJOR.MINOR.PATCH"
    elif [ "${chart_version}" != "${changelog_version}" ]; then
        _block "Chart.yaml says ${chart_version} but the changelog's newest section is '${changelog_version}'"
    else
        _ok "releasing ${chart_version}; previous tag ${last_tag}"
    fi

    if git -C "${PROJECT_DIR}" rev-parse "v${chart_version}" &>/dev/null; then
        _block "tag v${chart_version} already exists locally; delete it or bump the version"
    elif git -C "${PROJECT_DIR}" ls-remote --exit-code --tags origin "v${chart_version}" &>/dev/null; then
        _block "tag v${chart_version} already exists on the remote; this version was released"
    else
        _ok "tag v${chart_version} is free"
    fi
}

# A stale candidate in dist/dev is a leftover of an abandoned release and confuses voters.
check_dist_dev_clean() {
    echo "Previous attempts"
    local existing
    existing=$(svn ls "${SVN_DEV_URL}" 2>/dev/null || true)
    if [ -z "${existing}" ]; then
        _ok "dist/dev is empty"
    else
        _warn "dist/dev already holds: $(echo ${existing} | tr '\n' ' ') -- remove an abandoned candidate before uploading a new one"
    fi
}

# The chart ships generated CRDs and RBAC. A release built from a drifted tree installs an operator
# that cannot reconcile what it claims to.
check_generated_in_sync() {
    echo "Generated files"
    if ! git -C "${PROJECT_DIR}" diff --quiet || ! git -C "${PROJECT_DIR}" diff --cached --quiet; then
        _block "the working tree has uncommitted changes; release.sh clones from the remote, so they would be silently left out"
    else
        _ok "working tree is clean"
    fi
    if (cd "${PROJECT_DIR}" && make chart-check >/dev/null 2>&1); then
        _ok "chart manifests are in sync with the operator sources"
    else
        _block "make chart-check fails; run 'make chart-manifests' and commit the result"
    fi
}

# The release path pushes to Docker Hub with repository secrets this cannot read.
check_dockerhub() {
    echo "Docker Hub"
    _warn "DOCKERHUB_USER / DOCKERHUB_TOKEN cannot be checked from here (listing repository secrets needs an admin token). Confirm they exist, or the two Docker Hub jobs fail while GHCR still publishes."
}

run_preflight() {
    PREFLIGHT_BLOCKERS=0
    echo ""
    echo "=== Release preflight ==="
    echo ""
    check_tools;               echo ""
    check_gpg;                 echo ""
    check_github;              echo ""
    check_svn;                 echo ""
    check_version;             echo ""
    check_dist_dev_clean;      echo ""
    check_generated_in_sync;   echo ""
    check_dockerhub;           echo ""

    if [ "${PREFLIGHT_BLOCKERS}" -gt 0 ]; then
        echo "Preflight found ${PREFLIGHT_BLOCKERS} blocking problem(s). Fix them before releasing."
        return 1
    fi
    echo "Preflight passed. Items marked ! are advisory or cannot be checked from here."
    return 0
}

# Only run when executed directly; release.sh sources this file for run_preflight.
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    set -o pipefail
    SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
    PROJECT_DIR=$(cd "${SCRIPT_DIR}/../.." && pwd)
    CHART_DIR_REL="chart/skywalking-swck"
    run_preflight
fi
