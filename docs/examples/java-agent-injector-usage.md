# Java agent injector Usage

In this example, you will learn how to use the java agent injector.

## Install injector

The java agent injector is a component of the operator, so you need to
follow [Operator installation instrument](../../README.md#operator) to install the operator firstly.

## Deployment Example

Let's take a demo deployment for example.

```yaml
# demo1.yaml
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

## Enable Injection for Namespace and Deployments/StatefulSets.

At first, set the injection label in your namespace as below.

```shell
kubectl label namespace default(your namespace) swck-injection=enabled
```

Secondly, set the injection label for your target Deployment/StatefulSet.

```shell
kubectl -n default(your namespace) patch deployment demo1 --patch '{"spec": {"template": {"metadata": {"labels": {"swck-java-agent-injected": "true"}}}}}'
```

Then the pods create by the Deployments/StatefulSets would be recreated with agent injected.

The injected pods would be like this:

```yaml
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
$ kubectl get javaagent app-demo1-javaagent -o yaml
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

## Use SwAgent CR to setup override default configuration

Suppose that injection label had been set for Namespace and Deployments/StatefulSets as [previous said](java-agent-injector-usage.md#enable-injection-for-namespace-and-deploymentsstatefulsets).

Apply SwAgent CR with correct label selector and container matcher:

```yaml
# SwAgent.yaml
apiVersion: operator.skywalking.apache.org/v1alpha1
kind: SwAgent
metadata:
  name: swagent-001
  namespace: skywalking-system
spec:
  # container matcher, regular expression is supported
  containerMatcher: '.*'
  # label selector
  selector:
    app: demo1
  javaSidecar:
    # init container name
    name: swagent-001
    # init container image
    image: apache/skywalking-java-agent:8.9.0-java8
    # envs to be appended to target containers
    env:
      - name: SW_LOGGING_LEVEL
        value: "DEBUG"
  # share volume
  sharedVolume:
    name: "sky-agent-test-001"
    mountPath: "/sky/agent"
  # default configmap
  swConfigMapVolume:
    name: "java-agent-configmap-test-001-volume"
    configMapName: "skywalking-swck-java-agent-configmap"
    configMapMountPath: "/sky/agent/config"

```

```shell
kubectl -n skywalking-system apply swagent.yaml
```

You can also get SwAgent CR by:

```shell
kubectl -n skywalking-system get SwAgent
```

Now the pod is still the old one, because pod could not load the SwAgent config automatically.

So you need to recreate pod to load swagent config. For the pods created by Deployment/StatefulSet, you can just simply delete the old pod.

```shell
# verify pods to be delete 
kubectl -n skywalking-system get pods -l app=demo1
# delete pods
kubectl -n skywalking-system delete pods -l app=demo1
```

After the pods recreated, we can get injected pod as below.

```shell
kubectl -n skywalking-system get pods -l app=demo1
```

```yaml
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
    - name: SW_LOGGING_LEVEL
      value: "DEBUG"
    image: ghcr.io/apache/skywalking-swck-spring-demo:v0.0.1
    name: demo1
    - mountPath: /sky/agent
      name: sky-agent-test-001
    - mountPath: /sky/agent/config
      name: java-agent-configmap-test-001-volume
  initContainers:
  - args:
    - -c
    - mkdir -p /sky/agent && cp -r /skywalking/agent/* /sky/agent
    command:
    - sh
    image: apache/skywalking-java-agent:8.9.0-java8
    name: swagent-001
    volumeMounts:
    - mountPath: /sky/agent
      name: sky-agent-test-001
  volumes:
  - emptyDir: {}
    name: sky-agent-test-001
  - configMap:
      name: skywalking-swck-java-agent-configmap
    name: java-agent-configmap-test-001-volume
```

## Use annotation to override sidecar configuration

Suppose that injection label had been set for Namespace and Deployments/StatefulSets as [previous said](java-agent-injector-usage-draft.md#enable-injection-for-namespace-and-deploymentsstatefulsets).

Then add [agent configuration](../java-agent-injector.md#use-annotations-to-overlay-default-agent-configuration)
and [sidecar configuration](../java-agent-injector.md#configure-sidecar) to annotations as below.

```yaml
# demo1.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: demo1
  namespace: default
spec:
  replicas: 3
  selector:
    matchLabels:
      app: demo1
  template:
    metadata:
      name: inject-demo3
      labels:
        swck-java-agent-injected: "true"
        app: demo1
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
      - name: demo1
        image: ghcr.io/apache/skywalking-swck-spring-demo:v0.0.1
        command: ["java"]
        args: ["-jar","/app.jar"]
```

Then we can get injected pod as below:

```shell
kubectl -n skywalking-system get pods -l app=demo1
```

```yaml
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
    name: demo1
    - mountPath: /sky/agent
      name: sky-agent-test-001
    - mountPath: /sky/agent/config
      name: java-agent-configmap-test-001-volume
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
    name: swagent-001
    volumeMounts:
    - mountPath: /sky/agent
      name: sky-agent-test-001
  volumes:
  - emptyDir: {}
    name: sky-agent-test-001
  - configMap:
      name: skywalking-swck-java-agent-configmap
    name: java-agent-configmap-test-001-volume

```

Then you can get the final agent configuration and the pod as below.

```shell
$ kubectl get javaagent
NAME                  PODSELECTOR   SERVICENAME   BACKENDSERVICE
app-demo1-javaagent   app=demo1     app           127.0.0.1:11800
$ kubectl get pod -l app=demo1(the podSelector)
NAME                     READY   STATUS    RESTARTS   AGE
demo1-69bff546df-55w8c   1/1     Running   0          57s
demo1-69bff546df-mn5fq   1/1     Running   0          57s
demo1-69bff546df-skklk   1/1     Running   0          57s
```

Get the javaagent's yaml for more datails.

```shell
$ kubectl get javaagent app-demo3-javaagent -oyaml
apiVersion: operator.skywalking.apache.org/v1alpha1
kind: JavaAgent
metadata:
  creationTimestamp: "2021-10-15T05:21:06Z"
  generation: 1
  name: app-demo1-javaagent
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