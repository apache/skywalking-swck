## 0.7.0

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
