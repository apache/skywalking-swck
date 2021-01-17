# Fetch metrics from the Istio control plane(istiod)

In this example, you will learn how to setup a Fetcher to fetch Istio control plane metrics,
then push them to OAP server.

## Install Operator

Follow [Operator installation instrument](../../README.md#operator) to install the operator.

## Install Istio control plane

Follow [Install with istioctl](https://istio.io/latest/docs/setup/install/istioctl/) to install a
istiod.

## Deploy Fetcher, OAP server and UI with default settings

Clone this repo, then change current directory to [samples](../../config/operator/samples).

Issue the below command to deploy an OAP server and UI.

```shell
kubectl apply -f fetcher.yaml
```

Get created custom resources as below:

```shell
$ kubectl get oapserver,ui,fetcher
NAME                                               INSTANCES   RUNNING   ADDRESS
oapserver.operator.skywalking.apache.org/default   1           1         default-oap.skywalking-swck-system

NAME                                        INSTANCES   RUNNING   INTERNALADDRESS                     EXTERNALIPS   PORTS
ui.operator.skywalking.apache.org/default   1           1         default-ui.skywalking-swck-system                 [80]

NAME                                                        AGE
fetcher.operator.skywalking.apache.org/istio-prod-cluster   36h
```

# View Istio Control Plane Dashboard from UI

Follow [View the UI](./default-backend.md#view-the-ui) to access the UI service.

Navigate to `Dashboard->Istio Control Plane` to view relevant metric diagrams.
