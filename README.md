Apache SkyWalking Cloud on Kubernetes
============

![](https://github.com/apache/skywalking-swck/workflows/Build/badge.svg?branch=master)

<img src="http://skywalking.apache.org/assets/logo.svg" alt="Sky Walking logo" height="90px" align="right" />

A bridge project between [Apache SkyWalking](https://github.com/apache/skywalking) and Kubernetes.

SWCK is a platform for the SkyWalking user, provisions, upgrades, maintains SkyWalking relevant components, and makes them work natively on Kubernetes. 

# Quick Start

 1. Go to the [download page](https://skywalking.apache.org/downloads/) to download latest release manifest. 
 
 2. Apply the manifests for the Controller and CRDs in release/config:
 ```
 kubectl apply -f release/config
 ```
 
# Guides of Deployment
## Use kustomize to customise your deployment

1. Clone the source code:

```sh
git clone git@github.com:apache/skywalking-swck.git
```

2. Edit file `config/default/kustomization.yaml` file to change your preferences. If you prefer to your private docker image, a quick path to override `IMG` environment variable : `export IMG=<private registry>/controller:<tag>`

3. Use `make` to generate the final manifests and deploy:

```sh
make deploy
```

4. Deploy the CRDs:

```sh
make install
```

## Test your deployment

1. Deploy a sample OAP server, this will create a OAP server in the default namespace:

```sh
curl https://raw.githubusercontent.com/apache/skywalking-swck/master/config/samples/oap.yaml | kubectl apply -f -
```

2. Check the OAP server in Kubernetes:

```sh
kubectl get oapserver
```

## Troubleshooting

If you encounter any issue, you can check the log of the controller by pulling it from Kubernetes:

```sh
# get the pod name of your controller
kubectl --namespace skywalking-swck-system get pods

# pull the logs
kubectl --namespace skywalking-swck-system logs -f [name_of_the_controller_pod]
```

# Contributing
For developers who want to contribute to this project, see [Contribution Guide](CONTRIBUTING.md)

# License
[Apache 2.0 License.](/LICENSE)
