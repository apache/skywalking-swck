# End-to-end tests

Every end-to-end case runs the real thing: a [kind](https://kind.sigs.k8s.io/) cluster, the operator
built from the working tree, real custom resources, and — where the case needs it — a real OAP
server, UI, storage and Java application. They are driven by
[skywalking-infra-e2e](https://github.com/apache/skywalking-infra-e2e), configured by one
`test/e2e/<case>/e2e.yaml` per case, and each has a job in
[`.github/workflows/go.yml`](../../../.github/workflows/go.yml).

Run one locally:

```shell
make e2e-chart          # runs test/e2e/chart/e2e.yaml
make e2e-test           # runs every case; this takes a long time
```

The target name is the directory name, so there is no list to keep in step with the tree.

## Anatomy of a case

```yaml
setup:
  env: kind                 # a kind cluster from test/e2e/kind.yaml
  steps:                    # ordered; a failing step fails the case
    - name: ...
      command: ...
      wait: ...             # kubectl wait conditions, retried until the setup timeout
trigger:                    # optional: generate traffic, so there is telemetry to assert on
verify:
  cases:
    - query: ...            # any shell command
      expected: ...         # a file the output must match, or contain
```

`expected` files live in [`test/e2e/verify/`](../../../test/e2e/verify) and may use
`{{- contains . }}`, which asserts the output contains the listed entries rather than equals them.

## Test images

Every image the suite deploys is pinned in [`test/e2e/env`](../../../test/e2e/env), one variable
per image. skywalking-infra-e2e loads it into every setup step via
`setup.init-system-environment: ../env`, and the manifests reference the variables directly, so
applying one is plain `envsubst`:

```shell
envsubst "$E2E_IMAGES" < test/e2e/skywalking-components.yaml | kubectl apply -f -
```

`E2E_IMAGES` is the list of every image variable, defined at the bottom of `env`. Passing it
restricts substitution to those names, so the regexes and shell-looking strings elsewhere in the
manifests — `namespace: "^skywalking-system$"`, for one — are left alone. A bare `envsubst` would
mangle them.

The rules:

* **Pin, never float.** `apache/skywalking-banyandb:latest` used to be in this suite: a moving
  target that silently changes what CI tests and breaks on someone else's merge.
* **Prefer a GHCR commit id.** Docker Hub applies anonymous pull rate limits that CI runners share
  with the rest of the internet, and every SkyWalking component publishes a per-commit image to
  GHCR on every push to its default branch.
* **Except OAP and the UI**, which are pinned to their Docker Hub *release* tags, matching
  [`apache/skywalking-helm`'s `test/e2e/env`](https://github.com/apache/skywalking-helm/blob/master/test/e2e/env).
  Those two are what the operator's own webhook defaults point at, so CI validates the artifacts
  users are actually told to install.

The operator and the metrics adapter are **not** in that file — they are built from the working
tree and side-loaded into kind, because e2e exists to test the code in the pull request.

Elasticsearch and the kind node image are third-party and stay on their own registries,
version-pinned.

### The UI is Horizon

`horizon` is the only `spec.kind` the operator accepts, and what every case here deploys. Horizon
shares the `apache/skywalking-ui` repository with the retired Booster UI and is told apart by the
`horizon-` tag prefix, so `UI_IMAGE` is `apache/skywalking-ui:horizon-1.0.0` and a Horizon
`spec.version` is the *Horizon* version, not the OAP one.

Booster is gone: `apache/skywalking` removed its submodule in 11.0.0 and its master `Makefile` now
has `DOCKER_TARGETS := docker.oap`, so no 11.x image is built for it and
`ghcr.io/apache/skywalking/ui` stops at the last commit that produced one.

OAP, Horizon and BanyanDB move as one: OAP 11.0.0 accepts BanyanDB server API 0.11 only, and
Horizon 1.0.0's admin host and template store are OAP 11 additions. Bumping one alone produces a
combination nobody supports.

Horizon does not proxy GraphQL -- its BFF serves its own `/api/*` routes, behind RBAC -- so the
`swctl` queries that used to go through `http://<ui>/graphql` now go straight to OAP, exactly as
`apache/skywalking-helm`'s e2e does.

The UI is checked two ways. [`ui-cases.yaml`](../../../test/e2e/ui-cases.yaml), shared by every case
that deploys a UI, calls `/api/auth/health` -- the one route Horizon declares public -- and asserts
the auth backend it reports. That proves the container is up on 8081 rather than Booster's 8080 and
that the Service maps 80 to it. It deliberately claims no more than that: `local` is also Horizon's
own default, so the assertion holds whether or not the operator's `HORIZON_*` variables took effect.

What proves those is `oap-ui-agent`, which logs in and reads `/api/oap/info` and
`/api/cluster/state` back through the BFF -- both report what Horizon resolved its OAP hosts to, so
they fail if the wiring did not arrive.

### Configuration paths

Horizon takes its configuration from `HORIZON_*` environment variables against the tokenised config
baked into its image, and that is what almost every case runs: no `spec.config`, so no ConfigMap is
created and nothing is mounted over the baked file.
[`oap-ui-agent`](../../../test/e2e/oap-ui-agent/e2e.yaml) is the case that proves it — it logs in
and reads `/api/oap/info` and `/api/cluster/state` back through the BFF, both of which report what
Horizon resolved its OAP hosts to, so the case fails if the operator's variables did not arrive.

Exactly one case takes the override path.
[`oap-ui-agent-oapserverconfig-oapserverdynamicconfig`](../../../test/e2e/oap-ui-agent-oapserverconfig-oapserverdynamicconfig/e2e.yaml)
uses [`skywalking-components-config-override.yaml`](../../../test/e2e/skywalking-components-config-override.yaml),
where the `UI` carries a whole `horizon.yaml` in `spec.config` — mounted as a ConfigMap over the
baked one — while its `OAPServerConfig` overlays a static file on the OAP through `file:`. It is the
case that proves a user can take configuration over from the operator on both sides.

Logging in needs a user, and Horizon ships `auth.local.users` empty, so a UI refuses every login
until one is seeded. On the env path that comes from a `Secret` through `spec.envFrom`, which is
also what proves `envFrom` reaches the container: if it did not, there would be no user and the
login would fail. On the override path the user is in the `horizon.yaml` itself;
[`test/tools/horizon.sh`](../../../test/tools/horizon.sh) then logs in, keeps the `horizon_sid`
session cookie and calls an RBAC-protected route, mirroring
`apache/skywalking-helm`'s `test/e2e/script/horizon.sh`. `oap-ui-agent` does this, which also makes
it the only case that exercises `spec.config` at all.

### Expectations that name an image

[`verify/swagent-initcontainer.yaml`](../../../test/e2e/verify/swagent-initcontainer.yaml) and its
configmap sibling assert that the injector injected *exactly* the image the `SwAgent` CR asked for,
so they carry the same variable. The case runs the same `envsubst` over them into
`*.rendered.yaml` — gitignored — in the step that creates the workload, and points `expected:` at
the rendered file.

Two images are not in any manifest, and both used to slip past the pins: the sidecar the Java agent
injector adds, whose default is hardcoded in
[`annotations.yaml`](../../../operator/pkg/operator/manifests/injector/templates/annotations.yaml),
and the EventExporter image, which the operator derives from `spec.version`. `demo.yaml` now sets
`sidecar.skywalking.apache.org/initcontainer.Image` and the EventExporter CR sets `image`, so both
come from `env` like everything else.

## How the operator gets installed

Ten of the twelve cases install the operator with the **Helm chart**:

```shell
helm install skywalking-swck chart/skywalking-swck \
  --namespace skywalking-swck-system --create-namespace \
  --set operator.image.repository=controller \
  --set operator.image.tag=latest \
  --set operator.image.pullPolicy=IfNotPresent \
  --wait --timeout 10m
```

This is deliberate. Before, every case ran `make -C operator install && make -C operator deploy`,
which installs the CRDs and the operator with kustomize. Swapping that one step turned the whole
existing suite into chart coverage — real custom resources, cert-manager, webhooks, RBAC — without
writing a single new assertion.

[`test/e2e/oap-ui-agent`](../../../test/e2e/oap-ui-agent/e2e.yaml) is deliberately **left on
kustomize**. `operator/config` is still a supported install path and ships in the binary release as
`operator-bundle.yaml`, so something has to keep exercising it.

## Coverage

| Case | Install | What it proves |
| --- | --- | --- |
| [`chart`](../../../test/e2e/chart/e2e.yaml) | chart, **packaged tarball** | The chart's own lifecycle — see below |
| [`oap-ui-agent`](../../../test/e2e/oap-ui-agent/e2e.yaml) | **kustomize** | The kustomize install path; `OAPServer` + `UI` + Java agent injection end to end |
| [`oap-ui-swagent`](../../../test/e2e/oap-ui-swagent/e2e.yaml) | chart | `SwAgent` drives the injected sidecar |
| [`oap-ui-swagent-configmap`](../../../test/e2e/oap-ui-swagent-configmap/e2e.yaml) | chart | `SwAgent` mounting an agent config from a ConfigMap |
| [`oap-ui-agent-internal-storage`](../../../test/e2e/oap-ui-agent-internal-storage/e2e.yaml) | chart | `Storage` provisioning Elasticsearch inside the cluster |
| [`oap-ui-agent-external-storage`](../../../test/e2e/oap-ui-agent-external-storage/e2e.yaml) | chart | `Storage` pointing at an Elasticsearch the operator does not own |
| [`banyandb`](../../../test/e2e/banyandb/e2e.yaml) | chart | `BanyanDB` as the OAP storage |
| [`oap-ui-agent-satellite`](../../../test/e2e/oap-ui-agent-satellite/e2e.yaml) | chart | `Satellite` in front of OAP |
| [`oap-agent-adapter-hpa`](../../../test/e2e/oap-agent-adapter-hpa/e2e.yaml) | chart, adapter on | HPA scales a deployment on a SkyWalking metric |
| [`oap-satellite-adapter-hpa`](../../../test/e2e/oap-satellite-adapter-hpa/e2e.yaml) | chart, adapter on | The same, with Satellite in the path |
| [`oap-ui-agent-oapserverconfig-oapserverdynamicconfig`](../../../test/e2e/oap-ui-agent-oapserverconfig-oapserverdynamicconfig/e2e.yaml) | chart | `OAPServerConfig` and `OAPServerDynamicConfig` reach a running OAP |
| [`oap-eventexporter`](../../../test/e2e/oap-eventexporter/e2e.yaml) | chart | `EventExporter` exports operator events into OAP |

Per feature:

| Feature | Covered by |
| --- | --- |
| Operator deployment, kustomize | `oap-ui-agent` |
| Operator deployment, Helm chart | every other case |
| Chart install / upgrade / uninstall | `chart` |
| CRDs installed and reconciled | every case; counted explicitly in `chart` |
| Admission webhooks and CA injection | every case implicitly; asserted in `chart` |
| Java agent sidecar injection | `oap-ui-agent`, `oap-ui-swagent`, `oap-ui-swagent-configmap`, `chart` |
| `SwAgent` / `JavaAgent` | `oap-ui-swagent`, `oap-ui-swagent-configmap` |
| `OAPServer`, `UI` | `oap-ui-agent` and most others |
| `Storage` | `oap-ui-agent-internal-storage`, `oap-ui-agent-external-storage` |
| `BanyanDB` | `banyandb` |
| `Satellite` | `oap-ui-agent-satellite`, `oap-satellite-adapter-hpa` |
| `OAPServerConfig`, `OAPServerDynamicConfig` | `oap-ui-agent-oapserverconfig-oapserverdynamicconfig` |
| `EventExporter` | `oap-eventexporter` |
| Custom metrics adapter, HPA | `oap-agent-adapter-hpa`, `oap-satellite-adapter-hpa`, `chart` |
| Operator metrics behind kube-rbac-proxy | `chart` |

`Fetcher` has no case of its own; it is documented by the
[Istio control plane example](../examples/istio-controlplane.md). Adding one is the clearest gap in
this suite.

## The chart case

[`test/e2e/chart`](../../../test/e2e/chart/e2e.yaml) is the only case that tests the **chart**
rather than the operator. Everything it does happens in one step,
[`lifecycle.sh`](../../../test/e2e/chart/lifecycle.sh), because part of what it asserts is what
survives `helm uninstall` — which has to run last. The step writes a report that the `verify` phase
compares against [`verify/chart-lifecycle.yaml`](../../../test/e2e/verify/chart-lifecycle.yaml).

It installs the **packaged tarball**, not the source directory, because the tarball is what is
released and what users install. It asserts:

* the packaged chart installs, with the adapter enabled;
* it brings one CRD per file in `operator/config/crd/bases` — the count is read off the generated
  sources at run time, so adding a CRD does not need the expectations edited;
* cert-manager injects the CA into both webhook configurations. Without it the API server cannot
  reach the webhook and every custom resource is rejected — while the chart still installs cleanly
  and the operator still reports ready, so nothing else would notice;
* the webhook configurations cover every kind the operator serves (12 mutating, 10 validating);
* the operator reconciles an `OAPServer` into a Deployment and a Service, which needs the CRD, both
  webhooks, the ClusterRole and the service account binding all to be right;
* the Java agent injector mutates a pod in an opted-in namespace;
* the metrics adapter owns and serves `external.metrics.k8s.io`;
* the operator's metrics endpoint is served through the kube-rbac-proxy sidecar;
* `helm upgrade` removes the adapter and puts it back, APIService and all;
* `helm uninstall` leaves the CRDs — and therefore every custom resource in the cluster — in place,
  and leaves nothing cluster-scoped behind for the next install to collide with.

The chart is also checked without a cluster, on every PR, by `make chart-check` and
`make chart-lint`:

* **`chart-check`** regenerates the CRDs, the manager ClusterRole and the webhook configurations
  from the operator sources and fails on any difference. This is why the chart lives in this
  repository: it cannot drift from the operator it ships beside.
* **`chart-lint`** renders the chart under the values that change what it produces — adapter on and
  off, webhooks off, metrics off, GHCR images, externally managed service accounts, a maximum-length
  release name — and asserts on the output. `helm lint` and `helm template` both exit 0 on a chart
  that renders nothing, so neither proves anything on its own.

## Adding a case

1. `mkdir test/e2e/<case>` and write `e2e.yaml`. Copy the closest existing case; the setup steps up
   to and including the operator install are the same everywhere.
2. Put expectations in `test/e2e/verify/`.
3. Add a job to `.github/workflows/go.yml` and list it under the `checks` job's `needs`, otherwise
   a failure will not block the merge.
4. Add the case to `E2E_CASES` in the [`Makefile`](../../../Makefile) and to the table above.
