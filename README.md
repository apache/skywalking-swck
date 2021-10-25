Apache SkyWalking Cloud on Kubernetes
============

![](https://github.com/apache/skywalking-swck/workflows/Build/badge.svg?branch=master)

<img src="http://skywalking.apache.org/assets/logo.svg" alt="Sky Walking logo" height="90px" align="right" />

A bridge project between [Apache SkyWalking](https://github.com/apache/skywalking) and Kubernetes.

SWCK is a platform for the SkyWalking user, provisions, upgrades, maintains SkyWalking relevant components, and makes them work natively on Kubernetes. 

# Features

 1. Java Agent Injector: Inject the java agent into the application pod natively.
 1. Operator: Provision and maintain SkyWalking backend components.
 1. Custom Metrics Adapter: Provides custom metrics come from SkyWalking OAP cluster for autoscaling by Kubernetes HPA

# Quick Start

 * Go to the [download page](https://skywalking.apache.org/downloads/#SkyWalkingCloudonKubernetes) to download latest release manifest. 

## Java Agent Injector

The java agent injector share the same binary with the operator. Follow the installation procedure of the operator
to onboard the injector.

The injector can:

* Inject the java agent into the application pod.
* Leverage a global configuration to simplify the agent and injector setup.
* Use the annotation to customize specific workloads.
* Sync injecting status to `JavaAgent` CR for monitoring purpose.

For more details, please read [Java agent injector](docs/java-agent-injector.md)

## Operator

 * To install the operator in an existing cluster, make sure you have [`cert-manager` installed](https://cert-manager.io/docs/installation/)
 * Apply the manifests for the Controller and CRDs in release/config:
 
 ```
 kubectl apply -f release/operator/config
 ```

For more details, please refer to [deploy operator](docs/operator.md)

## Custom Metrics Adapter
  
 * Deploy OAP server by referring to Operator Quick Start.
 * Apply the manifests for an adapter in release/adapter/config:
 
 ```
 kubectl apply -f release/adapter/config
 ```

For more details, please read [Custom metrics adapter](docs/custom-metrics-adapter.md)

# Contributing
For developers who want to contribute to this project, see [Contribution Guide](CONTRIBUTING.md)

# License
[Apache 2.0 License.](/LICENSE)
