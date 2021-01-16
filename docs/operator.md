# Guides of Operator Deployment
## Use kustomize to customise your deployment

1. Clone the source code:

```sh
git clone git@github.com:apache/skywalking-swck.git
```

2. Edit file `config/operator/default/kustomization.yaml` file to change your preferences. If you prefer to your private
 docker image, a quick path to override `OPERATOR_IMG` environment variable : `export OPERATOR_IMG=<private registry>/controller:<tag>`

3. Use `make` to generate the final manifests and deploy:

```sh
make operator-deploy
```

4. Deploy the CRDs:

```sh
make operator-install
```

## Test your deployment

1. Deploy a sample OAP server, this will create an OAP server in the default namespace:

```sh
curl https://raw.githubusercontent.com/apache/skywalking-swck/master/config/operator/samples/default.yaml | kubectl apply -f -
```

2. Check the OAP server in Kubernetes:

```sh
kubectl get oapserver
```

2. Check the UI server in Kubernetes:

```sh
kubectl get ui
```

## Troubleshooting

If you encounter any issue, you can check the log of the controller by pulling it from Kubernetes:

```sh
# get the pod name of your controller
kubectl --namespace skywalking-swck-system get pods

# pull the logs
kubectl --namespace skywalking-swck-system logs -f [name_of_the_controller_pod]
```


## Custom manifests templates

If you want to custom the manifests templates to generate dedicated Kubernetes resources,
please edit YAMLs in `pkg/operator/manifests`.
After saving your changes, issue `make update-templates` to transfer them to binary assets.
The last step is to rebuild `operator` by `make operator-docker-build`.
