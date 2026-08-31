# Install with Helm

`skywalking-swck` is the Helm chart for SWCK. One chart installs the operator and, behind a values
flag, the custom metrics adapter. It is released from this repository at the same version as the
operator it deploys, and the CRDs, the operator's ClusterRole and the admission webhook
configurations it ships are generated from the operator sources — they cannot drift from the
operator binary.

The chart's own reference — every value, with the reasoning behind each default — is
[`chart/skywalking-swck/README.md`](../../../chart/skywalking-swck/README.md).

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
* Helm 3.8+ (OCI registry support went GA in 3.8)
* **cert-manager**, unless you turn the webhooks off. The operator's admission webhooks and the
  Java agent injector are served over TLS, and the chart asks cert-manager for the certificate:

  ```shell
  kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.11.5/cert-manager.yaml
  kubectl wait --for=condition=Available -n cert-manager deployment --all --timeout=5m
  ```

## Install

```shell
helm install skywalking-swck oci://docker.io/apache/skywalking-swck \
  --version 0.11.0-helm \
  --namespace skywalking-swck-system --create-namespace
```

The `-helm` suffix is not a typo — see [Where the chart is published](#where-the-chart-is-published).

Then create an OAP cluster:

```shell
kubectl apply -f https://raw.githubusercontent.com/apache/skywalking-swck/master/operator/config/samples/default.yaml
```

and follow [Deploy OAP server and UI](../examples/default-backend.md) from there.

## Where the chart is published

One release publishes the same chart to two registries, and the operator as two different kinds of
image.

| Registry | Artifact | Coordinate at 0.11.0 |
| --- | --- | --- |
| Docker Hub | **combined** image, both `/manager` and `/adapter` | `apache/skywalking-swck:0.11.0` |
| Docker Hub | chart | `apache/skywalking-swck:0.11.0-helm` |
| GHCR | operator image, `/manager` only | `ghcr.io/apache/skywalking-swck/operator:0.11.0` |
| GHCR | metrics adapter image, `/adapter` only | `ghcr.io/apache/skywalking-swck/metrics-adapter:0.11.0` |
| GHCR | chart | `ghcr.io/apache/skywalking-swck/helm/skywalking-swck:0.11.0` |
| dist.apache.org | chart tarball, signed and voted | `skywalking-swck-0.11.0.tgz` |

On Docker Hub the image and the chart share one repository, so they are told apart by tag — hence
`-helm`. It is a SemVer *pre-release*, which means it sorts *below* `0.11.0` and is skipped by
version ranges: always request it with an exact `--version 0.11.0-helm`.

Installing from GHCR needs two extra `--set`s, because the GHCR images carry one binary each and
neither can serve both components:

```shell
helm install skywalking-swck oci://ghcr.io/apache/skywalking-swck/helm/skywalking-swck \
  --version 0.11.0 \
  --namespace skywalking-swck-system --create-namespace \
  --set operator.image.repository=ghcr.io/apache/skywalking-swck/operator \
  --set operator.image.tag=0.11.0
```

GHCR also carries a snapshot of every commit to master, tagged with the commit SHA and `latest`,
with the chart at `0.0.0-<sha>`.

## The metrics adapter

Off by default, and it needs the address of a running OAP cluster — the chart refuses to render
without one:

```shell
helm install skywalking-swck oci://docker.io/apache/skywalking-swck \
  --version 0.11.0-helm \
  --namespace skywalking-swck-system --create-namespace \
  --set adapter.enabled=true \
  --set adapter.oap.service.name=my-oap-oap \
  --set adapter.oap.service.namespace=observability
```

The operator names the Service it creates `<OAPServer name>-oap`, in the namespace of the CR. There
is no default because a wrong address fails silently: the APIService reports `Available` whether or
not the OAP it points at exists, so every HPA query just returns nothing.

The adapter also claims `v1beta1.external.metrics.k8s.io`, a **cluster-wide singleton** that only
one external metrics provider can own — enabling it in a cluster that already runs
prometheus-adapter or KEDA takes that API group away from them. See
[Custom metrics adapter](custom-metrics-adapter.md).

## Upgrade

```shell
helm upgrade skywalking-swck oci://docker.io/apache/skywalking-swck \
  --version <new version>-helm --namespace skywalking-swck-system
```

**Helm does not upgrade CRDs.** They live in the chart's `crds/` directory, which Helm installs on
first install and then never touches — so an upgrade that adds a field to a custom resource leaves
the cluster on the old schema. Apply them yourself:

```shell
helm show crds oci://docker.io/apache/skywalking-swck --version <new version>-helm \
  | kubectl apply --server-side --force-conflicts -f -
```

They are in `crds/` rather than `templates/` on purpose: resources in `templates/` are deleted by
`helm uninstall`, and deleting a CRD deletes every custom resource of that kind in the cluster with
it.

## Uninstall

```shell
helm uninstall skywalking-swck --namespace skywalking-swck-system
```

Your `OAPServer`, `UI` and other custom resources survive, because the CRDs do.

## Or install with kustomize

The chart is not the only way in. `operator/config` is a kustomize bundle, it is what
`make -C operator install && make -C operator deploy` applies, and the binary release ships it as
`config/operator-bundle.yaml` and `config/adapter-bundle.yaml`. See
[Operator](operator.md#deploy-the-operator).
