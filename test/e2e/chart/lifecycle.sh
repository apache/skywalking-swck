#!/usr/bin/env bash

#
# Licensed to Apache Software Foundation (ASF) under one or more contributor
# license agreements. See the NOTICE file distributed with
# this work for additional information regarding copyright
# ownership. Apache Software Foundation (ASF) licenses this file to you under
# the Apache License, Version 2.0 (the "License"); you may
# not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing,
# software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied.  See the License for the
# specific language governing permissions and limitations
# under the License.
#

# Drives the whole lifecycle of the skywalking-swck Helm chart against a live cluster and writes a
# report that test/e2e/chart/e2e.yaml verifies.
#
# The other e2e cases install the chart and then test the OPERATOR. This one tests the CHART: that
# the packaged artifact installs, that it brings its CRDs and its webhook wiring with it, that
# upgrading it adds and removes the adapter, and that uninstalling it does not take the CRDs -- and
# with them every custom resource in the cluster -- along with it.
#
# It installs the packaged tarball, not the source directory, because the tarball is what is
# released and what users install.

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
CHART_DIR="${ROOT_DIR}/chart/skywalking-swck"
RELEASE=swck
NAMESPACE=skywalking-swck-system
PROBE_NAMESPACE=swck-injection-probe
REPORT="${REPORT:-/tmp/swck-chart-report.yaml}"

# The image pins. skywalking-infra-e2e already exports these into every setup step
# (setup.init-system-environment), but source them so the script also runs by hand.
# shellcheck disable=SC1091
set -a; . "${ROOT_DIR}/test/e2e/env"; set +a

results=()
record() { results+=("- key: \"$1\"" "  value: \"$2\""); echo "  $1 = $2"; }

echo "==> packaging the chart"
package_dir=$(mktemp -d)
helm package "${CHART_DIR}" --destination "${package_dir}"
tarball=$(find "${package_dir}" -name 'skywalking-swck-*.tgz' | head -1)
echo "packaged ${tarball}"

echo "==> installing the packaged chart, with the adapter enabled"
helm install "${RELEASE}" "${tarball}" \
  --namespace "${NAMESPACE}" --create-namespace \
  --set operator.image.repository=controller \
  --set operator.image.tag=latest \
  --set operator.image.pullPolicy=IfNotPresent \
  --set adapter.enabled=true \
  --set adapter.image.repository=metrics-adapter \
  --set adapter.image.tag=latest \
  --set adapter.image.pullPolicy=IfNotPresent \
  --set adapter.oap.service.name=skywalking-system-oap \
  --set adapter.oap.service.namespace=skywalking-system \
  --wait --timeout 10m

echo "==> the chart installs the CRDs"
# The chart is the only thing that installs them now that the e2e suites no longer run
# `make -C operator install`, so a chart that ships none installs an operator that can do nothing.
expected_crds=$(find "${ROOT_DIR}/operator/config/crd/bases" -name '*.yaml' | wc -l | tr -d ' ')
installed_crds=$(kubectl get crd -o name | grep -c 'operator.skywalking.apache.org' || true)
record crds_expected "${expected_crds}"
record crds_installed "${installed_crds}"

echo "==> cert-manager injects the CA into both webhook configurations"
# Without the CA bundle the API server cannot call the webhook, and every custom resource is
# rejected -- but the chart installs cleanly and the operator reports ready, so nothing else here
# would notice.
for kind in mutating validating; do
  bundle=$(kubectl get "${kind}webhookconfiguration" "${RELEASE}-skywalking-swck-${kind}-webhook-configuration" \
    -o jsonpath='{.webhooks[0].clientConfig.caBundle}' 2>/dev/null || true)
  if [ -n "${bundle}" ]; then record "${kind}_webhook_ca_injected" true; else record "${kind}_webhook_ca_injected" false; fi
done

echo "==> the webhook configurations cover every kind the operator serves"
# Generated from the operator sources by tools/generate-chart.sh; if generation regresses, the count
# drops here rather than at the first custom resource nobody defaulted.
record mutating_webhooks "$(kubectl get mutatingwebhookconfiguration "${RELEASE}-skywalking-swck-mutating-webhook-configuration" -o jsonpath='{.webhooks[*].name}' | wc -w | tr -d ' ')"
record validating_webhooks "$(kubectl get validatingwebhookconfiguration "${RELEASE}-skywalking-swck-validating-webhook-configuration" -o jsonpath='{.webhooks[*].name}' | wc -w | tr -d ' ')"

echo "==> the operator reconciles a custom resource"
kubectl create namespace skywalking-system --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -f - <<EOF
apiVersion: operator.skywalking.apache.org/v1alpha1
kind: OAPServer
metadata:
  name: skywalking-system
  namespace: skywalking-system
spec:
  version: 11.0.0
  instances: 1
  image: ${OAP_IMAGE}
  # Storage is mandatory -- SkyWalking has had no embedded H2 since 10.2.0 and there is no
  # fallback -- but this case only proves the operator reconciles the resource into a Deployment
  # and a Service. Naming the storage in spec.config satisfies that without standing up a real
  # BanyanDB the case would then have to wait for and tear down.
  config:
    - name: SW_STORAGE
      value: banyandb
    - name: SW_STORAGE_BANYANDB_TARGETS
      value: storage-banyandb-grpc.skywalking-system:17912
  service:
    template:
      type: ClusterIP
EOF
# The Deployment and Service appearing proves the CRD, both webhooks, the operator's ClusterRole and
# its ServiceAccount binding are all in place. Whether OAP itself starts is what the other cases
# are for, so this does not wait for a running pod.
for _ in $(seq 1 30); do
  if kubectl get deployment skywalking-system-oap -n skywalking-system >/dev/null 2>&1; then break; fi
  sleep 5
done
kubectl get deployment,service -n skywalking-system
record oapserver_deployment_created "$(kubectl get deployment skywalking-system-oap -n skywalking-system -o name 2>/dev/null | wc -l | tr -d ' ')"

echo "==> the Java agent injector mutates pods in namespaces that opt in"
# Admission happens when the pod is created, so this proves the injector without waiting for the
# agent image to pull or the application to start.
kubectl create namespace "${PROBE_NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -
kubectl label namespace "${PROBE_NAMESPACE}" swck-injection=enabled --overwrite
kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: injector-probe
  namespace: ${PROBE_NAMESPACE}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: injector-probe
  template:
    metadata:
      labels:
        app: injector-probe
        swck-java-agent-injected: "true"
      annotations:
        agent.skywalking.apache.org/collector.backend_service: "skywalking-system-oap.skywalking-system:11800"
    spec:
      containers:
      - name: app
        image: ${DEMO_IMAGE}
        command: ["java"]
        args: ["-jar","/app.jar"]
EOF
for _ in $(seq 1 30); do
  if [ -n "$(kubectl get pod -n "${PROBE_NAMESPACE}" -l app=injector-probe -o name 2>/dev/null)" ]; then break; fi
  sleep 2
done
probe_pod=$(kubectl get pod -n "${PROBE_NAMESPACE}" -l app=injector-probe -o name | head -1)
kubectl get "${probe_pod}" -n "${PROBE_NAMESPACE}" -o yaml | grep -E 'name: inject-skywalking-agent|JAVA_TOOL_OPTIONS' || true
record agent_initcontainer_injected \
  "$(kubectl get "${probe_pod}" -n "${PROBE_NAMESPACE}" -o jsonpath='{.spec.initContainers[?(@.name=="inject-skywalking-agent")].name}' | wc -w | tr -d ' ')"
record agent_javaagent_env_set \
  "$(kubectl get "${probe_pod}" -n "${PROBE_NAMESPACE}" -o jsonpath='{.spec.containers[0].env[?(@.name=="JAVA_TOOL_OPTIONS")].value}' | grep -c 'skywalking-agent.jar' || true)"

echo "==> the metrics adapter owns the external metrics API"
kubectl wait --for=condition=Available apiservice/v1beta1.external.metrics.k8s.io --timeout=300s
record external_metrics_apiservice_available "$(kubectl get apiservice v1beta1.external.metrics.k8s.io -o jsonpath='{.status.conditions[?(@.type=="Available")].status}')"
# An aggregated apiserver that answers at all proves the Service, the APIService, the auth-delegator
# binding and the kube-system auth-reader binding are all wired up.
record external_metrics_api_serves \
  "$(kubectl get --raw '/apis/external.metrics.k8s.io/v1beta1' | grep -c 'APIResourceList' || true)"

echo "==> the operator metrics endpoint is served, and protected, by kube-rbac-proxy"
metrics_service="${RELEASE}-skywalking-swck-controller-manager-metrics-service"
record metrics_service_exists "$(kubectl get service "${metrics_service}" -n "${NAMESPACE}" -o name 2>/dev/null | wc -l | tr -d ' ')"
record kube_rbac_proxy_running \
  "$(kubectl get deployment "${RELEASE}-skywalking-swck-operator" -n "${NAMESPACE}" -o jsonpath='{.spec.template.spec.containers[?(@.name=="kube-rbac-proxy")].name}' | wc -w | tr -d ' ')"

echo "==> helm upgrade removes the adapter"
helm upgrade "${RELEASE}" "${tarball}" \
  --namespace "${NAMESPACE}" \
  --set operator.image.repository=controller \
  --set operator.image.tag=latest \
  --set operator.image.pullPolicy=IfNotPresent \
  --set adapter.enabled=false \
  --wait --timeout 5m
record adapter_removed_on_upgrade \
  "$(kubectl get apiservice v1beta1.external.metrics.k8s.io -o name 2>/dev/null | wc -l | tr -d ' ')"

echo "==> helm upgrade adds it back"
helm upgrade "${RELEASE}" "${tarball}" \
  --namespace "${NAMESPACE}" \
  --set operator.image.repository=controller \
  --set operator.image.tag=latest \
  --set operator.image.pullPolicy=IfNotPresent \
  --set adapter.enabled=true \
  --set adapter.image.repository=metrics-adapter \
  --set adapter.image.tag=latest \
  --set adapter.image.pullPolicy=IfNotPresent \
  --set adapter.oap.address=http://skywalking-system-oap.skywalking-system:12800/graphql \
  --wait --timeout 5m
record adapter_restored_on_upgrade \
  "$(kubectl get apiservice v1beta1.external.metrics.k8s.io -o name 2>/dev/null | wc -l | tr -d ' ')"
# The explicit address must win over the service fields, and reach the container arguments.
record adapter_uses_explicit_oap_address \
  "$(kubectl get deployment "${RELEASE}-skywalking-swck-adapter" -n "${NAMESPACE}" \
      -o jsonpath='{.spec.template.spec.containers[0].args}' | grep -c 'http://skywalking-system-oap.skywalking-system:12800/graphql' || true)"

echo "==> helm uninstall leaves the CRDs, and the custom resources, alone"
# CRDs live in the chart's crds/ directory rather than in templates/ precisely so this is true:
# deleting a CRD deletes every custom resource of that kind in the cluster with it.
helm uninstall "${RELEASE}" --namespace "${NAMESPACE}"
sleep 10
record crds_after_uninstall "$(kubectl get crd -o name | grep -c 'operator.skywalking.apache.org' || true)"
record oapservers_after_uninstall "$(kubectl get oapserver -n skywalking-system -o name | wc -l | tr -d ' ')"
# Nothing cluster-scoped may survive: those are the resources a second install would collide with.
record cluster_scoped_leftovers \
  "$(kubectl get clusterrole,clusterrolebinding,mutatingwebhookconfiguration,validatingwebhookconfiguration,apiservice -o name 2>/dev/null \
      | grep -c "${RELEASE}-skywalking-swck" || true)"
# The APIService is the one cluster-scoped resource NOT named after the release --
# v1beta1.external.metrics.k8s.io is a fixed, cluster-wide singleton -- so the count above cannot
# see it. Leaving it behind takes the external metrics API group away from the whole cluster until
# someone deletes it by hand, so check it by name.
record external_metrics_apiservice_after_uninstall \
  "$(kubectl get apiservice v1beta1.external.metrics.k8s.io -o name 2>/dev/null | wc -l | tr -d ' ')"

printf '%s\n' "${results[@]}" > "${REPORT}"
echo "==> report written to ${REPORT}"
cat "${REPORT}"
