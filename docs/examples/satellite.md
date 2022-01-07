# Satellite Usage

In this example, you will learn how to use the Satellite.

## Install Satellite

Install the Satellite component.

### Install Operator And Backend

1. Follow [Operator installation instrument](../../README.md#operator) to install the operator.
2. Follow [Deploy OAP server and UI](./default-backend.md) to install backend.

### Deploy Satellite with default setting

1. Deploy the Storage use the below command:

Clone this repo, then change current directory to [samples](../../operator/config/samples).

Issue the below command to deploy an OAP server and UI.

```shell
kubectl apply -f satellite.yaml
```

2. Check the Satellite in Kubernetes:

```shell
$ kubectl get satellite
NAME      INSTANCES   RUNNING   ADDRESS
default   1           1         default-satellite.default
```

## Satellite With HPA

1. Follow [Custom Metrics Adapter](./../custom-metrics-adapter.md) to install the metrics adapter.
2. Update the config in the Satellite `CRD` and re-apply it to activate the metrics service in satellite.
```xml
config:
    - name: SATELLITE_TELEMETRY_EXPORT_TYPE
      value: metrics_service
```
3. Update the config in the OAP `CRD` and re-apply it to activate the satellite MAL.

```xml
config:
    - name: SW_METER_ANALYZER_ACTIVE_FILES
      value: satellite
```

4. Add the HorizontalPodAutoScaler `CRD`, and [update the config file](../../operator/config/samples/satellite-hpa.yaml) the `service` and `target` to your excepted config.
   It's recommend to set the `stabilizationWindowSeconds` and `selectPolicy` of scaling up in HPA, which would help prevent continuous scaling up of pods due to metric delay fluctuations. 
5. Check the HorizontalPodAutoScaler in the Kubernetes:

```shell
$ kubectl get HorizontalPodAutoscaler
NAME       REFERENCE                                TARGETS        MINPODS   MAXPODS   REPLICAS   AGE
hpa-demo   Deployment/skywalking-system-satellite   2/1900, 5/75   1         3         1          92m
```