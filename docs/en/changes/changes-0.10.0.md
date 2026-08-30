## 0.10.0

#### Features
- Support the Horizon UI in the UI CRD through a `spec.kind` discriminator.
- Publish Docker images to ghcr.io on every push to master.

#### Bugs
- Fix the LAL rule missing the `layer` property, for compatibility with the latest OAP.
- Replace the deprecated `scheme.Builder` and update Go to 1.26.

#### Chores
- Bump Go to 1.26.3 in the adapter and 1.25.9 in the operator, to fix stdlib CVEs.
- Bump golang.org/x/net to v0.53.0 to fix CVE-2026-33814.
- Bump go.opentelemetry.io/otel/sdk to v1.43.0, sigs.k8s.io/controller-runtime and other dependencies.
