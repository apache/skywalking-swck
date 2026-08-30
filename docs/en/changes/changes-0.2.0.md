## 0.2.0

#### Features
- Introduce custom metrics adapter to SkyWalking OAP cluster for Kubernetes HPA autoscaling.
- Add RBAC files and service account to support Kubernetes coordination.
- Add default and validation webhooks to operator controllers.
- Add UI CRD to deploy skywalking UI server.
- Add Fetcher CRD to fetch metrics from other telemetry system, for example, Prometheus.

#### Chores
- Transform project layers to support multiple applications.
- Introduce unit test to verify the operator.
