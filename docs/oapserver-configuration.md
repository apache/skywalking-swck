## OAPSever Configuration Introduction

To configure the OAP Sever, we propose two CRDs: 

- OAPServerConfig: The CRD holds all static configuration, including [environment variable](https://skywalking.apache.org/docs/main/latest/en/setup/backend/configuration-vocabulary/) and [file configuration](https://github.com/apache/skywalking/tree/master/oap-server/server-starter/src/main/resources).
- OAPServerDynamicConfig: The CRD holds all [dynamic configuration](https://skywalking.apache.org/docs/main/latest/en/setup/backend/dynamic-config/).



## Spec of OAPServerConfig

| Field Name | Description                                                  |
| ---------- | ------------------------------------------------------------ |
| Version    | The version of OAP server, the default value is 9.0.0        |
| Env        | The environment variable of OAP server                       |
| File       | The static file in OAP Server, which contains three fields`file.path`、`file.name` and `file.data`.  The `file.path` plus the `file.name`  is the real file that needs to be replaced in the container image, and the `file.data` is the final data in the specific file. |



## Status of OAPServerConfig 

| Field Name            | Description                                          |
| --------------------- | ---------------------------------------------------- |
| Desired  | The number of oapserver that need to be configured   |
| Ready     | The number of oapserver that configured successfully |
| CreationTime          | The time the OAPServerConfig was created.            |
| LastUpdateTime        | The last time this condition was updated.            |



## Demo of OAPServerConfig 

> When using the `file`, please don't set the same name

```yaml
# static configuration of OAPServer
apiVersion: operator.skywalking.apache.org/v1alpha1
kind: OAPServerConfig
metadata:
  name: oapserverconfig-sample
  namespace: skywalking-system
spec:
  # The version of OAPServer
  version: 9.0.0
  # The env configuration of OAPServer
  env:
    - name: JAVA_OPTS
      value: -Xmx2048M
    - name: SW_CLUSTER
      value: kubernetes
    - name: SW_CLUSTER_K8S_NAMESPACE
      value: skywalking-system
    # enable the dynamic configuration
    - name: SW_CONFIGURATION
      value: k8s-configmap
    # set the labelselector of the dynamic configuration
    - name: SW_CLUSTER_K8S_LABEL
      value: app=collector,release=skywalking
    - name: SW_TELEMETRY
      value: prometheus
    - name: SW_HEALTH_CHECKER
      value: default
    - name: SKYWALKING_COLLECTOR_UID
      valueFrom:
        fieldRef:
          fieldPath: metadata.uid
    - name: SW_LOG_LAL_FILES
      value: test1
    - name: SW_LOG_MAL_FILES
      value: test2
  # The file configuration of OAPServer
  # we should avoid setting the same file name in the file
  file:
    - name: test1.yaml
      path: /skywalking/config/lal
      data: |
        rules:
          - name: example
            dsl: |
              filter {
                text {
                  abortOnFailure false // for test purpose, we want to persist all logs
                  regexp $/(?s)(?<timestamp>\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}.\d{3}) \[TID:(?<tid>.+?)] \[(?<thread>.+?)] (?<level>\w{4,}) (?<logger>.{1,36}) (?<msg>.+)/$
                }
                extractor {
                  metrics {
                    timestamp log.timestamp as Long
                    labels level: parsed.level, service: log.service, instance: log.serviceInstance
                    name "log_count"
                    value 1
                  }
                }
                sink {
                }
              }
    - name: test2.yaml
      path: /skywalking/config/log-mal-rules
      data: |
        expSuffix: instance(['service'], ['instance'], Layer.GENERAL)
        metricPrefix: log
        metricsRules:
          - name: count_info
            exp: log_count.tagEqual('level', 'INFO').sum(['service', 'instance']).downsampling(SUM)

```



## Spec of OAPServerDynamicConfig

| Field Name    | Description                                            |
| ------------- | ------------------------------------------------------ |
| Version       | The version of the OAP server, the default value is 9.0.0      |
| LabelSelector | The label selector of the specific configmap, the default value is "app=collector,release=skywalking"               |
| Data          | All configurations' key and value                      |



## Status of OAPServerDynamicConfig

| Field Name     | Description                                                |
| -------------- | ---------------------------------------------------------- |
| State          | The state of dynamic configuration, `running` or `stopped` |
| CreationTime   | All configurations in one CR, the default value is `false`     |
| LastUpdateTime | The last time this condition was updated                   |



## Usage of OAPServerDynamicConfig

> Notice, the CR's name cannot contain capital letters.

Users can split all configurations into several CRs. when using the OAPServerDynamicConfig, users can not only put some configurations in a CR, but also put a configuration in a CR, and the `spec.data.name` in CR represents one dynamic configuration.



#### Demo of Global configuration

```yaml
apiVersion: operator.skywalking.apache.org/v1alpha1
kind: OAPServerDynamicConfig
metadata:
  name: oapserverdynamicconfig-sample
spec:
  # The version of OAPServer
  version: 9.0.0
  # The labelselector of OAPServer's dynamic configuration, it should be the same as labelSelector of OAPServerConfig
  labelSelector: app=collector,release=skywalking
  data:
    - name: agent-analyzer.default.slowDBAccessThreshold
      value: default:200,mongodb:50
    - name: alarm.default.alarm-settings
      value: |-
        rules:
          # Rule unique name, must be ended with `_rule`.
          service_resp_time_rule:
            metrics-name: service_resp_time
            op: ">"
            threshold: 1000
            period: 10
            count: 3
            silence-period: 5
            message: Response time of service {name} is more than 1000ms in 3 minutes of last 10 minutes.
          service_sla_rule:
            # Metrics value need to be long, double or int
            metrics-name: service_sla
            op: "<"
            threshold: 8000
            # The length of time to evaluate the metrics
            period: 10
            # How many times after the metrics match the condition, will trigger alarm
            count: 2
            # How many times of checks, the alarm keeps silence after alarm triggered, default as same as period.
            silence-period: 3
            message: Successful rate of service {name} is lower than 80% in 2 minutes of last 10 minutes
          service_resp_time_percentile_rule:
            # Metrics value need to be long, double or int
            metrics-name: service_percentile
            op: ">"
            threshold: 1000,1000,1000,1000,1000
            period: 10
            count: 3
            silence-period: 5
            message: Percentile response time of service {name} alarm in 3 minutes of last 10 minutes, due to more than one condition of p50 > 1000, p75 > 1000, p90 > 1000, p95 > 1000, p99 > 1000
          service_instance_resp_time_rule:
            metrics-name: service_instance_resp_time
            op: ">"
            threshold: 1000
            period: 10
            count: 2
            silence-period: 5
            message: Response time of service instance {name} is more than 1000ms in 2 minutes of last 10 minutes
          database_access_resp_time_rule:
            metrics-name: database_access_resp_time
            threshold: 1000
            op: ">"
            period: 10
            count: 2
            message: Response time of database access {name} is more than 1000ms in 2 minutes of last 10 minutes
          endpoint_relation_resp_time_rule:
            metrics-name: endpoint_relation_resp_time
            threshold: 1000
            op: ">"
            period: 10
            count: 2
            message: Response time of endpoint relation {name} is more than 1000ms in 2 minutes of last 10 minutes
        #  Active endpoint related metrics alarm will cost more memory than service and service instance metrics alarm.
        #  Because the number of endpoint is much more than service and instance.
        #
        #  endpoint_resp_time_rule:
        #    metrics-name: endpoint_resp_time
        #    op: ">"
        #    threshold: 1000
        #    period: 10
        #    count: 2
        #    silence-period: 5
        #    message: Response time of endpoint {name} is more than 1000ms in 2 minutes of last 10 minutes

        webhooks:
        #  - http://127.0.0.1/notify/
        #  - http://127.0.0.1/go-wechat/
    - name: core.default.apdexThreshold
      value: |-
        default: 500
        # example:
        # the threshold of service "tomcat" is 1s
        # tomcat: 1000
        # the threshold of service "springboot1" is 50ms
        # springboot1: 50
    - name: agent-analyzer.default.uninstrumentedGateways
      value: |-
        #gateways:
        #  - name: proxy0
        #    instances:
        #      - host: 127.0.0.1 # the host/ip of this gateway instance
        #        port: 9099 # the port of this gateway instance, defaults to 80

```



#### Demo of Single configuration

Set the dynamic configuration `agent-analyzer.default.slowDBAccessThreshold` as follows.

```yaml
apiVersion: operator.skywalking.apache.org/v1alpha1
kind: OAPServerDynamicConfig
metadata:
  name: agent-analyzer.default
spec:
  # The version of OAPServer
  version: 9.0.0
  # The labelselector of OAPServer's dynamic configuration, it should be the same as labelSelector of OAPServerConfig
  labelSelector: app=collector,release=skywalking
  data:
    - name: slowDBAccessThreshold
      value: default:200,mongodb:50
```

Set the dynamic configuration `core.default.endpoint-name-grouping-openapi.customerAPI-v1` and `core.default.endpoint-name-grouping-openapi.productAPI-v1` as follows.

```yaml
apiVersion: operator.skywalking.apache.org/v1alpha1
kind: OAPServerDynamicConfig
metadata:
  name: core.default.endpoint-name-grouping-openapi
spec:
  # The version of OAPServer
  version: 9.0.0
  # The labelselector of OAPServer's dynamic configuration, it should be the same as labelSelector of OAPServerConfig
  labelSelector: app=collector,release=skywalking
  data:
    - name: customerAPI-v1
      value: value of customerAPI-v1
    - name: productAPI-v1
    	value: value of productAPI-v1
```

