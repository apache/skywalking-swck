# Overview

SkyWalking Cloud on Kubernetes (SWCK) provisions, upgrades and maintains SkyWalking components on
Kubernetes, and makes them work natively there. It is made of three things.

## The operator

A Kubernetes operator that reconciles custom resources into running SkyWalking components. Instead
of writing and maintaining Deployments, Services, ConfigMaps and Ingresses for an OAP cluster, you
declare an `OAPServer` and the operator maintains the rest.

| Custom resource | What it manages |
| --- | --- |
| `OAPServer` | An OAP cluster: its Deployment, Service, and the configuration it starts with |
| `UI` | The SkyWalking (Horizon) UI |
| `Storage` | The OAP storage — Elasticsearch provisioned in the cluster, or an existing one |
| `BanyanDB` | A BanyanDB cluster as the OAP storage |
| `Satellite` | A Satellite collector in front of OAP |
| `SwAgent` | How the agent injector should build the sidecar for a set of workloads |
| `JavaAgent` | A record of an injection, created by the injector rather than by you |
| `Fetcher` | Pulling metrics from another telemetry system, such as the Istio control plane |
| `OAPServerConfig` | Static configuration handed to a running OAP cluster |
| `OAPServerDynamicConfig` | Dynamic configuration handed to a running OAP cluster |
| `EventExporter` | Exporting Kubernetes events into SkyWalking |

See [Operator](../setup/operator.md).

## The Java agent injector

An admission webhook, part of the operator, that adds the SkyWalking Java agent to pods as a sidecar
— no rebuilt images, no changed Dockerfiles. It only touches namespaces that opt in with the
`swck-injection=enabled` label, and pods that opt in with the `swck-java-agent-injected=true` label.
What it injects is driven by annotations on the pod and by `SwAgent` resources.

See [Java agent injector](../setup/java-agent-injector.md).

## The custom metrics adapter

An aggregated apiserver that answers `external.metrics.k8s.io` queries out of a SkyWalking OAP
cluster, so a `HorizontalPodAutoscaler` can scale on a SkyWalking metric — service CPM, endpoint
latency — rather than on CPU.

It is optional, and off by default in the Helm chart: it needs the address of a running OAP cluster,
and it claims a cluster-wide API group that only one external metrics provider can own.

See [Custom metrics adapter](../setup/custom-metrics-adapter.md).

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

## How they are shipped

| Artifact | Where |
| --- | --- |
| Source and binary distributions | [dist.apache.org](https://downloads.apache.org/skywalking/swck/), signed and voted |
| Helm chart | dist.apache.org, Docker Hub and GHCR |
| Combined image, `/manager` + `/adapter` | `docker.io/apache/skywalking-swck` |
| Split images, one binary each | `ghcr.io/apache/skywalking-swck/{operator,metrics-adapter}` |

The Helm chart is the shortest path in — see [Install with Helm](../setup/helm-chart.md). The
kustomize bundle in `operator/config` is the other, and ships in the binary distribution.
