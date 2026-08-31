# Apache SkyWalking SWCK Helm Chart

`skywalking-swck` deploys **SkyWalking Cloud on Kubernetes**: the SkyWalking operator, and
optionally its custom metrics adapter.

* **Operator** — reconciles the SWCK custom resources (`OAPServer`, `UI`, `Storage`, `BanyanDB`,
  `Satellite`, `SwAgent`, `JavaAgent`, `Fetcher`, `EventExporter`, `OAPServerConfig`,
  `OAPServerDynamicConfig`) and runs the Java agent injector. Always installed.
* **Custom metrics adapter** — an aggregated apiserver that answers
  `external.metrics.k8s.io` queries out of a SkyWalking OAP cluster, so a
  `HorizontalPodAutoscaler` can scale on SkyWalking metrics. Off by default; see
  [Metrics adapter](#metrics-adapter).

The chart is released from the same repository and at the same version as the operator it
deploys, and the CRDs, the manager's ClusterRole and the admission webhook configurations it
ships are generated from the operator sources by `make chart-manifests` — they cannot drift from
the operator binary.

## Supported SkyWalking versions

| Component | Supported | Recommended |
| --- | --- | --- |
| OAP | 10.4.0 and later | **11.0.0** |
| UI | Horizon 1.0.0 and later | **Horizon 1.0.0** |
| BanyanDB | matched to the OAP -- 0.11.x for OAP 11.0.0 | **0.11.0** |

These move together. OAP 11.0.0 accepts BanyanDB server API 0.11 only, and the Horizon UI reaches
OAP over an admin host that arrived in 11.x, so an older OAP is not a combination the Horizon UI
supports. An `OAPServer` below 10.4.0 is admitted with a warning rather than rejected -- a cluster
already running one keeps working.

The legacy Booster UI is not supported: `apache/skywalking` removed it in 11.0.0 and no longer
builds an image for it.

## Prerequisites

* Kubernetes 1.21+
* Helm 3.8+ (3.8 is where OCI registry support went GA)
* **cert-manager**, unless you set `operator.webhook.enabled=false`. The operator's admission
  webhooks and the Java agent injector are served over TLS, and the chart asks cert-manager for
  the certificate:

  ```shell
  kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.11.5/cert-manager.yaml
  kubectl wait --for=condition=Available -n cert-manager deployment --all --timeout=5m
  ```

## Install

From Docker Hub, which carries the combined image the chart's default values point at:

```shell
helm install skywalking-swck oci://docker.io/apache/skywalking-swck \
  --version 0.11.0-helm \
  --namespace skywalking-swck-system --create-namespace
```

From GHCR:

```shell
helm install skywalking-swck oci://ghcr.io/apache/skywalking-swck/helm/skywalking-swck \
  --version 0.11.0 \
  --namespace skywalking-swck-system --create-namespace \
  --set operator.image.repository=ghcr.io/apache/skywalking-swck/operator \
  --set operator.image.tag=0.11.0
```

The `--version` values differ on purpose, and the extra `--set` is not optional. See
[Registries](#registries).

From a checkout:

```shell
helm install skywalking-swck chart/skywalking-swck \
  --namespace skywalking-swck-system --create-namespace
```

Then create your first OAP cluster:

```shell
kubectl apply -f https://raw.githubusercontent.com/apache/skywalking-swck/master/operator/config/samples/default.yaml
```

## Registries

One release publishes the same chart to two registries, and the same operator as two different
kinds of image. Which one you install from changes what you have to set.

| Registry | Artifact | Coordinate at 0.11.0 |
| --- | --- | --- |
| Docker Hub | **combined** image, both `/manager` and `/adapter` | `apache/skywalking-swck:0.11.0` |
| Docker Hub | chart | `apache/skywalking-swck:0.11.0-helm` |
| GHCR | operator image, `/manager` only | `ghcr.io/apache/skywalking-swck/operator:0.11.0` |
| GHCR | metrics adapter image, `/adapter` only | `ghcr.io/apache/skywalking-swck/metrics-adapter:0.11.0` |
| GHCR | chart | `ghcr.io/apache/skywalking-swck/helm/skywalking-swck:0.11.0` |

* On Docker Hub the image and the chart share one repository, so they have to be told apart by
  tag — hence the `-helm` suffix on the chart. `0.11.0-helm` is a SemVer *pre-release*, which
  means it sorts *below* `0.11.0` and is skipped by version ranges: always request it with an
  exact `--version 0.11.0-helm`.
* The Docker Hub image carries both binaries, so `image.repository` alone is enough — the chart
  selects the binary per component with an explicit `command`.
* The GHCR images carry one binary each, so **neither can serve both components**. Pointing the
  chart at GHCR means setting `operator.image.*` and, if you enable it, `adapter.image.*`
  separately; the shared `image.repository` goes unused.
* GHCR also carries a development snapshot of every commit to master, tagged with the commit SHA
  and `latest`, with the chart at `0.0.0-<sha>`.

## Upgrade

```shell
helm upgrade skywalking-swck oci://docker.io/apache/skywalking-swck \
  --version <new version>-helm --namespace skywalking-swck-system
```

**Helm does not upgrade CRDs.** The CRDs live in the chart's `crds/` directory, which Helm
installs on first install and then never touches again — so an upgrade that adds a field to a
custom resource leaves the cluster with the old schema and the new operator unable to use it.
Apply them yourself as part of the upgrade:

```shell
helm show crds oci://docker.io/apache/skywalking-swck --version <new version>-helm \
  | kubectl apply --server-side --force-conflicts -f -
```

They are in `crds/` rather than in `templates/` deliberately: resources in `templates/` are
deleted by `helm uninstall`, and deleting a CRD deletes every custom resource of that kind in the
cluster along with it.

## Uninstall

```shell
helm uninstall skywalking-swck --namespace skywalking-swck-system
```

This leaves the CRDs, and therefore your `OAPServer`, `UI` and other custom resources, in place.
Remove them only if you mean to:

```shell
# The CRDs are generated by controller-gen and carry no chart labels, so select them by group.
kubectl get crd -o name | grep '\.operator\.skywalking\.apache\.org$' | xargs -r kubectl delete
```

## Java agent injector

The injector is the `/mutate-v1-pod` webhook, part of `operator.webhook.enabled`. It only touches
namespaces that opt in:

```shell
kubectl label namespace <your-namespace> swck-injection=enabled
```

See [docs/java-agent-injector.md](../../docs/en/setup/java-agent-injector.md).

## Metrics adapter

The adapter is off by default, for two reasons: it is useless without the address of a running
OAP cluster, and it claims `v1beta1.external.metrics.k8s.io`, a **cluster-wide singleton** that
only one external metrics provider can own. Enabling it in a cluster that already runs
prometheus-adapter or KEDA takes that API group away from them.

`adapter.oap.address`, or `adapter.oap.service.name`, is required — the chart refuses to render
without one. There is no safe default: the APIService reports `Available` whether or not the OAP
it points at exists, so a wrong address fails silently, with every HPA query simply returning
nothing.

```shell
helm install skywalking-swck oci://docker.io/apache/skywalking-swck \
  --version 0.11.0-helm \
  --namespace skywalking-swck-system --create-namespace \
  --set adapter.enabled=true \
  --set adapter.oap.service.name=my-oap-oap \
  --set adapter.oap.service.namespace=observability
```

The operator names the Service it creates `<OAPServer name>-oap`, in the namespace of the CR — so
an `OAPServer` named `my-oap` in namespace `observability` is reached as above. See
[docs/custom-metrics-adapter.md](../../docs/en/setup/custom-metrics-adapter.md).

## Configuration

| Key | Description | Default |
| --- | --- | --- |
| `nameOverride` | Overrides the chart name in resource names | `""` |
| `fullnameOverride` | Overrides the full name prefix of every resource | `""` |
| `image.repository` | Image repository shared by both components | `docker.io/apache/skywalking-swck` |
| `image.tag` | Shared image tag; empty means the chart's `appVersion` | `""` |
| `image.pullPolicy` | Shared image pull policy | `IfNotPresent` |
| `operator.replicas` | Operator replicas | `1` |
| `operator.image.repository` | Operator image repository; empty falls back to `image.repository` | `""` |
| `operator.image.tag` | Operator image tag; empty falls back to `image.tag` | `""` |
| `operator.image.pullPolicy` | Operator image pull policy; empty falls back to `image.pullPolicy` | `""` |
| `operator.serviceAccount.name` | Use an existing service account instead of creating one | `""` |
| `operator.securityContext` | securityContext of the manager container | `{allowPrivilegeEscalation: false}` |
| `operator.podSecurityContext` | securityContext of the operator pod | `{runAsNonRoot: true}` |
| `operator.metrics.enabled` | Serve operator metrics through a kube-rbac-proxy sidecar | `true` |
| `operator.metrics.service.port` | Port the sidecar listens on | `8443` |
| `operator.metrics.kubeRbacProxy.image.repository` | kube-rbac-proxy image | `quay.io/brancz/kube-rbac-proxy` |
| `operator.metrics.kubeRbacProxy.image.tag` | kube-rbac-proxy tag | `v0.18.1` |
| `operator.metrics.kubeRbacProxy.image.pullPolicy` | kube-rbac-proxy pull policy | `IfNotPresent` |
| `operator.metrics.kubeRbacProxy.resources` | kube-rbac-proxy resources | `{}` |
| `operator.webhook.enabled` | Admission webhooks and the Java agent injector; needs cert-manager | `true` |
| `operator.webhook.service.port` | Port the webhook server listens on | `9443` |
| `operator.resources` | Operator resource requests and limits | `200m` / `300Mi`, both |
| `operator.nodeSelector` | Operator node selector | `{}` |
| `operator.tolerations` | Operator tolerations | `[]` |
| `operator.affinity` | Operator affinity | `{}` |
| `adapter.enabled` | Install the custom metrics adapter | `false` |
| `adapter.replicas` | Adapter replicas | `1` |
| `adapter.image.repository` | Adapter image repository; empty falls back to `image.repository` | `""` |
| `adapter.image.tag` | Adapter image tag; empty falls back to `image.tag` | `""` |
| `adapter.image.pullPolicy` | Adapter image pull policy; empty falls back to `image.pullPolicy` | `""` |
| `adapter.serviceAccount.name` | Use an existing service account instead of creating one | `""` |
| `adapter.service.port` | Port the adapter's apiserver listens on | `6443` |
| `adapter.logLevel` | Adapter klog verbosity | `10` |
| `adapter.oap.address` | Full OAP GraphQL endpoint; wins over `oap.service.*` | `""` |
| `adapter.oap.service.name` | Name of the OAP Service; **required** unless `oap.address` is set | `""` |
| `adapter.oap.service.namespace` | Namespace of the OAP Service; empty means the release namespace | `""` |
| `adapter.oap.service.port` | Port of the OAP Service | `12800` |
| `adapter.resources` | Adapter resource requests and limits | `{}` |
| `adapter.nodeSelector` | Adapter node selector | `{}` |
| `adapter.tolerations` | Adapter tolerations | `[]` |
| `adapter.affinity` | Adapter affinity | `{}` |

See [values.yaml](values.yaml) — the comments there carry the reasoning behind the defaults.
