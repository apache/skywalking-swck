## Java agent injector manual

The java agent injector consists of three parts:

* Determine whether a pod needs to be injected by labels.
* Only use the default configuration.
* Based on the default configuration , the configuration of the injection container and java agent is overwritten by annotations.  

### 一、Enable injector—label

The injector implements two levels of label judgment, namely the namespace level and the pod level.

1. **namespace level**. First, you need to label the injected resources under the namespace, as shown below, only the resources under the namespace can use the injector.

```
kubectl label namespace default(your namespace) swck-injection=enabled
```

2. **pod level**，Add a label to the pod that needs to be injected, as shown below. 

```
swck-java-agent-injected: "true" 
```

### 二、Use default configuration

After the injector is enabled, if you don't add other annotation to override, the default configuration is used for injection, as shown below.

```
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

The default configuration after injection is as shown above. In addition, the injector will additionally create a configmap to overwrite the `agent.config` in the image, so as to realize the configuration outside the image without entering the image to modify the agent configuration. When the user submits an injection request , There may be two situations:

- After the injector intercepts the user‘s request, if there is no specified configmap in the namespace where the user's request is located, the injector will first look for the configmap (skywalking-swck-java-agent-configmap) under the system namespace (skywalking-swck-system), the configuration is the default global configuration, which is also the `agent.config` in the original image, then extract the configmap's data and create a configmap containing the same data under the user namespace.
- After the injector intercepts the user's request, if a default configmap (skywalking-swck-java-agent-configmap) has been created under the namespace where the user's request is located, the configmap will be used to overwrite the agent configuration.

The default value of configmap is shown in [java agent configuration](https://skywalking.apache.org/docs/main/v8.7.0/en/setup/service-agent/java-agent/readme/#table-of-agent-configuration-properties). It should be noted that the configmap created for the first time contains all the configuration information of `agent.config`, including the configuration information of all plugins such as agent, collector, logging, plugin and so on. There are two main situations for using configmap:

- Since the configmap may be deleted by mistake, the injector contains a configmap controller, which will continue to watch the configmap under the system namespace. If the user deletes it by mistake, a new configmap will be created. The new configmap will not contain plugins, and the configuration information of the plugins needs to be manually added by the user.
- The user may wish to modify the global configuration, directly modify the configmap (skywalking-swck-java-agent-configmap) under the default namespace (skywalking-swck-system). Two of the fields need to be verified, one is `agent.service_name` and the string cann't be empty; the other is `collector.backend_service` and it needs to be a legal IP address and port, the other fields need to be guaranteed by users themselves.

### 三、Custom configuration—annotation

The annotations that the injector can recognize are divided into three categories:

#### 1. Coverage strategy

The coverage strategy is a field starting with the prefix `strategy.skywalking.apache.org/`, specifically the following two fields:

| Annotation key                                    | Description                                                  | Annotation Default value |
| ------------------------------------------------- | ------------------------------------------------------------ | ------------------------ |
| `strategy.skywalking.apache.org/inject.Container` | Select the container where the java agent needs to be injected, if not set, inject all containers. | not set                  |
| `strategy.skywalking.apache.org/agent.Overlay`    | This field means whether it is necessary to override the agent configuration through annotations. If set true , then you can follow [Java agent configuration](#3-java-agent-configuration)，if set false, then you can skip [Java agent configuration](#3-java-agent-configuration). | `false`                  |

#### 2. Sidecar configuration

The sidecar configuration is a field starting with the prefix `sidecar.skywalking.apache.org/`, specifically including the following fields:

| Annotation key                                               | Description                                                  | Annotation Default value                                     |
| ------------------------------------------------------------ | ------------------------------------------------------------ | ------------------------------------------------------------ |
| `sidecar.skywalking.apache.org/initcontainer.Name`           | The name of the injected java agent container.               | `inject-skywalking-agent`                                    |
| `sidecar.skywalking.apache.org/initcontainer.Image`          | The container image of the injected java agent container.    | `apache/skywalking-java-agent:8.7.0-jdk8`                    |
| `sidecar.skywalking.apache.org/initcontainer.Command`        | The command of the injected java agent container.            | `sh`                                                         |
| `sidecar.skywalking.apache.org/initcontainer.args.Option`    | The args option of  the injected java agent container.       | `-c`                                                         |
| `sidecar.skywalking.apache.org/initcontainer.args.Command`   | The args command of  the injected java agent container.      | `mkdir -p /sky/agent && cp -r /skywalking/agent/* /sky/agent` |
| `sidecar.skywalking.apache.org/sidecarVolume.Name`           | The name of sidecar Volume.                                  | `sky-agent`                                                  |
| `sidecar.skywalking.apache.org/sidecarVolumeMount.MountPath` | Mount path of the agent directory in the injected container. | `/sky/agent`                                                 |
| `sidecar.skywalking.apache.org/configmapVolume.Name`         | The name of configmap volume.                                | `java-agent-configmap-volume`                                |
| `sidecar.skywalking.apache.org/configmapVolumeMount.MountPath` | Mount path of the configmap in the injected container        | `/sky/agent/config`                                          |
| `sidecar.skywalking.apache.org/configmapVolume.ConfigMap.Name` | The name pf configmap used in the injected container as `agent.config ` | `skywalking-swck-java-agent-configmap`                       |
| `sidecar.skywalking.apache.org/env.Name`                     | Environment Name used by the injected container (application container). | `AGENT_OPTS`                                                 |
| `sidecar.skywalking.apache.org/env.Value`                    | Environment variables used by the injected container (application container). | `-javaagent:/sky/agent/skywalking-agent.jar`                 |

#### 3. Java agent configuration

Java agent configuration includes four parts: agent configuration, plugin configuration, optional plugin configuration and optional reporter plugin configuration.

##### 3.1 agent configuration

The key is a string containing the prefix `agent.skywalking.apache.org/`, and the method used in the annotation is `agent.skywalking.apache.org/{option}: {value}`, where option contains all the configurations in the agent.config file that are not prefixed with `plugins`, such as `agent.skywalking.apache.org/agent.namespace` , `agent.skywalking.apache.org/meter.max_meter_size` , etc. see details in

[agent configuration](https://skywalking.apache.org/docs/main/v8.7.0/en/setup/service-agent/java-agent/readme/#table-of-agent-configuration-properties).

##### 3.2 plugins configuration

The key is a string containing the prefix `plugins.skywalking.apache.org/`, and it is used in the annotation as `plugins.skywalking.apache.org/{option}: {value}`, where option contains all the configurations prefixed by `plugins` in the `agent.config`, such as `plugins.skywalking.apache.org/plugin.mount`, ``plugins.skywalking.apache.org/plugin.mongodb.trace_param``, etc. see details in [plugins configuration](https://skywalking.apache.org/docs/main/v8.7.0/en/setup/service-agent/java-agent/readme/#table-of-agent-configuration-properties)。

##### 3.3 optional plugin configuration

The key is `optional.skywalking.apache.org`, as follows.

| Annotation key                   | Description                                                  | Annotation value |
| -------------------------------- | ------------------------------------------------------------ | ---------------- |
| `optional.skywalking.apache.org` | The Annotation is used to select the plugins which need to be moved to to the directory(/plugins). The optional plugins are all in the directory(/optional-plugins), and the user can select multiple optional plugins via `|` , such as `trace|webflux|cloud-gateway-2.1.x` | not set          |

##### 3.4 optional reporter plugin configuration

The key is `optional-reporter.skywalking.apache.org`, as follows。

| Annotation key                            | Description                                                  | Annotation value |
| ----------------------------------------- | ------------------------------------------------------------ | ---------------- |
| `optional-reporter.skywalking.apache.org` | The Annotation is like`optional.skywalking.apache.org` . The optional-reporter plugins are all in the directory(/optional-reporter-plugins), and the user can select multiple optional plugins via `|` ，such as `kafka` | not set          |

### 四、Usage example

Next, several demos will be used to explain how to use the java agent injector.

##### 4.1 Use default configuration

First set the injection label in the namespace, then enable the injector in the labels of the resource, and use the default configuration, as shown below.

```
apiVersion: v1
kind: Pod
metadata:
  name: inject-demo
  labels:
    swck-java-agent-injected: "true"
    app: demo
  namespace: caoye
spec:
  containers:
  - name: demo
    image: dashanji/swck-spring-demo:v0.0.3
    command: ["java"]
    args: ["-jar","$(AGENT_OPTS)","-jar","/app.jar"]
   
```

The injected configuration is shown below. In addition, a  configmap named `skywalking-swck-java-agent-configmap` will be created under the same namespace to overwrite the `agent.config`.

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

##### 4.2 Use annotation to override sidecar configuration

Add the sidecar configuration to the annotations, as shown below.

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
  namespace: caoye
spec:
  containers:
  - name: demo
    image: dashanji/swck-spring-demo:v0.0.3
    command: ["java"]
    args: ["-jar","$(AGENT_OPTS)","-jar","/app.jar"]
```

The injection results are shown below.

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

##### 4.3 Use annotation to set the coverage strategy and override the agent configuration

Add coverage strategy and agent configuration to annotations, as shown below.

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
    plugins.skywalking.apache.org/plugin.mongodb.trace_param: "true"
    plugins.skywalking.apache.org/plugin.influxdb.trace_influxql: "false"
    optional.skywalking.apache.org: "trace|webflux|cloud-gateway-2.1.x"
    optional-reporter.skywalking.apache.org: "kafka"
  name: inject3
  namespace: caoye
spec:
  containers:
  - name: nginx
    image: nginx:1.16.1
  - name: demo
    image: dashanji/swck-spring-demo:v0.0.3
    command: ["java"]
    args: ["-jar","$(AGENT_OPTS)","-jar","/app.jar"]
```

The injection results are shown below.

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
      value: -javaagent:/sky/agent/skywalking-agent.jar=agent.sample_n_per_3_secs=6,agent.class_cache_mode=MEMORY,plugin.influxdb.trace_influxql=false,plugin.mongodb.trace_param=true
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

