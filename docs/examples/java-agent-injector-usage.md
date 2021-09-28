# Java agent injector Usage

In this example , you will learn how to use the java agent injector in three ways.

## Install injector

The java agent injector is builded in the operator , so you need to follow [Operator installation instrument](../../README.md#operator) to install the operator firstly.

## Use default configuration

At first , set the injection label in your namespace as below.

```shell
kubectl label namespace default(your namespace) swck-injection=enabled
```

Then add `swck-java-agent-injected: "true"` in the labels of yaml file as below.

```
apiVersion: v1
kind: Pod
metadata:
  name: inject-demo
  labels:
    swck-java-agent-injected: "true"
    app: demo
  namespace: default
spec:
  containers:
  - name: demo
    image: dashanji/swck-spring-demo:v0.0.3
    command: ["java"]
    args: ["-jar","$(AGENT_OPTS)","-jar","/app.jar"]
   
```

Get injected resources as below:

```
spec:
  containers:
  - args:
    - -jar
    - $(AGENT_OPTS)
    - -jar
    - /app.jar
    command:
    - java
    env:
    - name: AGENT_OPTS
      value: -javaagent:/sky/agent/skywalking-agent.jar
    image: dashanji/swck-spring-demo:v0.0.3
    name: demo
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
    image: apache/skywalking-java-agent:8.7.0-jdk8
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

## Use annotation to override sidecar configuration

At first , set the injection label in your namespace as below.

```shell
kubectl label namespace default(your namespace) swck-injection=enabled
```

Then add `swck-java-agent-injected: "true"` in the labels of yaml file and add the [sidecar configuration](../java-agent-injector.md#configure-sidecar) to the annotations as below.

```
apiVersion: v1
kind: Pod
metadata:
  labels:
    swck-java-agent-injected: "true"
  annotations:
    sidecar.skywalking.apache.org/initcontainer.Name: "test-inject-agent"
    sidecar.skywalking.apache.org/initcontainer.Image: "apache/skywalking-java-agent:8.5.0-jdk8"
    sidecar.skywalking.apache.org/initcontainer.Command: "sh"
    sidecar.skywalking.apache.org/initcontainer.args.Option: "-c"
    sidecar.skywalking.apache.org/initcontainer.args.Command: "mkdir -p /skytest/agent && cp -r /skywalking/agent/* /skytest/agent"
    sidecar.skywalking.apache.org/sidecarVolumeMount.MountPath: "/skytest/agent"
    sidecar.skywalking.apache.org/configmapVolumeMount.MountPath: "/skytest/agent/config"
    sidecar.skywalking.apache.org/configmapVolume.ConfigMap.Name: "haha"
    sidecar.skywalking.apache.org/env.Value: "-javaagent:/skytest/agent/skywalking-agent.jar"
  name: inject2
  namespace: default
spec:
  containers:
  - name: demo
    image: dashanji/swck-spring-demo:v0.0.3
    command: ["java"]
    args: ["-jar","$(AGENT_OPTS)","-jar","/app.jar"]
```

Get injected resources as below:

```
spec:
  containers:
  - args:
    - -jar
    - $(AGENT_OPTS)
    - -jar
    - /app.jar
    command:
    - java
    env:
    - name: AGENT_OPTS
      value: -javaagent:/skytest/agent/skywalking-agent.jar
    image: dashanji/swck-spring-demo:v0.0.3
    name: demo
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
      name: haha
    name: java-agent-configmap-volume
```

```shell
$ kubectl get configmap skywalking-swck-java-agent-configmap -n default(your namespace)
NAME                                   DATA   AGE
skywalking-swck-java-agent-configmap   1      61s
```

#### Use annotation to set the coverage strategy and override the agent configuration

At first , set the injection label in your namespace as below.

```shell
kubectl label namespace default(your namespace) swck-injection=enabled
```

Then add `swck-java-agent-injected: "true"` in the labels of yaml file and [agent configuration](../java-agent-injector.md#use-annotations-to-overlay-default-agent-configuration) and [sidecar configuration](../java-agent-injector.md#configure-sidecar) to annotations as below.


```
apiVersion: v1
kind: Pod
metadata:
  labels:
    swck-java-agent-injected: "true"
  annotations:
    strategy.skywalking.apache.org/inject.Container: "demo"
    strategy.skywalking.apache.org/agent.Overlay: "true"
    agent.skywalking.apache.org/agent.sample_n_per_3_secs: "6"
    agent.skywalking.apache.org/agent.class_cache_mode: "MEMORY"
    agent.skywalking.apache.org/agent.ignore_suffix: "'jpg,.jpeg'"
    plugins.skywalking.apache.org/plugin.mount: "'plugins,activations'"
    plugins.skywalking.apache.org/plugin.mongodb.trace_param: "true"
    plugins.skywalking.apache.org/plugin.influxdb.trace_influxql: "false"
    optional.skywalking.apache.org: "trace|webflux|cloud-gateway-2.1.x"
    optional-reporter.skywalking.apache.org: "kafka"
  name: inject3
  namespace: default
spec:
  containers:
  - name: nginx
    image: nginx:1.16.1
  - name: demo
    image: dashanji/swck-spring-demo:v0.0.3
    command: ["java"]
    args: ["-jar","$(AGENT_OPTS)","-jar","/app.jar"]
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
    - $(AGENT_OPTS)
    - -jar
    - /app.jar
    command:
    - java
    env:
    - name: AGENT_OPTS
      value: -javaagent:/sky/agent/skywalking-agent.jar=agent.ignore_suffix='jpg,.jpeg',agent.sample_n_per_3_secs=6,agent.class_cache_mode=MEMORY,plugin.mount='plugins,activations',plugin.influxdb.trace_influxql=false,plugin.mongodb.trace_param=true
    image: dashanji/swck-spring-demo:v0.0.3
    name: demo
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
    image: apache/skywalking-java-agent:8.7.0-jdk8
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
