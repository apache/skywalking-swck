Changes by Version
==================
Release Notes.

0.7.0
------------------

#### Features
- Replace go-bindata with embed lib.
- Add the OAPServerConfig CRD, webhooks and controller.
- Add the OAPServerDynamicConfig CRD, webhooks and controller.
- Add the SwAgent CRD, webhooks and controller.
- [Breaking Change] Remove the way to configure the agent through Configmap.

#### Bugs
- Fix the error in e2e testing.
- Fix status inconsistent with CI.
- Bump up prometheus client version to fix cve.

#### Chores
- Bump several dependencies of adapter.
- Update license eye version.
- Bump up SkyWalking OAP to 9.0.0.
- Bump up the k8s api of the e2e environment to v1.21.10.

0.6.1
------------------

#### Bugs
- Fix could not deploy metrics adapter to GKE

0.6.0
------------------

#### Features
- Add the Satellite CRD, webhooks and controller

#### Bugs
- Update release images to set numeric user id 
- Fix the satellite config not support number error
- Use env JAVA_TOOL_OPTIONS to replace AGENT_OPTS
#### Chores
- Add stabilization windows feature in satellite HPA documentation

0.5.0
------------------

#### Features
- Add E2E test cases to verify OAPServer, UI, Java agent and Storage components.
- Add the Satellite component.

### Bugs

- Fix operator role patch issues
- Fix invalid CSR signername
- Fix bug in the configmap controller

#### Chores
- Bump up KubeBuilder to V3
- Bump up metric adapter server to v1.21.0
- Split mono-project to two independent projects

0.4.0
------------------

#### Features
- Add Java agent injector.
- Add JavaAgent and Storage CRDs of the operator.

### Vulnerabilities

- CVE-2021-3121: An issue was discovered in GoGo Protobuf before 1.3.2. plugin/unmarshal/unmarshal.go lacks certain index validation
- CVE-2020-29652: A nil pointer dereference in the golang.org/x/crypto/ssh component through v0.0.0-20201203163018-be400aefbc4c for Go allows remote attackers to cause a denial of service against SSH servers.

#### Chores
- Bump up GO to 1.17.
- Bump up k8s api to 0.20.11.
- Polish documents.
- Bump up SkyWalking OAP to 8.8.1.

0.3.0
------------------

#### Features
- Support special characters in the metric selector of HPA metric adapter.
- Add the namespace to HPA metric name.

#### Chores
- Upgrade skywalking-cli dependency.

0.2.0
------------------

#### Features
- Introduce custom metrics adapter to SkyWalking OAP cluster for Kubernetes HPA autoscaling.
- Add RBAC files and service account to support Kubernetes coordination.
- Add default and validation webhooks to operator controllers.
- Add UI CRD to deploy skywalking UI server.
- Add Fetcher CRD to fetch metrics from other telemetry system, for example, Prometheus.

#### Chores
- Transform project layers to support multiple applications.
- Introduce unit test to verify the operator.

0.1.0
------------------

#### Features
- Add OAPServer CRDs and controller.

#### Chores
- Set up GitHub actions to build from sources, check code styles, licenses.
