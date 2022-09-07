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
        app: demo1
    spec:
      containers:
        - name: demo1
          image: ghcr.io/apache/skywalking-swck-spring-demo:v0.0.1
          command: ["java"]
          args: ["-jar","/app.jar"]
          ports:
            - containerPort: 8085
          readinessProbe:
            httpGet:
              path: /hello
              port: 8085
            initialDelaySeconds: 3
            periodSeconds: 3
            failureThreshold: 10

```

## Enable Injection for Namespace and Deployments/StatefulSets.

Firstly, set the injection label in your namespace as below.

```shell
kubectl label namespace default(your namespace) swck-injection=enabled
```

Secondly, set the injection label for your target Deployment/StatefulSet.

```shell
kubectl -n default patch deployment demo1 --patch '{
    "spec": {
        "template": {
            "metadata": {
                "labels": {
                    "swck-java-agent-injected": "true"
                }
            }
        }
    }
}'
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
```

Then you can get the final agent configuration and the pod as below.

```shell
$ kubectl get javaagent
NAME                  PODSELECTOR   SERVICENAME   BACKENDSERVICE
app-demo1-javaagent   app=demo1     demo1         127.0.0.1:11800
$ kubectl get pod -l app=demo1(the podSelector)
NAME                     READY   STATUS    RESTARTS   AGE
demo1-5fbb6fcd98-cq5ws   1/1     Running   0          54s
```

Get the javaagent's yaml for more datails.

```shell
$ kubectl get javaagent app-demo1-javaagent -o yaml
apiVersion: operator.skywalking.apache.org/v1alpha1
kind: JavaAgent
metadata:
  creationTimestamp: "2022-08-16T12:09:34Z"
  generation: 1
  name: app-demo1-javaagent
  namespace: default
  ownerReferences:
  - apiVersion: apps/v1
    blockOwnerDeletion: true
    controller: true
    kind: ReplicaSet
    name: demo1-7fdffc7b95
    uid: 417c413f-0cc0-41f9-b6eb-0192eb8c8622
  resourceVersion: "25067"
  uid: 1cdab012-784c-4efb-b5d2-c032eb2fb22a
spec:
  backendService: 127.0.0.1:11800
  podSelector: app=demo1
  serviceName: Your_ApplicationName
status:
  creationTime: "2022-08-16T12:09:34Z"
  expectedInjectiedNum: 1
  lastUpdateTime: "2022-08-16T12:10:04Z"
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
  name: swagent-demo
  namespace: default
spec:
  containerMatcher: ''
  selector:
  javaSidecar:
    name: swagent-demo
    image: apache/skywalking-java-agent:8.10.0-java8
    env:
      - name: "SW_LOGGING_LEVEL"
        value: "DEBUG"
      - name: "SW_AGENT_COLLECTOR_BACKEND_SERVICES"
        value: "skywalking-system-oap:11800"
  sharedVolumeName: "sky-agent-demo"
  optionalPlugins:
    - "webflux"
    - "cloud-gateway-2.1.x"
```

```shell
kubectl -n default apply swagent.yaml
```

You can also get SwAgent CR by:

```shell
kubectl -n default get SwAgent
NAME           AGE
swagent-demo   38s
```

Now the pod is still the old one, because pod could not load the SwAgent config automatically.

So you need to recreate pod to load SwAgent config. For the pods created by Deployment/StatefulSet, you can just simply delete the old pod.

```shell
# verify pods to be delete 
kubectl -n default get pods -l app=demo1
# delete pods
kubectl -n default delete pods -l app=demo1
```

After the pods recreated, we can get injected pod as below.

```shell
kubectl -n default get pods -l app=demo1
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
        value: -javaagent:/sky/agent/skywalking-agent.jar=agent.service_name=demo1,collector.backend_service=skywalking-system-oap.skywalking-system:11800
      - name: SW_LOGGING_LEVEL
        value: DEBUG
      - name: SW_AGENT_COLLECTOR_BACKEND_SERVICES
        value: skywalking-system-oap.default.svc:11800
    image: ghcr.io/apache/skywalking-swck-spring-demo:v0.0.1
    name: demo1
    - mountPath: /sky/agent
      name: sky-agent-demo
  initContainers:
  - args:
    - -c
    - mkdir -p /sky/agent && cp -r /skywalking/agent/* /sky/agent && cd /sky/agent/optional-plugins/
      && ls | grep -E "webflux|cloud-gateway-2.1.x"  | xargs -i cp {} /sky/agent/plugins/
    command:
    - sh
    image: apache/skywalking-java-agent:8.10.0-java8
    name: swagent-demo
    volumeMounts:
    - mountPath: /sky/agent
      name: sky-agent-demo
  volumes:
  - emptyDir: {}
    name: sky-agent-demo
```

## Use annotation to override sidecar configuration

Suppose that injection label had been set for Namespace and Deployments/StatefulSets as [previous said](java-agent-injector-usage.md#enable-injection-for-namespace-and-deploymentsstatefulsets).

Then add [agent configuration](../java-agent-injector.md#use-annotations-to-overlay-default-agent-configuration)
and [sidecar configuration](../java-agent-injector.md#configure-sidecar) to annotations as below.

```yaml
# demo1_anno.yaml
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
      annotations:
        strategy.skywalking.apache.org/inject.Container: "demo1"
        agent.skywalking.apache.org/agent.service_name: "app"
        agent.skywalking.apache.org/agent.sample_n_per_3_secs: "6"
        agent.skywalking.apache.org/agent.class_cache_mode: "MEMORY"
        agent.skywalking.apache.org/agent.ignore_suffix: "'jpg,.jpeg'"
        plugins.skywalking.apache.org/plugin.mount: "'plugins,activations'"
        plugins.skywalking.apache.org/plugin.mongodb.trace_param: "true"
        plugins.skywalking.apache.org/plugin.influxdb.trace_influxql: "false"
        optional.skywalking.apache.org: "trace|webflux|cloud-gateway-2.1.x"
        optional-reporter.skywalking.apache.org: "kafka"
      labels:
        swck-java-agent-injected: "true"
        app: demo1
    spec:
      containers:
        - name: demo1
          image: ghcr.io/apache/skywalking-swck-spring-demo:v0.0.1
          command: ["java"]
          args: ["-jar","/app.jar"]
          ports:
            - containerPort: 8085
          readinessProbe:
            httpGet:
              path: /hello
              port: 8085
            initialDelaySeconds: 3
            periodSeconds: 3
            failureThreshold: 10
```

Then we can get injected pod as below:

```shell
kubectl -n default get pods -l app=demo1
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
      value: -javaagent:/sky/agent/skywalking-agent.jar=agent.ignore_suffix='jpg,.jpeg',agent.service_name=app,agent.class_cache_mode=MEMORY,agent.sample_n_per_3_secs=6,plugin.mongodb.trace_param=true,plugin.influxdb.trace_influxql=false,plugin.mount='plugins,activations'
    image: ghcr.io/apache/skywalking-swck-spring-demo:v0.0.1
    name: demo1
    ports:
    - containerPort: 8085
      protocol: TCP
    readinessProbe:
      failureThreshold: 10
      httpGet:
        path: /hello
        port: 8085
        scheme: HTTP
      initialDelaySeconds: 3
      periodSeconds: 3
      successThreshold: 1
      timeoutSeconds: 1
    volumeMounts: 
    - mountPath: /sky/agent
      name: sky-agent
  initContainers:
  - args:
    - -c
    - mkdir -p /sky/agent && cp -r /skywalking/agent/* /sky/agent && cd /sky/agent/optional-plugins/
      && ls | grep -E "trace|webflux|cloud-gateway-2.1.x"  | xargs -i cp {} /sky/agent/plugins/
      && cd /sky/agent/optional-reporter-plugins/ && ls | grep -E "kafka"  | xargs
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

```

Then you can get the final agent configuration and the pod as below.

```shell
$ kubectl get javaagent
NAME                  PODSELECTOR   SERVICENAME   BACKENDSERVICE
app-demo1-javaagent   app=demo1     app           127.0.0.1:11800

$ kubectl get pod -l app=demo1(the podSelector)
NAME                    READY   STATUS    RESTARTS   AGE
demo1-d48b96467-p7zrv   1/1     Running   0          5m25s
```

Get the javaagent's yaml for more datails.

```shell
$ kubectl get javaagent app-demo1-javaagent -o yaml
apiVersion: operator.skywalking.apache.org/v1alpha1
kind: JavaAgent
metadata:
  creationTimestamp: "2022-08-16T12:18:53Z"
  generation: 1
  name: app-demo1-javaagent
  namespace: default
  ownerReferences:
  - apiVersion: apps/v1
    blockOwnerDeletion: true
    controller: true
    kind: ReplicaSet
    name: demo1-d48b96467
    uid: 2b7f1ac4-b459-41cd-8568-ecd4578ca457
  resourceVersion: "26187"
  uid: c2b2f3e2-9442-4465-9423-d24249b2c53b
spec:
  agentConfiguration:
    agent.class_cache_mode: MEMORY
    agent.ignore_suffix: '''jpg,.jpeg'''
    agent.sample_n_per_3_secs: "6"
    agent.service_name: app
    optional-plugin: trace|webflux|cloud-gateway-2.1.x
    optional-reporter-plugin: kafka
    plugin.influxdb.trace_influxql: "false"
    plugin.mongodb.trace_param: "true"
    plugin.mount: '''plugins,activations'''
  backendService: 127.0.0.1:11800
  podSelector: app=demo1
  serviceName: app
status:
  creationTime: "2022-08-16T12:18:53Z"
  expectedInjectiedNum: 1
  lastUpdateTime: "2022-08-16T12:19:18Z"
  realInjectedNum: 1

```