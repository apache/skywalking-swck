# SkyWalking Kubernetes Event Exporter User Guide

[SkyWalking Kubernetes Event Exporter](https://github.com/apache/skywalking-kubernetes-event-exporter) is able to watch,
filter, and send Kubernetes events into the Apache SkyWalking backend.

## Demo

### Step 1: Create a Local Kubernetes Cluster

Please follow step 1 to 3 in [getting started](../getting-started.md) to create a cluster.

### Step 2: Deploy OAP server and Event Exporter

Create the `skywalking-system` namespace.

```shell
$ kubectl create namespace skywalking-system
```

Deploy an OAP server and an event exporter.

```shell
cat <<EOF | kubectl apply -f -
apiVersion: operator.skywalking.apache.org/v1alpha1
kind: OAPServer
metadata:
  name: skywalking-system
  namespace: skywalking-system
spec:
  version: 9.5.0
  instances: 1
  image: apache/skywalking-oap-server:9.5.0
  service:
    template:
      type: ClusterIP

---
apiVersion: operator.skywalking.apache.org/v1alpha1
kind: EventExporter
metadata:
  name: skywalking-system
  namespace: skywalking-system
spec:
  replicas: 1
  config: |
    filters:
      - reason: ""     
        message: ""    
        minCount: 1    
        type: ""       
        action: ""     
        kind: "Pod|Service"
        namespace: "^skywalking-system$"  
        name: ""       
        service: "[^\\s]{1,}"  
        exporters:     
          - skywalking 
    exporters:         
      skywalking:      
        template:      
          source:
            service: "{{ .Service.Name }}"
            serviceInstance: "{{ .Pod.Name }}"
            endpoint: ""
          message: "{{ .Event.Message }}" 
        address: "skywalking-system-oap.skywalking-system:11800"
EOF
```

Wait until both components are ready...

```shell
$ kubectl get pod -n skywalking-system 
NAME                                               READY   STATUS    RESTARTS   AGE
skywalking-system-eventexporter-566db46fb6-npx8v   1/1     Running   0          50s
skywalking-system-oap-68bd877f57-zs8hw             1/1     Running   0          50s
```

### Step 3: Check Reported Events

We can verify k8s events is reported to the OAP server by
using [skywalking-cli](https://github.com/apache/skywalking-cli).

First, port-forward the OAP http service to your local machine.

```shell
$ kubectl port-forward svc/skywalking-system-oap 12800:12800 -n skywalking-system
```

Next, use `swctl` to list reported events in YAML format.

```shell
$ swctl --display yaml event ls
```

The output should contain k8s events of the OAP server.

```yaml
events:
  - uuid: 1d5bfe48-bc8d-4f5a-9680-188f59793459
    source:
      service: skywalking-system-oap
      serviceinstance: skywalking-system-oap-68bd877f57-cvkjb
      endpoint: ""
    name: Pulled
    type: Normal
    message: Successfully pulled image "apache/skywalking-oap-server:9.5.0" in 6m4.108914335s
    parameters: [ ]
    starttime: 1713793327000
    endtime: 1713793327000
    layer: K8S
  - uuid: f576f6ad-748d-4cec -9260-6587c145550e
    source:
      service: skywalking-system-oap
      serviceinstance: skywalking-system-oap-68bd877f57-cvkjb
      endpoint: ""
    name: Created
    type: Normal
    message: Created container oap
    parameters: [ ]
    starttime: 1713793327000
    endtime: 1713793327000
    layer: K8S
  - uuid: 0cec5b55-4cb0-4ff7-a670-a097609c531f
    source:
      service: skywalking-system-oap
      serviceinstance: skywalking-system-oap-68bd877f57-cvkjb
      endpoint: ""
    name: Started
    type: Normal
    message: Started container oap
    parameters: [ ]
    starttime: 1713793327000
    endtime: 1713793327000
    layer: K8S
  - uuid: 28f0d004-befe-4c27-a7b7-dfdc4dd755fa
    source:
      service: skywalking-system-oap
      serviceinstance: skywalking-system-oap-68bd877f57-cvkjb
      endpoint: ""
    name: Pulling
    type: Normal
    message: Pulling image "apache/skywalking-oap-server:9.5.0"
    parameters: [ ]
    starttime: 1713792963000
    endtime: 1713792963000
    layer: K8S
  - uuid: 6d766801-5057-42c0-aa63-93ce1e201418
    source:
      service: skywalking-system-oap
      serviceinstance: skywalking-system-oap-68bd877f57-cvkjb
      endpoint: ""
    name: Scheduled
    type: Normal
    message: Successfully assigned skywalking-system/skywalking-system-oap-68bd877f57-cvkjb
      to kind-worker
    parameters: [ ]
    starttime: 1713792963000
    endtime: 1713792963000
    layer: K8S
```

We can also verify by checking logs of the event exporter.

```shell
kubectl logs -f skywalking-system-eventexporter-566db46fb6-npx8v -n skywalking-system
...
DEBUG done: rendered event is: uuid:"8d8c2bd1-1812-4b0c-8237-560688366280" source:{service:"skywalking-system-oap" serviceInstance:"skywalking-system-oap-68bd877f57-zs8hw"} name:"Started" message:"Started container oap" startTime:1713795214000 endTime:1713795214000 layer:"K8S"
```

## Spec

| name     | description                                            | default value                                        |
|----------|--------------------------------------------------------|------------------------------------------------------|
| image    | Docker image of the event exporter.                    | `apache/skywalking-kubernetes-event-exporter:latest` |                                                                                                           | `"skywalking-system-oap.skywalking-system:11800"`                       |
| replicas | Number of event exporter pods.                         | `1`                                                  |
| config   | Configuration of filters and exporters in YAML format. | `""`                                                 |

Please note: if you ignore the `config` field, no filters or exporter will be created.

This is because the EventExporter controller creates a configMap for all `config` values and
attach the configMap to the event exporter container as configuration file.
Ignoring the `config` field means an **empty** configuration file (with content `""`) is provided to the event exporter.

## Status

| name              | description                                                                |
|-------------------|----------------------------------------------------------------------------|
| availableReplicas | Total number of available event exporter pods.                             |
| conditions        | Latest available observations of the underlying deployment's current state |
| configMapName     | Name of the underlying configMap.                                          |

## Configuration

The event exporter supports reporting specific events by different exporters.
We can add filter configs to choose which events we are interested in,
and include exporter names in each filter config to tell event exporter how to export filtered events.

An example configuration is listed below:

```yaml
filters:
  - reason: ""
    message: ""
    minCount: 1
    type: ""
    action: ""
    kind: "Pod|Service"
    namespace: "^default$"
    name: ""
    service: "[^\\s]{1,}"
    exporters:
      - skywalking

exporters:
  skywalking:
    template:
      source:
        service: "{{ .Service.Name }}"
        serviceInstance: "{{ .Pod.Name }}"
        endpoint: ""
      message: "{{ .Event.Message }}"
    address: "skywalking-system-oap.skywalking-system:11800" 
```

### Filter Config

| name      | description                                                                                                                         | example          |
|-----------|-------------------------------------------------------------------------------------------------------------------------------------|------------------|
| reason    | Filter events of the specified reason, regular expression like `"Killing\|Killed"` is supported.                                    | `""`             |                                                                                                           | `"skywalking-system-oap.skywalking-system:11800"`                       |
| message   | Filter events of the specified message, regular expression like `"Pulling container.*"` is supported.                               | `""`             |
| minCount  | Filter events whose count is >= the specified value.                                                                                | `1`              |
| type      | Filter events of the specified type, regular expression like `"Normal\|Error"` is supported.                                        | `""`             |
| action    | Filter events of the specified action, regular expression is supported.                                                             | `""`             |
| kind      | Filter events of the specified kind, regular expression like `"Pod\|Service"` is supported.                                         | `"Pod\|Service"` |
| namespace | Filter events from the specified namespace, regular expression like `"default\|bookinfo"` is supported, empty means all namespaces. | `"^default$"`    |
| name      | Filter events of the specified involved object name, regular expression like `".*bookinfo.*"` is supported.                         | `""`             |
| service   | Filter events belonging to services whose name is not empty.                                                                        | `"[^\\s]{1,}"`   |
| exporters | Events satisfy this filter can be exported into several exporters that are defined below.                                           | `["skywalking"]` |

### Skywalking Exporter Config

SkyWalking exporter exports the events into Apache SkyWalking OAP server using grpc.

| name                            | description                                                                                                                                                                  | example                                           |
|---------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------------------------------|
| address                         | The SkyWalking backend address where this exporter will export to.                                                                                                           | `"skywalking-system-oap.skywalking-system:11800"` |
| enableTLS                       | Whether to use TLS for grpc server connection validation. <br/> If TLS is enabled, the `trustedCertPath` is required, but `clientCertPath` and `clientKeyPath` are optional. | `false`                                           |
| clientCertPath                  | Path of the X.509 certificate file.                                                                                                                                          | `""`                                              |
| clientKeyPath                   | Path of the X.509 private key file.                                                                                                                                          | `""`                                              |
| trustedCertPath                 | Path of the root certificate file.                                                                                                                                           | `""`                                              |
| insecureSkipVerify              | Whether a client verifies the server's certificate chain and host name. Check [`tls.Config`](https://pkg.go.dev/crypto/tls#Config) for more details.                         | `false`                                           |
| template                        | The event template of SkyWalking exporter, it can be composed of metadata like Event, Pod, and Service.                                                                      |                                                   |
| template.source                 | Event source information.                                                                                                                                                    |                                                   |
| template.source.service         | Service name, can be a [template string](https://pkg.go.dev/text/template).                                                                                                  | `"{{ .Service.Name }}"`                           |
| template.source.serviceInstance | Service instance name, can be a [template string](https://pkg.go.dev/text/template).                                                                                         | `"{{ .Pod.Name }}"`                               |
| template.source.endpoint        | Endpoint, can be a [template string](https://pkg.go.dev/text/template).                                                                                                      | `""`                                              |
| template.message                | Message format, can be a [template string](https://pkg.go.dev/text/template).                                                                                                | `"{{ .Event.Message }}"`                          |

### Console Exporter Config

Console exporter exports the events into console logs, this exporter is typically used for debugging.

| name                            | description                                                                                             | example                  |
|---------------------------------|---------------------------------------------------------------------------------------------------------|--------------------------|
| template                        | The event template of SkyWalking exporter, it can be composed of metadata like Event, Pod, and Service. |                          |
| template.source                 | Event source information.                                                                               |                          |
| template.source.service         | Service name, can be a [template string](https://pkg.go.dev/text/template).                             | `"{{ .Service.Name }}"`  |
| template.source.serviceInstance | Service instance name, can be a [template string](https://pkg.go.dev/text/template).                    | `"{{ .Pod.Name }}"`      |
| template.source.endpoint        | Endpoint, can be a [template string](https://pkg.go.dev/text/template).                                 | `""`                     |
| template.message                | Message format, can be a [template string](https://pkg.go.dev/text/template).                           | `"{{ .Event.Message }}"` |