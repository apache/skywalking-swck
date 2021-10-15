# JavaAgent Introduction

To see the final injected agent's configuration, we define a CustomDefinitionResource called JavaAgent.

When the pod is injected, the pod will be labeled with `sidecar.skywalking.apache.org/succeed`, then the controller will watch the specific pod labeled with `sidecar.skywalking.apache.org/succeed`. After the pod is created, the controller will create JavaAgent(custom resource), which contains the final agent configuration as below.

## Spec

| Field Name         | Description                                                  |
| ------------------ | ------------------------------------------------------------ |
| podSelector        | We hope users can use [workloads](https://kubernetes.io/docs/concepts/workloads/) to create pods, the podSelector is the selector label of workload. |
| serviceName        | serviceName is an important attribute that needs to be printed. |
| backendService     | backendService is an important attribute that needs to be printed. |
| agentConfiguration | agentConfiguration contains serviceName、backendService and covered agent configuration, other default configurations will not be displayed, please see [agent.config](https://skywalking.apache.org/docs/skywalking-java/latest/en/setup/service-agent/java-agent/configurations/#table-of-agent-configuration-properties) for details. |

## Status

| Field Name           | Description                                    |
| -------------------- | ---------------------------------------------- |
| creationTime         | The creation time of the JavaAgent             |
| lastUpdateTime       | The last Update time of the JavaAgent          |
| expectedInjectiedNum | The number of the pod that need to be injected |
| realInjectedNum      | The real number of injected pods.              |

## Demo

This demo shows the usage of javaagent. If you want to see the complete process, please see [java-agent-injector-usage](examples/java-agent-injector-usage.md)for details.

When we use [java-agent-injector](java-agent-injector.md), we can get custom resources as below.

```
$ kubectl get javaagent -A
NAMESPACE   NAME                  PODSELECTOR   SERVICENAME            BACKENDSERVICE
default     app-demo1-javaagent   app=demo1     Your_ApplicationName   127.0.0.1:11800
default     app-demo2-javaagent   app=demo2     Your_ApplicationName   127.0.0.1:11800
$ kubectl get pod -l app=demo1
NAME                    READY   STATUS    RESTARTS   AGE
demo1-bb97b8b4d-bkwm4   1/1     Running   0          28s
demo1-bb97b8b4d-wxgs2   1/1     Running   0          28s
$ kubectl get pod -l app=demo2
NAME     READY   STATUS    RESTARTS   AGE
app2-0   1/1     Running   0          27s
app2-1   1/1     Running   0          25s
app2-2   1/1     Running   0          23s
```

If we want to see more information, we can get the specific javaagent's yaml as below.

```
$ kubectl get javaagent app-demo1-javaagent -oyaml
apiVersion: operator.skywalking.apache.org/v1alpha1
kind: JavaAgent
metadata:
  creationTimestamp: "2021-10-14T07:07:12Z"
  generation: 1
  name: app-demo1-javaagent
  namespace: default
  ownerReferences:
  - apiVersion: apps/v1
    blockOwnerDeletion: true
    controller: true
    kind: ReplicaSet
    name: demo1-bb97b8b4d
    uid: c712924f-4652-4c07-8332-b3938ad72392
  resourceVersion: "330808"
  selfLink: /apis/operator.skywalking.apache.org/v1alpha1/namespaces/default/javaagents/app-demo1-javaagent
  uid: 9350338f-15a5-4832-84d1-530f8d0e1c3b
spec:
  agentConfiguration:
    agent.namespace: default-namespace
    agent.service_name: Your_ApplicationName
    collector.backend_service: 127.0.0.1:11800
  backendService: 127.0.0.1:11800
  podSelector: app=demo1
  serviceName: Your_ApplicationName
status:
  creationTime: "2021-10-14T07:07:12Z"
  expectedInjectiedNum: 2
  lastUpdateTime: "2021-10-14T07:07:14Z"
  realInjectedNum: 2
```

