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

## Storage Operator Introduction

The Storage Operator can manage and monitor storage pods(ES, MySQL, and etc.).
We provide external and internal configuration methods to deploy Storage.
When you deployment the Storage,you also need to specify the Storage Name in the OAP server configuration。

### Storage Configuration

| Field Name         | Description                                                  |
| ------------------ | ------------------------------------------------------------ |
| type               | specify the database type(elasticsearch7,mysql,and etc.).|
| connectType        | specify how to connect to storage(use external or internal).|
| address     | specify connection address of database(when you use an external configuration method,you need to set).|
| version            | specify the version of storage.|      
| image              | specify the docker image name of database.|
| instances          | specify the number of storage instance.|
| security           | set oapserver and storage secure connection configuration.|  
| security.user.secretName  | enable the user authentication of the storage connection,you need to configure the user name and password in the secret,and specify the secret name here.if you specify the secret name as default,this will use the default username(elastic) and password(changeme).| 
| security.tls       | enable the tls authentication of the storage connection,you can set true or false to indicate whether the authentication is open.|

## Custom manifests templates

If you want to custom the manifests templates to generate dedicated Kubernetes resources,
please edit YAMLs in `pkg/operator/manifests`.
After saving your changes, issue `make update-templates` to transfer them to binary assets.
The last step is to rebuild `operator` by `make operator-docker-build`.
