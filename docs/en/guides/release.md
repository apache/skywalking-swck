# Apache SkyWalking Cloud on Kubernetes release guide

This documentation guides the release manager to release the SkyWalking Cloud on Kubernetes in the Apache Way, and also helps people to check the release for vote.

## Two scripts

Most of what follows is automated by two scripts, modelled on `apache/skywalking`'s own
`tools/releasing/release.sh`. Read the rest of this guide anyway: the scripts do exactly these steps,
and when one fails you need to know which step it was on.

```shell
bash tools/releasing/release.sh          # everything up to the vote
# ... the vote runs for at least 72 hours ...
bash tools/releasing/release-passed.sh   # everything after it passes
```

| | `release.sh` | `release-passed.sh` |
| --- | --- | --- |
| Tags and pushes `v$VERSION` | yes | |
| Builds, signs and verifies the three tarballs | yes | |
| Uploads the candidate to `dist/dev` | yes | |
| Drafts the vote email | yes | |
| Opens the next-version PR | yes | |
| Moves the release to `dist/release`, removes the previous one | | yes |
| Publishes the GitHub release, which publishes the images and the chart | | yes |
| Verifies all five registry artifacts | | yes |
| Drafts the vote-result and announcement emails | | yes |

Neither script publishes an image or a chart before the vote passes. That is not only policy: the
Docker Hub image is built from the **released** binary tarball, downloaded from `archive.apache.org`
and GPG-verified, so it cannot be built until the release is real.

Do not confuse `tools/releasing/release.sh` with `build/package/release.sh`, which is the packaging
helper `make release` calls.

## Prerequisites

1. Close(if finished, or move to next milestone otherwise) all issues in the current milestone from [skywalking-swck](https://github.com/apache/skywalking-swck/milestones) and [skywalking](https://github.com/apache/skywalking/milestones), create a new milestone if needed.
2. Update [the changelog](../changes/changes.md).
3. Update image tags of adapter and operator.
4. Update the chart version in `chart/skywalking-swck/Chart.yaml` to the version being released. See
   [Release artifacts and version scheme](#release-artifacts-and-version-scheme) for how the tag, the
   image tags and the chart version relate.
5. Refresh the generated parts of the chart. The CRDs, the operator's ClusterRole and the admission
   webhook configurations are **generated** from the operator sources, never hand-edited:

   ```shell
   make chart-manifests
   git status --porcelain chart/skywalking-swck
   ```

   CI runs `make chart-check`, which does the same and fails on any difference, so this should be a
   no-op by the time you release. The chart is the only thing that installs the CRDs -- `helm install`
   replaced `make -C operator install` in the e2e suites -- so a chart released with a stale or empty
   `crds/` produces an operator that cannot reconcile anything.

## Release artifacts and version scheme

One release of this repository produces **one source release** plus a set of **convenience binaries**.
Everything is derived from a single Git tag -- never invent a per-artifact version.

| Where | Artifact | Coordinate, for `v0.11.0` |
| --- | --- | --- |
| dist.apache.org | source tarball -- **this is the release, and this is what is voted on** | `skywalking-swck-0.11.0-src.tgz` |
| dist.apache.org | binary distribution | `skywalking-swck-0.11.0-bin.tgz` |
| dist.apache.org | Helm chart tarball | `skywalking-swck-0.11.0.tgz` |
| GHCR | operator image, built from `operator/Dockerfile` | `ghcr.io/apache/skywalking-swck/operator:0.11.0` |
| GHCR | custom metrics adapter image, built from `adapter/Dockerfile` | `ghcr.io/apache/skywalking-swck/metrics-adapter:0.11.0` |
| GHCR | Helm chart | `ghcr.io/apache/skywalking-swck/helm/skywalking-swck:0.11.0` |
| Docker Hub | combined image, carrying **both** `/manager` and `/adapter`, `ENTRYPOINT ["/manager"]` | `apache/skywalking-swck:0.11.0` |
| Docker Hub | Helm chart | `apache/skywalking-swck:0.11.0-helm` |

The rules behind that table:

* The Git tag is `v$VERSION` (`v0.11.0`). Every other coordinate drops the `v` and is plain `$VERSION`
  (`0.11.0`). Do not push an image tagged `v0.11.0`. Docker Hub history is mixed -- `0.1.0`-`0.3.0` had no
  `v`, `v0.4.0`-`v0.9.0` did, and `0.10.0` dropped it again -- so `0.10.0` is the convention to follow.
  `build/package/release.sh` stamps the same `$VERSION` into the generated `operator-bundle.yaml` and
  `adapter-bundle.yaml`, so the bundles in the binary tarball resolve to the image the workflow pushes.
* There is exactly **one** chart, named `skywalking-swck`. It deploys the operator and, behind a values
  flag, the custom metrics adapter. It is not split into two charts, because the Docker Hub layout below
  leaves room for only one chart artifact.
* The chart version is `$VERSION` on GHCR and `$VERSION-helm` on Docker Hub. Both are cut from the same
  chart sources in the same release, but they are **not byte-identical**: `helm package` stamps the
  version into `Chart.yaml`, so the Docker Hub copy differs from the voted tarball in that one field.
  The voted artifact is the `$VERSION` tarball on `dist.apache.org`.
* GHCR carries the images **split** (`operator`, `metrics-adapter`); Docker Hub carries the single
  **combined** image that has both binaries in it. Do not "fix" this into a split on Docker Hub -- the
  operator bundle manifests generated by `build/package/release.sh` point at `apache/skywalking-swck`
  for both the manager and the adapter.

### Why the `-helm` suffix on Docker Hub

`helm push` derives the OCI **repository** from the chart *name* and the OCI **tag** from the chart
*version*. Because this chart is named `skywalking-swck`, pushing it into the `apache` namespace on Docker
Hub lands it on `apache/skywalking-swck` -- the very repository the combined image already occupies. Image
and chart therefore have to be separated by tag:

* An OCI tag points at exactly one manifest. Pushing the chart to the tag the image occupies **re-points
  that tag** at the chart manifest, whose config mediaType is `application/vnd.cncf.helm.config.v1+json`;
  `docker pull apache/skywalking-swck:$VERSION` then no longer returns the image. The image manifest is
  not deleted -- it survives by digest -- but the tag is lost, which for users amounts to the same thing.
  Distinct tags are mandatory. The publish workflow re-inspects the image tag after the chart push and
  fails the run if it finds a Helm config mediaType there.
* The distinguishing part is carried in the **version**, so it has to be valid SemVer. `0.11.0-helm` is
  accepted and pushes to the tag `0.11.0-helm`, coexisting with the image at `0.11.0`; `helm-0.11.0` is
  rejected by `helm package` with `Error: invalid semantic version` (verified against helm v4.1.1).
* Note that `0.11.0-helm` is a SemVer **pre-release**: it sorts *below* `0.11.0` and is excluded from
  range constraints, so it must always be requested with an exact `--version 0.11.0-helm`.

The alternative would have been to name the chart `skywalking-swck-helm`, which is what the sibling
`skywalking-helm` and `skywalking-banyandb-helm` charts do -- their chart names carry the `-helm` suffix,
so they get their own Docker Hub repository and never collide with an image. That was not chosen here
because the chart is released from, and named after, this repository. If that trade is ever revisited, the
`-helm` version suffix and the mediaType guard in the workflow both go away.

GHCR has no such constraint: images and the chart sit in separate repositories under
`ghcr.io/apache/skywalking-swck/`, so the chart is published there at the plain `$VERSION`.

## Add your GPG public key to Apache svn

1. Log in [id.apache.org](https://id.apache.org/) and submit your key fingerprint.

1. Add your GPG public key into [SkyWalking GPG KEYS](https://dist.apache.org/repos/dist/release/skywalking/KEYS) file, **you can do this only if you are a PMC member**.  You can ask a PMC member for help. **DO NOT override the existed `KEYS` file content, only append your key at the end of the file.**

## Build and sign the source code package

```shell
export VERSION=<the version to release>
git clone git@github.com:apache/skywalking-swck && cd skywalking-swck
git tag -a "v$VERSION" -m "Release Apache SkyWalking Cloud on Kubernetes $VERSION"
git push --tags
make clean && make release
```

The `skywalking-swck-${VERSION}-bin.tgz`, `skywalking-swck-${VERSION}-src.tgz`, and their corresponding `asc`, `sha512` are written to `build/release`. **In total, six files should be automatically generated in the directory.**

### Package and sign the Helm chart

The chart tarball is a signed, voted release artifact and ships next to the source and binary tarballs --
this follows the existing SkyWalking precedent, where `skywalking-helm` and `skywalking-banyandb-helm`
both publish a signed `helm package` tarball on `dist.apache.org` alongside their `-src.tgz`.

`make release` already does this -- it runs `release-chart` and `release-sign` -- so there is nothing
extra to run. What it does, and what to do by hand if you have to:

```shell
# LICENSE and NOTICE have to be *inside* the chart tarball
cp LICENSE NOTICE chart/skywalking-swck/
helm dep up chart/skywalking-swck
helm package chart/skywalking-swck --version "$VERSION" --app-version "$VERSION" -d build/release
rm -f chart/skywalking-swck/LICENSE chart/skywalking-swck/NOTICE

pushd build/release
gpg --batch --yes --armor --detach-sig "skywalking-swck-$VERSION.tgz"
shasum -a 512 "skywalking-swck-$VERSION.tgz" > "skywalking-swck-$VERSION.tgz.sha512"
popd
```

`build/package/release.sh` refuses to package a chart whose `Chart.yaml` version disagrees with the
release version, so a forgotten bump fails here rather than shipping a mislabelled chart.

Note the version passed here is the plain `$VERSION`. The Docker Hub `$VERSION-helm` variant is produced
at publish time, after the vote -- it is a registry-tagging concern, not a release artifact, and it is
**not** uploaded to `dist.apache.org`.

**Nine files in total** should now be in `build/release`: the `-src`, `-bin` and chart tarballs, each with
its `.asc` and `.sha512`.

## Upload to Apache svn

```shell
svn co https://dist.apache.org/repos/dist/dev/skywalking/
mkdir -p skywalking/swck/"$VERSION"
cp skywalking-swck/build/release/skywalking-swck*.tgz skywalking/swck/"$VERSION"
cp skywalking-swck/build/release/skywalking-swck*.tgz.asc skywalking/swck/"$VERSION"
cp skywalking-swck/build/release/skywalking-swck*.tgz.sha512 skywalking/swck/"$VERSION"

cd skywalking/swck && svn add "$VERSION" && svn commit -m "Draft Apache SkyWalking-SWCK release $VERSION"
```

The globs above pick up all nine files, the chart tarball included. Confirm that before committing.

## Make the internal announcement

Send an announcement email to dev@ mailing list.

```text
Subject: [ANNOUNCEMENT] SkyWalking Cloud on Kubernetes $VERSION test build available

Content:

The test build of SkyWalking Cloud on Kubernetes $VERSION is now available.

We welcome any comments you may have, and will take all feedback into
account if a quality vote is called for this build.

Release notes:

 * https://github.com/apache/skywalking-swck/blob/$VERSION/docs/en/changes/changes.md

Release Candidate:

 * https://dist.apache.org/repos/dist/dev/skywalking/swck/$VERSION
 * sha512 checksums
   - sha512xxxxyyyzzz skywalking-swck-x.x.x-src.tgz
   - sha512xxxxyyyzzz skywalking-swck-x.x.x-bin.tgz
   - sha512xxxxyyyzzz skywalking-swck-x.x.x.tgz (Helm chart)

Release Tag :

 * (Git Tag) v$VERSION

Release Commit Hash :

 * https://github.com/apache/skywalking-swck/tree/<Git Commit Hash>

Keys to verify the Release Candidate :

 * https://dist.apache.org/repos/dist/release/skywalking/KEYS

Guide to build the release from source :

 * https://github.com/apache/skywalking-swck/blob/$VERSION/docs/en/setup/operator.md#build-from-sources
 * https://github.com/apache/skywalking-swck/blob/$VERSION/docs/en/setup/custom-metrics-adapter.md#use-kustomize-to-customise-your-deployment
 * https://github.com/apache/skywalking-swck/blob/$VERSION/docs/en/guides/release.md

A vote regarding the quality of this test build will be initiated
within the next couple of days.
```

## Wait at least 48 hours for test responses

Any PMC, committer or contributor can test features for releasing, and feedback.
Based on that, PMC will decide whether to start a vote or not.

## Call for vote in dev@ mailing list

Call for vote in `dev@skywalking.apache.org`

```text
Subject: [VOTE] Release Apache SkyWalking Cloud on Kubernetes version $VERSION

Content:

Hi the SkyWalking Community:
This is a call for vote to release Apache SkyWalking Cloud on Kubernetes version $VERSION.

Release notes:

 * https://github.com/apache/skywalking-swck/blob/$VERSION/docs/en/changes/changes.md

Release Candidate:

 * https://dist.apache.org/repos/dist/dev/skywalking/swck/$VERSION
 * sha512 checksums
   - sha512xxxxyyyzzz skywalking-swck-x.x.x-src.tgz
   - sha512xxxxyyyzzz skywalking-swck-x.x.x-bin.tgz
   - sha512xxxxyyyzzz skywalking-swck-x.x.x.tgz (Helm chart)

Release Tag :

 * (Git Tag) $VERSION

Release Commit Hash :

 * https://github.com/apache/skywalking-swck/tree/<Git Commit Hash>

Keys to verify the Release Candidate :

 * https://dist.apache.org/repos/dist/release/skywalking/KEYS

Guide to build the release from source :

 * https://github.com/apache/skywalking-swck/blob/$VERSION/docs/en/guides/release.md

Voting will start now and will remain open for at least 72 hours, all PMC members are required to give their votes.

[ ] +1 Release this package.
[ ] +0 No opinion.
[ ] -1 Do not release this package because....

Thanks.

[1] https://github.com/apache/skywalking/blob/master/docs/en/guides/How-to-release.md#vote-check
```

## Vote Check

All PMC members and committers should check these before voting +1:

1. Features test.
1. All artifacts in staging repository are published with `.asc`, `.md5`, and `sha` files.
1. Source codes, distribution packages and the Helm chart
(`skywalking-swck-$VERSION-src.tgz`, `skywalking-swck-$VERSION-bin.tgz`, `skywalking-swck-$VERSION.tgz`)
are in `https://dist.apache.org/repos/dist/dev/skywalking/swck/$VERSION` with `.asc`, `.sha512`.
1. `LICENSE` and `NOTICE` are in source codes, distribution package and the chart tarball.
1. Check `shasum -c skywalking-swck-$VERSION-{src,bin}.tgz.sha512` and `shasum -c skywalking-swck-$VERSION.tgz.sha512`.
1. Check GPG signature. Download KEYS and import them by `curl https://www.apache.org/dist/skywalking/KEYS -o KEYS && gpg --import KEYS`. Check `gpg --batch --verify <artifact>.tgz.asc <artifact>.tgz` for each of the three tarballs.
1. Build distribution from source code package by following this [the build guide](#build-and-sign-the-source-code-package).
1. Chart sanity. `helm lint` and `helm template` both **succeed on a chart that renders nothing**, so
neither on its own proves anything -- check the output, not just the exit code:

    ```shell
    helm lint skywalking-swck-$VERSION.tgz
    # must be non-empty, and must contain the operator Deployment
    helm template swck skywalking-swck-$VERSION.tgz | tee /tmp/swck-render.yaml | wc -l
    grep -q 'kind: Deployment' /tmp/swck-render.yaml
    # the CRDs must be in the tarball -- one per file in operator/config/crd/bases
    tar tzf skywalking-swck-$VERSION.tgz | grep -c 'skywalking-swck/crds/.*\.yaml'
    # and the chart must be stamped with the release version
    tar xzf skywalking-swck-$VERSION.tgz -O skywalking-swck/Chart.yaml | grep -E '^(version|appVersion):'
    ```

    `version` and `appVersion` must both be `$VERSION`, and `LICENSE`/`NOTICE` must be inside the tarball.
1. Licenses header check.

Container images and the chart in the registries are **not** part of the vote -- they do not exist yet at
this point. They are convenience binaries, published only after the vote passes.

Vote result should follow these:

1. PMC vote is +1 binding, all others is +1 no binding.

1. Within 72 hours, you get at least 3 (+1 binding), and have more +1 than -1. Vote pass. 

1. **Send the closing vote mail to announce the result**.  When count the binding and no binding votes, please list the names of voters. An example like this:

   ```
   [RESULT][VOTE] Release Apache SkyWalking Cloud on Kubernetes version $VERSION
   
   3 days passed, we’ve got ($NUMBER) +1 bindings:
   xxx
   xxx
   xxx
   ...
   (list names)
    
   I’ll continue the release process.
   ```

## Publish release

All of this is `bash tools/releasing/release-passed.sh`. It is idempotent -- every step checks
whether it has already been done -- so it is safe to re-run after a failure. What it does, and how to
do each step by hand:

1. Move the source code tarballs, the distributions and the chart tarball to
   `https://dist.apache.org/repos/dist/release/skywalking/`. **This is the act of releasing, and only
   a PMC member can do it.**

    ```shell
    export SVN_EDITOR=vim
    svn mv https://dist.apache.org/repos/dist/dev/skywalking/swck/$VERSION https://dist.apache.org/repos/dist/release/skywalking/swck
    # ....
    # enter your apache password
    # ....
    ```

1. Remove the last released tarballs from `https://dist.apache.org/repos/dist/release/skywalking`.
   `dist/release` is mirrored everywhere, so it carries only the current release; older ones stay
   available on `archive.apache.org`.

1. Publish the convenience binaries: the container images and the Helm chart.

    These are **not** the release -- the voted source tarball is. They are convenience binaries and
    **must not be pushed before the vote passes** and the tarballs have moved to `dist/release`. This
    is not only policy: `build/images/Dockerfile.release` builds the Docker Hub image by downloading
    `skywalking-swck-$VERSION-bin.tgz` from `archive.apache.org` and verifying its `.asc`, so it
    simply cannot be built until the tarball is released and has propagated to the archive.

    Publishing is triggered by **publishing the GitHub release**, exactly as in `apache/skywalking`:

    ```shell
    gh release create "v$VERSION" --repo apache/skywalking-swck --title "$VERSION" \
      --notes "See https://github.com/apache/skywalking-swck/blob/v$VERSION/docs/en/changes/changes-$VERSION.md"
    ```

    That fires `.github/workflows/publish-docker.yml` on `release: [released]`, which builds from the
    release tag and pushes the five registry artifacts listed in
    [Release artifacts and version scheme](#release-artifacts-and-version-scheme). Pushes to `master`
    continue to publish only SHA-tagged development images and a `0.0.0-<sha>` chart to GHCR.

    The workflow waits up to 30 minutes for `archive.apache.org` to serve the binary tarball before
    it builds the combined image. If the mirrors take longer than that, re-run the release path by
    hand once the URL resolves -- it is the same workflow, and the tag is its only input:

    ```shell
    gh workflow run publish-docker.yml --repo apache/skywalking-swck --ref master -f tag=v$VERSION
    ```

    Pass the tag *with* its leading `v`; the workflow strips it and refuses a tag that does not have
    one.

    Both registries need credentials: `GITHUB_TOKEN` with `packages: write` covers GHCR; Docker Hub
    needs the ASF `DOCKERHUB_USER` / `DOCKERHUB_TOKEN` secrets, provisioned by INFRA on request as
    they are for `apache/skywalking` and `apache/skywalking-java`.

    Verify every artifact afterwards, and in particular verify that the image survived the chart
    push:

    ```shell
    docker buildx imagetools inspect ghcr.io/apache/skywalking-swck/operator:$VERSION
    docker buildx imagetools inspect ghcr.io/apache/skywalking-swck/metrics-adapter:$VERSION
    helm show chart oci://ghcr.io/apache/skywalking-swck/helm/skywalking-swck --version "$VERSION"

    # the combined image must still pull AFTER the chart has been pushed to the same repository
    docker pull apache/skywalking-swck:$VERSION
    helm show chart oci://registry-1.docker.io/apache/skywalking-swck --version "$VERSION-helm"
    ```

    If `docker pull` fails here, the chart was pushed onto the image's tag and clobbered its manifest
    -- see [Why the `-helm` suffix on Docker Hub](#why-the--helm-suffix-on-docker-hub); rebuild and
    re-push the image, and correct the chart version.

    The workflow also pulls each advertised platform of every image back and checks the ELF machine
    type of the binary inside it. `apache/skywalking-swck:0.10.0`, built by hand, advertised
    `linux/arm64` while its arm64 manifest reused the amd64 layers byte for byte, so an arm64 node
    got `exec format error`. If that check fails, the release tarball is missing a
    `manager-linux-<arch>` / `adapter-linux-<arch>` pair -- see `RELEASE_ARCHS` in
    `operator/Makefile` and `adapter/Makefile`.

    If the workflow is unavailable and you must publish by hand, from a checkout of `v$VERSION` with
    `docker login` and `helm registry login` already done:

    ```shell
    # GHCR: the two split images
    docker buildx build --push --platform linux/amd64,linux/arm64 \
      -t ghcr.io/apache/skywalking-swck/operator:$VERSION operator
    docker buildx build --push --platform linux/amd64,linux/arm64 \
      -t ghcr.io/apache/skywalking-swck/metrics-adapter:$VERSION adapter

    # Docker Hub: the combined image, built from the released and signed bin tarball
    docker buildx build --push --platform linux/amd64,linux/arm64 \
      build/images -f build/images/Dockerfile.release \
      --build-arg version=$VERSION -t apache/skywalking-swck:$VERSION

    # GHCR: the chart, at the plain version
    helm package chart/skywalking-swck --version "$VERSION" --app-version "$VERSION" -d build/release
    helm push "build/release/skywalking-swck-$VERSION.tgz" oci://ghcr.io/apache/skywalking-swck/helm

    # Docker Hub: the same chart, repackaged at the -helm tag
    helm package chart/skywalking-swck --version "$VERSION-helm" --app-version "$VERSION" -d build/release
    helm push "build/release/skywalking-swck-$VERSION-helm.tgz" oci://registry-1.docker.io/apache
    ```

    No retagging is involved on GHCR: `helm push` appends the chart *name* to the OCI namespace it is
    given, so pushing into `oci://ghcr.io/apache/skywalking-swck/helm` produces
    `ghcr.io/apache/skywalking-swck/helm/skywalking-swck:$VERSION` -- the chart artifact of the
    `skywalking-swck` namespace, sitting beside `/operator` and `/metrics-adapter`.

1. Refer to the previous [PR](https://github.com/apache/skywalking-website/pull/508), update news and links on the website. There are seven files need to modify.

1. Update [Github release page](https://github.com/apache/skywalking-swck/releases), follow the previous convention.

1. Send ANNOUNCE email to `dev@skywalking.apache.org` and `announce@apache.org`, the sender should use his/her Apache email account. You can get the permlink of vote thread at [here](https://lists.apache.org/list.html?dev@skywalking.apache.org).

    ```
    Subject: [ANNOUNCEMENT] Apache SkyWalking Cloud on Kubernetes $VERSION Released

    Content:

    Hi the SkyWalking Community

    On behalf of the SkyWalking Team, I’m glad to announce that SkyWalking Cloud on Kubernetes $VERSION is now released.

    SkyWalking Cloud on Kubernetes: A bridge platform between Apache SkyWalking and Kubernetes.

    SkyWalking: APM (application performance monitor) tool for distributed systems, especially designed for microservices, cloud native and container-based (Docker, Kubernetes, Mesos) architectures.

    Vote Thread: $VOTE_THREAD_PERMALINK

    Download Links: https://skywalking.apache.org/downloads/

    Release Notes : https://github.com/apache/skywalking-swck/blob/$VERSION/docs/en/changes/changes.md

    Website: https://skywalking.apache.org/

    SkyWalking Cloud on Kubernetes Resources:
    - Issue: https://github.com/apache/skywalking/issues
    - Mailing list: dev@skywalkiing.apache.org
    - Documents: https://github.com/apache/skywalking-swck/blob/$VERSION/README.md

    The Apache SkyWalking Team
    ```
    
    
