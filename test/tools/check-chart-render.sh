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

# Renders chart/skywalking-swck under a spread of values and asserts on the
# result.
#
# `helm lint` is not a substitute for this and neither is `helm template` on its
# own: BOTH exit 0 on a chart that renders nothing at all, so a chart whose
# templates/ directory went missing passes them both. Every check here looks at
# the output.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# test/tools -> the repository root.
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
CHART_DIR="${CHART_DIR:-${ROOT_DIR}/chart/skywalking-swck}"
YQ="${YQ:-${ROOT_DIR}/bin/yq}"

failures=0
render=""

pass() { printf '  ok   %s\n' "$1"; }
fail() { printf '  FAIL %s\n' "$1"; failures=$((failures + 1)); }

scenario() { printf '\n== %s\n' "$1"; }

# render <release> <namespace> [--set ...]
render() {
  local release="$1" namespace="$2"
  shift 2
  render_out=$(helm template "$release" "$CHART_DIR" --namespace "$namespace" "$@")
}

# has <description> <extended regexp>
has() {
  if grep -Eq "$2" <<<"$render_out"; then pass "$1"; else fail "$1 (no match for: $2)"; fi
}

# hasnt <description> <extended regexp>
hasnt() {
  if grep -Eq "$2" <<<"$render_out"; then fail "$1 (unexpected match for: $2)"; else pass "$1"; fi
}

# kinds_named <kind> -> the metadata.name of every document of that kind
# yq writes a "---" separator between the documents it selects, so every helper
# that reads names back out of the render has to drop those.
names_of() {
  "$YQ" "select(.kind == \"$1\") | .metadata.name" <<<"$render_out" | grep -Ev '^(null|---)$' || true
}

all_names() {
  "$YQ" '.metadata.name' <<<"$render_out" | grep -Ev '^(null|---)$' || true
}

if [ ! -x "$YQ" ]; then
  echo "$YQ not found; run this through 'make chart-lint'" >&2
  exit 1
fi

scenario "default values: the operator, and nothing else"
render swck skywalking-swck-system
has  "operator Deployment"              '^  name: swck-skywalking-swck-operator$'
has  "manager runs /manager"            '^        - /manager$'
has  "manager ClusterRole"              '^  name: swck-skywalking-swck-manager-role$'
has  "manager ClusterRoleBinding"       '^  name: swck-skywalking-swck-manager-rolebinding$'
has  "leader election Role"             '^  name: swck-skywalking-swck-leader-election-role$'
has  "service account"                  '^  name: swck-skywalking-swck-controller-manager$'
has  "manager config ConfigMap"         '^  name: swck-skywalking-swck-manager-config$'
has  "metrics Service"                  'swck-skywalking-swck-controller-manager-metrics-service'
has  "kube-rbac-proxy sidecar"          'quay.io/brancz/kube-rbac-proxy:'
has  "webhook Service"                  'swck-skywalking-swck-webhook-service'
has  "MutatingWebhookConfiguration"     '^kind: MutatingWebhookConfiguration$'
has  "ValidatingWebhookConfiguration"   '^kind: ValidatingWebhookConfiguration$'
has  "cert-manager Certificate"         '^kind: Certificate$'
has  "cert-manager Issuer"              '^kind: Issuer$'
has  "CA injection annotation"          'cert-manager.io/inject-ca-from: skywalking-swck-system/swck-skywalking-swck-serving-cert'
has  "pod injector webhook"             'name: mpod.kb.io'
has  "pod injector is namespace-scoped" 'swck-injection: enabled'
has  "eventexporter webhook"            'name: meventexporter.kb.io'
hasnt "no adapter Deployment"           'swck-skywalking-swck-adapter'
hasnt "no external metrics APIService"  'v1beta1.external.metrics.k8s.io'
hasnt "no adapter binary"               '^        - /adapter$'
# The manager reads its config file once at start-up, so a config change that does
# not roll the pod silently does nothing.
has  "config checksum annotation"       'checksum/config:'

scenario "the CRDs ship with the chart"
render swck skywalking-swck-system --include-crds
crd_files=$(find "$CHART_DIR/crds" -name '*.yaml' | wc -l | tr -d ' ')
crd_rendered=$(names_of CustomResourceDefinition | grep -c . || true)
if [ "$crd_files" -gt 0 ] && [ "$crd_files" -eq "$crd_rendered" ]; then
  pass "$crd_rendered CRDs, one per file in crds/"
else
  fail "crds/ holds $crd_files files but $crd_rendered CRDs rendered"
fi
for crd in banyandbs eventexporters fetchers javaagents oapserverconfigs \
           oapserverdynamicconfigs oapservers satellites storages swagents uis; do
  has "CRD $crd" "^  name: ${crd}.operator.skywalking.apache.org$"
done

scenario "adapter.enabled=true, OAP addressed by service"
render swck skywalking-swck-system \
  --set adapter.enabled=true \
  --set adapter.oap.service.name=my-oap-oap \
  --set adapter.oap.service.namespace=observability
has  "adapter Deployment"               '^  name: swck-skywalking-swck-adapter$'
has  "adapter runs /adapter"            '^        - /adapter$'
has  "OAP address built from service"   '\-\-oap-addr=http://my-oap-oap\.observability:12800/graphql'
has  "external metrics APIService"      '^  name: v1beta1.external.metrics.k8s.io$'
has  "adapter Service"                  '^  name: swck-skywalking-swck-custom-metrics-apiserver$'
has  "auth-delegator binding"           'swck-skywalking-swck-custom-metrics-auth-delegator'
has  "auth-reader in kube-system"       '^  namespace: kube-system$'
has  "HPA controller binding"           'name: horizontal-pod-autoscaler'
has  "the operator is still installed"  '^  name: swck-skywalking-swck-operator$'

scenario "adapter.enabled=true, OAP addressed directly"
render swck skywalking-swck-system \
  --set adapter.enabled=true \
  --set adapter.oap.address=http://oap.elsewhere:12800/graphql
has  "explicit address wins"            '\-\-oap-addr=http://oap\.elsewhere:12800/graphql'

scenario "adapter.enabled=true with no OAP address is rejected"
# The APIService reaches Available whether or not its upstream exists, so an
# unset OAP address fails silently at runtime. It has to fail here instead.
if helm template swck "$CHART_DIR" --set adapter.enabled=true >/dev/null 2>&1; then
  fail "rendered without an OAP address"
else
  pass "rendering is refused"
fi

scenario "adapter defaults its OAP namespace to the release namespace"
render swck my-namespace \
  --set adapter.enabled=true \
  --set adapter.oap.service.name=oap
has  "release namespace used"           '\-\-oap-addr=http://oap\.my-namespace:12800/graphql'

scenario "operator.webhook.enabled=false"
render swck skywalking-swck-system --set operator.webhook.enabled=false
hasnt "no MutatingWebhookConfiguration"   '^kind: MutatingWebhookConfiguration$'
hasnt "no ValidatingWebhookConfiguration" '^kind: ValidatingWebhookConfiguration$'
hasnt "no Certificate"                    '^kind: Certificate$'
hasnt "no webhook Service"                'webhook-service'
hasnt "no serving-certs volume"           'mountPath: /tmp/k8s-webhook-server/serving-certs'
# Without this the manager still starts a TLS webhook server, finds no
# certificate, and never becomes ready.
has  "ENABLE_WEBHOOKS=false is set"       'name: ENABLE_WEBHOOKS'
has  "the operator is still installed"    '^  name: swck-skywalking-swck-operator$'

scenario "operator.metrics.enabled=false"
render swck skywalking-swck-system --set operator.metrics.enabled=false
hasnt "no kube-rbac-proxy sidecar"      'image: quay.io/brancz/kube-rbac-proxy'
hasnt "no metrics Service"              'controller-manager-metrics-service'
hasnt "no proxy ClusterRole"            'swck-skywalking-swck-proxy-role'
has  "metrics endpoint turned off"      'bindAddress: "0"'

scenario "image resolution"
render swck skywalking-swck-system --set adapter.enabled=true --set adapter.oap.service.name=oap
has  "operator falls back to the shared repository and the chart appVersion" \
     'image: docker.io/apache/skywalking-swck:[0-9]'
render swck skywalking-swck-system \
  --set adapter.enabled=true --set adapter.oap.service.name=oap \
  --set operator.image.repository=ghcr.io/apache/skywalking-swck/operator \
  --set operator.image.tag=latest \
  --set adapter.image.repository=ghcr.io/apache/skywalking-swck/metrics-adapter \
  --set adapter.image.tag=latest
has  "split GHCR operator image"        'image: ghcr.io/apache/skywalking-swck/operator:latest'
has  "split GHCR adapter image"         'image: ghcr.io/apache/skywalking-swck/metrics-adapter:latest'

scenario "externally managed service accounts"
render swck skywalking-swck-system \
  --set adapter.enabled=true --set adapter.oap.service.name=oap \
  --set operator.serviceAccount.name=my-operator-sa \
  --set adapter.serviceAccount.name=my-adapter-sa
has  "operator runs as the given account"  'serviceAccountName: my-operator-sa'
has  "adapter runs as the given account"   'serviceAccountName: my-adapter-sa'
has  "bindings follow the given account"   '^  name: my-operator-sa$'
sa_count=$(names_of ServiceAccount | grep -c . || true)
if [ "$sa_count" -eq 0 ]; then
  pass "no ServiceAccount is created"
else
  fail "created $sa_count ServiceAccounts even though names were supplied"
fi

scenario "names stay within the 63 character limit"
# Every resource is named after the release, so a long release name is what
# pushes generated names past the limit Kubernetes enforces.
# 53 characters, which is the longest release name helm will accept.
render a-very-long-release-name-for-the-skywalking-swck skywalking-swck-system \
  --set adapter.enabled=true --set adapter.oap.service.name=oap
too_long=$(all_names | awk 'length > 63' || true)
if [ -z "$too_long" ]; then
  pass "no name exceeds 63 characters"
else
  fail "names exceed 63 characters: $too_long"
fi
invalid=$(all_names | grep -Ev '^[a-z0-9]([-.a-z0-9]*[a-z0-9])?$' || true)
if [ -z "$invalid" ]; then
  pass "every name is a valid DNS subdomain"
else
  fail "invalid names: $invalid"
fi

scenario "no rendered document has a duplicate key"
# Emitting two label helpers into one mapping repeats app.kubernetes.io/name and
# app.kubernetes.io/instance. Nothing here would notice -- helm renders it, `kubectl apply` accepts
# it with last-one-wins -- until a strict decoder, or a server-side apply, rejects the manifest.
# yq does not object to duplicate keys either, so this is checked with a parser that does.
if ! python3 -c 'import yaml' 2>/dev/null; then
  fail "python3 with PyYAML is required for the duplicate key check"
else
  for values in \
    "" \
    "--set adapter.enabled=true --set adapter.oap.service.name=oap" \
    "--set operator.webhook.enabled=false" \
    "--set operator.metrics.enabled=false"; do
    # shellcheck disable=SC2086
    duplicates=$(helm template swck "$CHART_DIR" $values | python3 -c '
import sys, yaml

found = []

class Check(yaml.SafeLoader):
    pass

def mapping(loader, node, deep=False):
    seen = set()
    for key_node, _ in node.value:
        key = loader.construct_object(key_node, deep=deep)
        if key in seen:
            found.append("%s (line %d)" % (key, key_node.start_mark.line + 1))
        seen.add(key)
    return yaml.SafeLoader.construct_mapping(loader, node, deep)

Check.add_constructor(yaml.resolver.BaseResolver.DEFAULT_MAPPING_TAG, mapping)
list(yaml.load_all(sys.stdin.read(), Loader=Check))
print("\n".join(found))
')
    if [ -n "${duplicates}" ]; then
      fail "duplicate keys with values \"${values:-<defaults>}\": ${duplicates}"
    else
      pass "no duplicate keys with values ${values:-<defaults>}"
    fi
  done
fi

printf '\n'
if [ "$failures" -ne 0 ]; then
  echo "$failures check(s) failed"
  exit 1
fi
echo "all chart render checks passed"
