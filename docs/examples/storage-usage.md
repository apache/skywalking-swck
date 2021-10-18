# Storage Operator Usage

The Storage Operator can manage and monitor storage pods(ES, MySQL, and etc.). 
We provide external and internal configuration methods to deploy Storage.
When you deployment the Storage,you also need to specify the Storage Name in the OAP server configuration。

## Install Operator

Follow [Operator installation instrument](../../README.md#operator) to install the operator.

## Storage Configuration

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

## Define Storage with default setting

1. sample.yaml(use the internal type)
```yaml
apiVersion: operator.skywalking.apache.org/v1alpha1
kind: Storage
metadata:
  name: sample
spec:
  type: elasticsearch7
  connectType: internal
  version: 7.5.1
  instances: 3
  image: docker.elastic.co/elasticsearch/elasticsearch:7.5.1
  security:
    user:
      secretName: default
    tls: true
```
2. sample.yaml(use the external type)
```yaml
apiVersion: operator.skywalking.apache.org/v1alpha1
kind: Storage
metadata:
  name: sample
spec:
  type: elasticsearch7
  connectType: external
  address: "https://elasticsearch"
  security:
    user:
      secretName: default
```

## Deploy Storage

1. Deploy the Storage

```sh
kubectl apply -f sample.yaml
```

2. Check the Storage in Kubernetes:

```sh
kubectl get storage
```

## Specify Storage Name in OAP server

Here we modify the [default OAP server configuration file](../../config/operator/samples/default.yaml),
Then you need to redeploy the OAP server.


```yaml
apiVersion: operator.skywalking.apache.org/v1alpha1
kind: OAPServer
metadata:
  name: default
spec:
  version: 8.3.0
  instances: 1
  image: apache/skywalking-oap-server:8.3.0-es7
  service:
    template:
      type: ClusterIP
  storage:
    name: sample
```
