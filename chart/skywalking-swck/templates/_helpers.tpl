{{/*
Licensed to Apache Software Foundation (ASF) under one or more contributor
license agreements. See the NOTICE file distributed with
this work for additional information regarding copyright
ownership. Apache Software Foundation (ASF) licenses this file to you under
the Apache License, Version 2.0 (the "License"); you may
not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied.  See the License for the
specific language governing permissions and limitations
under the License.
*/}}

{{/*
The chart name, overridable with nameOverride.
*/}}
{{- define "swck.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
The prefix every resource this chart creates is named after. Truncated at 63
characters because that is the limit of most Kubernetes name fields.
*/}}
{{- define "swck.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
"<fullname>-<suffix>", clamped to 63 characters by truncating the fullname part
rather than the suffix -- the suffix is what tells two resources apart, so it is
the half that must survive. Call it as:

    {{ include "swck.componentName" (list . "controller-manager") }}
*/}}
{{- define "swck.componentName" -}}
{{- $top := index . 0 -}}
{{- $suffix := index . 1 -}}
{{- $base := include "swck.fullname" $top | trunc (int (sub 62 (len $suffix))) | trimSuffix "-" -}}
{{- printf "%s-%s" $base $suffix -}}
{{- end -}}

{{- define "swck.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "swck.labels" -}}
helm.sh/chart: {{ include "swck.chart" . }}
app.kubernetes.io/name: {{ include "swck.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Selector labels. These land in Deployment.spec.selector, which is IMMUTABLE, so
they must stay stable across releases of the chart.

`control-plane: <fullname>-controller-manager` is inherited from the kustomize
bundle in operator/config: the metrics Service and the webhook Service both
select on it.
*/}}
{{- define "swck.operator.selectorLabels" -}}
control-plane: {{ include "swck.componentName" (list . "controller-manager") }}
app.kubernetes.io/name: {{ include "swck.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: operator
{{- end -}}

{{- define "swck.adapter.selectorLabels" -}}
app: {{ include "swck.componentName" (list . "custom-metrics-apiserver") }}
app.kubernetes.io/name: {{ include "swck.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: metrics-adapter
{{- end -}}

{{/*
Every label a component's resources carry: the common set, plus the selector
labels that identify the component.

These exist so that metadata.labels is written from ONE helper. Emitting
"swck.labels" and a selector helper into the same mapping repeats
app.kubernetes.io/name and app.kubernetes.io/instance, and a duplicate key in a
YAML mapping is rejected by strict decoding.
*/}}
{{- define "swck.operator.labels" -}}
{{ include "swck.labels" . }}
control-plane: {{ include "swck.componentName" (list . "controller-manager") }}
app.kubernetes.io/component: operator
{{- end -}}

{{- define "swck.adapter.labels" -}}
{{ include "swck.labels" . }}
app: {{ include "swck.componentName" (list . "custom-metrics-apiserver") }}
app.kubernetes.io/component: metrics-adapter
{{- end -}}

{{/*
Image coordinates. Each component falls back to the shared `image.*` block for
any field it does not set itself, and the shared tag falls back to the chart's
appVersion -- so a chart released at 0.11.0 deploys the 0.11.0 image unless it is
told otherwise. See the comment above `image:` in values.yaml for why the two
GHCR images cannot be addressed through the shared repository.
*/}}
{{- define "swck.operator.image" -}}
{{- $repository := default .Values.image.repository .Values.operator.image.repository -}}
{{- $tag := default (default .Chart.AppVersion .Values.image.tag) .Values.operator.image.tag -}}
{{- printf "%s:%s" (required "operator.image.repository or image.repository is required" $repository) (required "operator.image.tag, image.tag or the chart appVersion is required" $tag) -}}
{{- end -}}

{{- define "swck.operator.imagePullPolicy" -}}
{{- default .Values.image.pullPolicy .Values.operator.image.pullPolicy -}}
{{- end -}}

{{- define "swck.adapter.image" -}}
{{- $repository := default .Values.image.repository .Values.adapter.image.repository -}}
{{- $tag := default (default .Chart.AppVersion .Values.image.tag) .Values.adapter.image.tag -}}
{{- printf "%s:%s" (required "adapter.image.repository or image.repository is required" $repository) (required "adapter.image.tag, image.tag or the chart appVersion is required" $tag) -}}
{{- end -}}

{{- define "swck.adapter.imagePullPolicy" -}}
{{- default .Values.image.pullPolicy .Values.adapter.image.pullPolicy -}}
{{- end -}}

{{/*
The service account each component runs as.
*/}}
{{- define "swck.operator.serviceAccountName" -}}
{{- default (include "swck.componentName" (list . "controller-manager")) .Values.operator.serviceAccount.name -}}
{{- end -}}

{{- define "swck.adapter.serviceAccountName" -}}
{{- default (include "swck.componentName" (list . "custom-metrics-apiserver")) .Values.adapter.serviceAccount.name -}}
{{- end -}}

{{/*
The OAP GraphQL endpoint the metrics adapter queries.

`adapter.oap.address` wins when set. Otherwise it is built from
adapter.oap.service.{name,namespace,port}, and `name` is REQUIRED: there is no
sensible default, and a wrong one fails silently -- the APIService reaches
Available whether or not its upstream exists, so every HPA query just returns
nothing. See values.yaml.
*/}}
{{- define "swck.adapter.oapAddress" -}}
{{- if .Values.adapter.oap.address -}}
{{- .Values.adapter.oap.address -}}
{{- else -}}
{{- $name := required "adapter.oap.address or adapter.oap.service.name is required when adapter.enabled is true" .Values.adapter.oap.service.name -}}
{{- $namespace := default .Release.Namespace .Values.adapter.oap.service.namespace -}}
{{- printf "http://%s.%s:%v/graphql" $name $namespace .Values.adapter.oap.service.port -}}
{{- end -}}
{{- end -}}
