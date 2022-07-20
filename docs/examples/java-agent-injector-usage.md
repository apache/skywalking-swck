# Java agent injector Usage

In this example, you will learn how to use the java agent injector in three ways.

## Install injector

The java agent injector is a component of the operator, so you need to follow [Operator installation instrument](../../README.md#operator) to install the operator firstly.

## Use default configuration

At first, set the injection label in your namespace as below.

```shell
kubectl label namespace default(your namespace) swck-injection=enabled
```

Then add `swck-java-agent-injected: "true"` in the labels of the yaml file as below.

```
apiVersion: apps/v1
kind: Deployment
metadata:
  name: demo1
  namespace: default
spec:
  selector:
    matchLabels:
      app: demo1
  template:
    metadata:
      labels:
        swck-java-agent-injected: "true"
        app: demo1
    spec:
      containers:
      - name: demo1
        image: ghcr.io/apache/skywalking-swck-spring-demo:v0.0.1
        command: ["java"]
        args: ["-jar","/app.jar"]
   
```

Get injected resources as below:

```
spec:
  containers:
  - args:
    - -jar
    - /app.jar
    command:
    - java
    env:
    - name: JAVA_TOOL_OPTIONS
      value: -javaagent:/sky/agent/skywalking-agent.jar
    image: ghcr.io/apache/skywalking-swck-spring-demo:v0.0.1
    name: demo1
    - mountPath: /sky/agent
      name: sky-agent
    - mountPath: /sky/agent/config
      name: java-agent-configmap-volume
  initContainers:
  - args:
    - -c
    - mkdir -p /sky/agent && cp -r /skywalking/agent/* /sky/agent
    command:
    - sh
    image: apache/skywalking-java-agent:8.10.0-java8
    name: inject-skywalking-agent
    volumeMounts:
    - mountPath: /sky/agent
      name: sky-agent
  volumes:
  - emptyDir: {}
    name: sky-agent
  - configMap:
      name: skywalking-swck-java-agent-configmap
    name: java-agent-configmap-volume
```

```shell
$ kubectl get configmap skywalking-swck-java-agent-configmap -n default(your namespace)
NAME                                   DATA   AGE
skywalking-swck-java-agent-configmap   1      61s
```

Then you can get the final agent configuration and the pod as below.

```shell
$ kubectl get javaagent
NAME                  PODSELECTOR   SERVICENAME            BACKENDSERVICE
app-demo1-javaagent   app=demo1     Your_ApplicationName   127.0.0.1:11800
$ kubectl get pod -l app=demo1(the podSelector)
NAME                     READY   STATUS    RESTARTS   AGE
demo1-8554b96b4c-6czv7   1/1     Running   0          85s
```

Get the javaagent's yaml for more datails.

```shell
$ kubectl get javaagent app-demo1-javaagent -oyaml
apiVersion: operator.skywalking.apache.org/v1alpha1
kind: JavaAgent
metadata:
  creationTimestamp: "2021-10-15T04:52:46Z"
  generation: 1
  name: app-demo1-javaagent
  namespace: default
  ownerReferences:
  - apiVersion: apps/v1
    blockOwnerDeletion: true
    controller: true
    kind: ReplicaSet
    name: demo1-8554b96b4c
    uid: f3d71b5c-1e26-401a-8d0d-055d1e89bc64
  resourceVersion: "455104"
  selfLink: /apis/operator.skywalking.apache.org/v1alpha1/namespaces/default/javaagents/app-demo1-javaagent
  uid: 2bf828d6-4f83-4c7d-9356-83066ae334d3
spec:
  agentConfiguration:
    agent.service_name: Your_ApplicationName
    collector.backend_service: 127.0.0.1:11800
  backendService: 127.0.0.1:11800
  podSelector: app=demo1
  serviceName: Your_ApplicationName
status:
  creationTime: "2021-10-15T04:52:46Z"
  expectedInjectiedNum: 1
  lastUpdateTime: "2021-10-15T04:52:49Z"
  realInjectedNum: 1
```

## Use annotation to override sidecar configuration

At first, set the injection label in your namespace as below.

```shell
kubectl label namespace default(your namespace) swck-injection=enabled
```

Then add `swck-java-agent-injected: "true"` in the labels of yaml file and add the [sidecar configuration](../java-agent-injector.md#configure-sidecar) to the annotations as below.

```
apiVersion: apps/v1
kind: Deployment
metadata:
  name: demo2
  namespace: default
spec:
  selector:
    matchLabels:
      app: demo2
  template:
    metadata:
      labels:
        swck-java-agent-injected: "true"
        app: demo2
      annotations:
        sidecar.skywalking.apache.org/initcontainer.Name: "test-inject-agent"
        sidecar.skywalking.apache.org/initcontainer.Image: "apache/skywalking-java-agent:8.5.0-jdk8"
        sidecar.skywalking.apache.org/initcontainer.Command: "sh"
        sidecar.skywalking.apache.org/initcontainer.args.Option: "-c"
        sidecar.skywalking.apache.org/initcontainer.args.Command: "mkdir -p /skytest/agent && cp -r /skywalking/agent/* /skytest/agent"
        sidecar.skywalking.apache.org/sidecarVolumeMount.MountPath: "/skytest/agent"
        sidecar.skywalking.apache.org/configmapVolumeMount.MountPath: "/skytest/agent/config"
        sidecar.skywalking.apache.org/configmapVolume.ConfigMap.Name: "newconfigmap"
        sidecar.skywalking.apache.org/env.Value: "-javaagent:/skytest/agent/skywalking-agent.jar"
    spec:
      containers:
      - name: demo2
        image: ghcr.io/apache/skywalking-swck-spring-demo:v0.0.1
        command: ["java"]
        args: ["-jar","/app.jar"]
```

Get injected resources as below:

```
spec:
  containers:
  - args:
    - -jar
    - /app.jar
    command:
    - java
    env:
    - name: JAVA_TOOL_OPTIONS
      value: -javaagent:/skytest/agent/skywalking-agent.jar
    image: ghcr.io/apache/skywalking-swck-spring-demo:v0.0.1
    name: demo2
    - mountPath: /skytest/agent
      name: sky-agent
    - mountPath: /skytest/agent/config
      name: java-agent-configmap-volume
  initContainers:
  - args:
    - -c
    - mkdir -p /skytest/agent && cp -r /skywalking/agent/* /skytest/agent
    command:
    - sh
    image: apache/skywalking-java-agent:8.5.0-jdk8
    name: test-inject-agent
    volumeMounts:
    - mountPath: /skytest/agent
      name: sky-agent
  volumes:
  - emptyDir: {}
    name: sky-agent
  - configMap:
      name: newconfigmap
    name: java-agent-configmap-volume
```

```shell
$ kubectl get configmap newconfigmap -n default
NAME           DATA   AGE
newconfigmap   1      2m29s
```

Then you can get the final agent configuration and the pod as below.

```shell
$ kubectl get javaagent
NAME                  PODSELECTOR   SERVICENAME            BACKENDSERVICE
app-demo2-javaagent   app=demo2     Your_ApplicationName   127.0.0.1:11800
$ kubectl get pod -l app=demo2(the podSelector)
NAME                     READY   STATUS    RESTARTS   AGE
demo2-74b65f98b9-k5wvd   1/1     Running   0          3m28s
```

Get the javaagent's yaml for more datails.

```shell
$ kubectl get javaagent app-demo2-javaagent -oyaml
apiVersion: operator.skywalking.apache.org/v1alpha1
kind: JavaAgent
metadata:
  creationTimestamp: "2021-10-15T05:10:16Z"
  generation: 1
  name: app-demo2-javaagent
  namespace: default
  ownerReferences:
  - apiVersion: apps/v1
    blockOwnerDeletion: true
    controller: true
    kind: ReplicaSet
    name: demo2-74b65f98b9
    uid: cbc2c680-4f84-469a-bb43-fc48161d6958
  resourceVersion: "458626"
  selfLink: /apis/operator.skywalking.apache.org/v1alpha1/namespaces/default/javaagents/app-demo2-javaagent
  uid: 7c59aab7-30fc-4122-8b39-4cba2d1711b5
spec:
  agentConfiguration:
    agent.service_name: Your_ApplicationName
    collector.backend_service: 127.0.0.1:11800
  backendService: 127.0.0.1:11800
  podSelector: app=demo2
  serviceName: Your_ApplicationName
status:
  creationTime: "2021-10-15T05:10:16Z"
  expectedInjectiedNum: 1
  lastUpdateTime: "2021-10-15T05:10:18Z"
  realInjectedNum: 1
```

#### Use annotation to set the coverage strategy and override the agent configuration

At first, set the injection label in your namespace as below.

```shell
kubectl label namespace default(your namespace) swck-injection=enabled
```

Then add `swck-java-agent-injected: "true"` in the labels of yaml file and [agent configuration](../java-agent-injector.md#use-annotations-to-overlay-default-agent-configuration) and [sidecar configuration](../java-agent-injector.md#configure-sidecar) to annotations as below.


```
apiVersion: apps/v1
kind: Deployment
metadata:
  name: demo3
  namespace: default
spec:
  replicas: 3
  selector:
    matchLabels:
      app: demo3
  template:
    metadata:
      name: inject-demo3
      labels:
        swck-java-agent-injected: "true"
        app: demo3
      annotations:
        strategy.skywalking.apache.org/inject.Container: "demo"
        strategy.skywalking.apache.org/agent.Overlay: "true"
        agent.skywalking.apache.org/agent.service_name: "app"
        agent.skywalking.apache.org/agent.sample_n_per_3_secs: "6"
        agent.skywalking.apache.org/agent.class_cache_mode: "MEMORY"
        agent.skywalking.apache.org/agent.ignore_suffix: "'jpg,.jpeg'"
        plugins.skywalking.apache.org/plugin.mount: "'plugins,activations'"
        plugins.skywalking.apache.org/plugin.mongodb.trace_param: "true"
        plugins.skywalking.apache.org/plugin.influxdb.trace_influxql: "false"
        optional.skywalking.apache.org: "trace|webflux|cloud-gateway-2.1.x"
        optional-reporter.skywalking.apache.org: "kafka"
      namespace: default
    spec:
      containers:
      - name: demo3
        image: ghcr.io/apache/skywalking-swck-spring-demo:v0.0.1
        command: ["java"]
        args: ["-jar","/app.jar"]
```

Get injected resources as below:

```
spec:
  containers:
  - image: nginx:1.16.1
    imagePullPolicy: IfNotPresent
    name: nginx
  - args:
    - -jar
    - /app.jar
    command:
    - java
    env:
    - name: JAVA_TOOL_OPTIONS
      value: -javaagent:/sky/agent/skywalking-agent.jar=agent.ignore_suffix='jpg,.jpeg',agent.class_cache_mode=MEMORY,agent.sample_n_per_3_secs=6,agent.service_name=app,plugin.mount='plugins,activations',plugin.influxdb.trace_influxql=false,plugin.mongodb.trace_param=true
    image: ghcr.io/apache/skywalking-swck-spring-demo:v0.0.1
    name: demo3
    - mountPath: /sky/agent
      name: sky-agent
    - mountPath: /sky/agent/config
      name: java-agent-configmap-volume
  initContainers:
  - args:
    - -c
    - mkdir -p /sky/agent && cp -r /skywalking/agent/* /sky/agent && cd /sky/agent/optional-plugins/&&
      ls | grep -E "trace|webflux|cloud-gateway-2.1.x"  | xargs -i cp {} /sky/agent/plugins/
      && cd /sky/agent/optional-reporter-plugins/&& ls | grep -E "kafka"  | xargs
      -i cp {} /sky/agent/plugins/
    command:
    - sh
    image: apache/skywalking-java-agent:8.10.0-java8
    name: inject-skywalking-agent
    volumeMounts:
    - mountPath: /sky/agent
      name: sky-agent
  volumes:
  - emptyDir: {}
    name: sky-agent
  - configMap:
      name: skywalking-swck-java-agent-configmap
    name: java-agent-configmap-volume
```

```shell
$ kubectl get configmap skywalking-swck-java-agent-configmap -n default
NAME                                   DATA   AGE
skywalking-swck-java-agent-configmap   1      17s
```

Then you can get the final agent configuration and the pod as below.

```shell
$ kubectl get javaagent
NAME                  PODSELECTOR   SERVICENAME   BACKENDSERVICE
app-demo3-javaagent   app=demo3     app           127.0.0.1:11800
$ kubectl get pod -l app=demo3(the podSelector)
NAME                     READY   STATUS    RESTARTS   AGE
demo3-69bff546df-55w8c   1/1     Running   0          57s
demo3-69bff546df-mn5fq   1/1     Running   0          57s
demo3-69bff546df-skklk   1/1     Running   0          57s
```

Get the javaagent's yaml for more datails.

```shell
$ kubectl get javaagent app-demo3-javaagent -oyaml
apiVersion: operator.skywalking.apache.org/v1alpha1
kind: JavaAgent
metadata:
  creationTimestamp: "2021-10-15T05:21:06Z"
  generation: 1
  name: app-demo3-javaagent
  namespace: default
  ownerReferences:
  - apiVersion: apps/v1
    blockOwnerDeletion: true
    controller: true
    kind: ReplicaSet
    name: demo3-69bff546df
    uid: 1f04a239-0247-4f5c-967a-64009adae42c
  resourceVersion: "461000"
  selfLink: /apis/operator.skywalking.apache.org/v1alpha1/namespaces/default/javaagents/app-demo3-javaagent
  uid: be35588e-9b2d-4528-90df-c1a630616632
spec:
  agentConfiguration:
    agent.class_cache_mode: MEMORY
    agent.ignore_suffix: '''jpg,.jpeg'''
    agent.sample_n_per_3_secs: "6"
    agent.service_name: app
    collector.backend_service: 127.0.0.1:11800
    optional-plugin: trace|webflux|cloud-gateway-2.1.x
    optional-reporter-plugin: kafka
    plugin.influxdb.trace_influxql: "false"
    plugin.mongodb.trace_param: "true"
    plugin.mount: '''plugins,activations'''
  backendService: 127.0.0.1:11800
  podSelector: app=demo3
  serviceName: app
status:
  creationTime: "2021-10-15T05:21:06Z"
  expectedInjectiedNum: 3
  lastUpdateTime: "2021-10-15T05:21:10Z"
  realInjectedNum: 3
```