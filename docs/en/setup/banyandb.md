# BanyanDB storage

SkyWalking removed H2 permanently in 10.2.0, so no OAP version this operator supports (10.4.0 and
later) has an embedded storage, and `storage.selector` defaults to `banyandb`. There is no fallback:
an `OAPServer` with nowhere to write starts, looks for a BanyanDB on localhost, and never becomes
ready. Every OAP this operator deploys needs a storage, and BanyanDB is the one that needs nothing
else standing.

Two resources are involved, and the split matters:

* **`BanyanDB`** runs the database. The operator creates its Deployment, its gRPC Service on
  `17912` and its HTTP Service on `17913`.
* **`Storage`** is the *pointer* an `OAPServer` follows. For `type: banyandb` it never provisions
  anything — that is the `BanyanDB` resource's job — so it is always `connectType: external`, and
  the webhook rejects `internal` with a message saying as much.

## Getting one running

```yaml
apiVersion: operator.skywalking.apache.org/v1alpha1
kind: BanyanDB
metadata:
  name: storage
  namespace: skywalking-system
spec:
  version: 0.11.0
  counts: 1
  image: apache/skywalking-banyandb:0.11.0
  config:
    - "standalone"
---
apiVersion: operator.skywalking.apache.org/v1alpha1
kind: Storage
metadata:
  name: storage
  namespace: skywalking-system
spec:
  type: banyandb
  connectType: external
  # <banyandb name>-banyandb-grpc.<namespace>:17912
  address: storage-banyandb-grpc.skywalking-system:17912
  version: 0.11.0
---
apiVersion: operator.skywalking.apache.org/v1alpha1
kind: OAPServer
metadata:
  name: skywalking-system
  namespace: skywalking-system
spec:
  version: 11.0.0
  instances: 1
  image: apache/skywalking-oap-server:11.0.0
  storage:
    name: storage
```

The operator turns that into `SW_STORAGE=banyandb` and `SW_STORAGE_BANYANDB_TARGETS` on the OAP.

**Match the versions.** OAP 11.0.0 accepts BanyanDB server API 0.11 only, and OAP 10.4 needs
0.10.3. There is no negotiation: a mismatch fails at startup.

## The endpoint

`address` is a `host:port`, and the port is BanyanDB's **gRPC** port — `17912`, not the `17913`
HTTP one. If you leave the port off, the operator appends `17912`.

The Service the `BanyanDB` resource creates is named `<name>-banyandb-grpc`, so from another
namespace the address is `<name>-banyandb-grpc.<namespace>:17912`.

A BanyanDB the operator did not deploy works just as well — point `address` at whatever host serves
its gRPC port.

## Clusters

`SW_STORAGE_BANYANDB_TARGETS` is a **list**, so `address` may hold several comma-separated
endpoints and the OAP will spread its traffic across them:

```yaml
spec:
  type: banyandb
  connectType: external
  address: "bdb-0.bdb.svc:17912,bdb-1.bdb.svc:17912,bdb-2.bdb.svc:17912"
```

The `BanyanDB` resource itself runs BanyanDB **standalone** — `spec.counts` scales the Deployment,
it does not turn on BanyanDB's liaison/data cluster roles. For a real cluster, deploy it with
[`skywalking-banyandb-helm`](https://github.com/apache/skywalking-banyandb-helm) or by hand and
point a `Storage` at the liaison endpoints.

## Authentication

Name a secret with `username` and `password` keys and the operator points
`SW_STORAGE_BANYANDB_USER` and `SW_STORAGE_BANYANDB_PASSWORD` at it with a `secretKeyRef`. The
values are read by the kubelet when the pod starts — they are never copied into the `Storage` or
the `OAPServer`, so they do not show up in `kubectl get -o yaml`:

```yaml
spec:
  type: banyandb
  connectType: external
  address: storage-banyandb-grpc.skywalking-system:17912
  security:
    user:
      secretName: banyandb-credentials
```

```shell
kubectl create secret generic banyandb-credentials -n skywalking-system \
  --from-literal=username=admin --from-literal=password=...
```

## TLS

Set `security.tls` and name a secret holding the CA certificate under the key `ca.crt`. The
operator mounts it into the OAP pod at `/skywalking/bydb-tls` and sets
`SW_STORAGE_BANYANDB_SSL_TRUST_CA_PATH` to the file:

```yaml
spec:
  type: banyandb
  connectType: external
  address: storage-banyandb-grpc.skywalking-system:17912
  security:
    tls: true
    tlsSecretName: banyandb-ca
    user:
      secretName: banyandb-credentials
```

```shell
kubectl create secret generic banyandb-ca -n skywalking-system --from-file=ca.crt=./ca.crt
```

`tlsSecretName` is required when `tls` is true for `banyandb`, and the webhook says so rather than
letting the OAP pod sit unschedulable waiting on a secret nothing creates. `ca.crt` is the key
cert-manager and `kubernetes.io/tls` secrets already use, so a cert-manager-issued CA can be named
directly.

## Persistence

The example above keeps its data in the container filesystem, which is right for a test cluster and
wrong for anything else. `spec.storages` attaches persistent volumes:

```yaml
spec:
  config:
    - "standalone"
    - "--measure-root-path=/data/banyandb"
    - "--stream-root-path=/data/banyandb"
    - "--schema-server-root-path=/data/banyandb"
  storages:
    - name: banyandb-volume
      path: "/data/banyandb"
      persistentVolumeClaimSpec:
        resources:
          requests:
            storage: 10Gi
        volumeMode: Filesystem
        accessModes:
          - ReadWriteOnce
```

`--metadata-root-path`, which older examples pass, was removed from BanyanDB along with etcd.
BanyanDB **exits on an unknown flag** rather than ignoring it, so an old config stops the database
from starting at all — and the OAP then fails with a refused connection, which is a confusing way
to learn about a bad flag.

A worked example is in
[`test/e2e/deploy-banyandb.yaml`](../../../test/e2e/deploy-banyandb.yaml), which also pins the
Deployment to a labelled node.

## Elasticsearch instead

Elasticsearch is the other storage OAP 11 accepts, and unlike BanyanDB the `Storage` resource can
provision it (`connectType: internal`). See [Deploy a storage](../examples/storage.md).
