Apache SkyWalking Cloud on Kubernetes
============

![](https://github.com/apache/skywalking-swck/workflows/Build/badge.svg?branch=master)

<img src="https://skywalking.apache.org/assets/logo.svg" alt="Sky Walking logo" height="90px" align="right" />

A bridge project between [Apache SkyWalking](https://github.com/apache/skywalking) and Kubernetes.

SWCK is a platform for the SkyWalking user that provisions, upgrades, maintains SkyWalking relevant components, and makes them work natively on Kubernetes.

# Features

* Java Agent Injector: Inject the java agent into the application pod natively.
  * Inject the java agent into the application pod.
  * Leverage a global configuration to simplify the agent and injector setup.
  * Use the annotation to customize specific workloads.
  * Synchronize injecting status to `JavaAgent` CR for monitoring purposes.
* Operator: Provision and maintain SkyWalking backend components.
* Custom Metrics Adapter: Provides custom metrics coming from SkyWalking OAP cluster for autoscaling by Kubernetes HPA

# Quick Start

* Go to the [download page](https://skywalking.apache.org/downloads/#SkyWalkingCloudonKubernetes) to download the latest release binary, `skywalking-swck-<SWCK_VERSION>-bin.tgz`. Unarchive the package to
a folder named `skywalking-swck-<SWCK_VERSION>-bin`

## Java Agent Injector

* Install the [Operator](#operator)
* Label the namespace with `swck-injection=enabled`

```shell
$ kubectl label namespace default(your namespace) swck-injection=enabled
```

* Add label `swck-java-agent-injected: "true"` to the workloads

For more details, please read [Java agent injector](/docs/java-agent-injector.md)

## Operator

* To install the operator in an existing cluster, ensure you have [`cert-manager`](https://cert-manager.io/docs/installation/) installed.
* Apply the manifests for the Controller and CRDs in release/config:
 
 ```
 kubectl apply -f skywalking-swck-<SWCK_VERSION>-bin/config/operator-bundle.yaml
 ```

For more details, please refer to [deploy operator](docs/operator.md)

## Custom Metrics Adapter
  
* Deploy the OAP server by referring to Operator Quick Start.
* Apply the manifests for an adapter in release/adapter/config:
 
 ```
 kubectl apply -f skywalking-swck-<SWCK_VERSION>-bin/config/adapter-bundle.yaml
 ```

For more details, please read [Custom metrics adapter](docs/custom-metrics-adapter.md)

# Contributing
For developers who want to contribute to this project, see [Contribution Guide](CONTRIBUTING.md). What's more, we have a guide about how to add new CRDs and Controllers, see [How to add new CRD and Controller in SWCK](docs/how-to-add-new-crd-and-controller.md).

# License
[Apache 2.0 License.](/LICENSE)
