# Operator Usage Guide

In this guide, you will learn:

* How to deploy the operator from a released package or scratch
* The core CRDs the operator supports

## Operator Deployment

You could provision the operator from a binary package or build from sources.

### Binary Package

* Go to the [download page](https://skywalking.apache.org/downloads/#SkyWalkingCloudonKubernetes) to download the latest release binary, `skywalking-swck-<SWCK_VERSION>-bin.tgz`. Unarchive the package to
a folder named `skywalking-swck-<SWCK_VERSION>-bin`
* To install the operator in an existing cluster, make sure you have [`cert-manager`](https://cert-manager.io/docs/installation/) installed.
* Apply the manifests for the Controller and CRDs in `config`:

```sh
kubectl apply -f skywalking-swck-<SWCK_VERSION>-bin/config/operator-bundle.yaml
```

### Build from sources

1.  Download released [source package](https://skywalking.apache.org/downloads/#SkyWalkingCloudonKubernetes) or clone the source code:

```sh
git clone git@github.com:apache/skywalking-swck.git
```

1. Build docker image from scratch. If you prefer to your private
docker image, a quick path to override `OPERATOR_IMG` environment variable : `export OPERATOR_IMG=<private registry>/controller:<tag>`

```sh
export OPERATOR_IMG=controller
make -C operator docker-build
```

Then, push this image `controller:latest` to a repository where the operator's pod could pull from.
If you use a local `KinD` cluster:

```sh
kind load docker-image controller
```

1. Customize resource configurations based the templates laid in `operator/config`. We use `kustomize` to build them, please refer to [kustomize](https://kustomize.io/) in case you don't familiar with its syntax.

1. Install the CRDs to Kubernetes:

```sh
make -C operator install
```

1. Use `make` to generate the final manifests and deploy:

```sh
make -C operator deploy
```

### Test your deployment

1. Deploy a sample OAP server, this will create an OAP server in the default namespace:

```sh
curl https://raw.githubusercontent.com/apache/skywalking-swck/master/operator/config/samples/default.yaml | kubectl apply -f -
```

1. Check the OAP server in Kubernetes:

```sh
kubectl get oapserver
```

1. Check the UI server in Kubernetes:

```sh
kubectl get ui
```

### Troubleshooting

If you encounter any issue, you can check the log of the controller by pulling it from Kubernetes:

```sh
# get the pod name of your controller
kubectl --namespace skywalking-swck-system get pods

# pull the logs
kubectl --namespace skywalking-swck-system logs -f [name_of_the_controller_pod]
```

## Custom Resource Define(CRD)

The custom resources that the operator introduced are:

### JavaAgent

The `JavaAgent` custom resource definition (CRD) declaratively defines a view to tracing the injection result.  
The [java-agent-injector](java-agent-injector.md) creat JavaAgents once it injects agents into some workloads.
Refer to [Java Agent](./javaagent.md) for more details.

### OAP

The `OAP` custom resource definition (CRD) declaratively defines a desired OAP setup to run in a Kubernetes cluster.
It provides options to configure environment variables and how to connect a `Storage`.

### UI

The `UI` custom resource definition (CRD) declaratively defines a desired UI setup to run in a Kubernetes cluster.
It provides options for how to connect an `OAP`.

### Storage

The `Storage` custom resource definition (CRD) declaratively defines a desired storage setup to run in a Kubernetes cluster.
The `Storage` could be managed instances onboarded by the operator or an external service. The `OAP` has options to select
which `Storage` it would connect.

> Caveat: `Stroage` only supports the `Elasticsearch`.

### Satellite

The `Satellite` custom resource definition (CRD) declaratively defines a desired Satellite setup to run in a Kubernetes cluster.
It provides options for how to connect an `OAP`.

### Fetcher

The `Fetcher` custom resource definition (CRD) declaratively defines a desired Fetcher setup to run in a Kubernetes cluster.
It provides options to configure OpenTelemetry collector, which fetches metrics to the deployed `OAP`.

## Examples of the Operator

There are some instant examples to represent the functions or features of the Operator.

* [Deploy OAP server and UI with default settings](./examples/default-backend.md)
* [Fetch metrics from the Istio control plane(istiod)](./examples/istio-controlplane.md)
* [Inject the java agent to pods](./examples/java-agent-injector-usage.md)
* [Deploy a storage](./examples/storage.md)
* [Deploy a Satellite](./examples/satellite.md)
