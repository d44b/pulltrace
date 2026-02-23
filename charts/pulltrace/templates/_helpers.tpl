{{/*
Expand the name of the chart.
*/}}
{{- define "pulltrace.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "pulltrace.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "pulltrace.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "pulltrace.labels" -}}
helm.sh/chart: {{ include "pulltrace.chart" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Agent selector labels
*/}}
{{- define "pulltrace.agent.selectorLabels" -}}
app.kubernetes.io/name: {{ include "pulltrace.name" . }}-agent
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Agent labels
*/}}
{{- define "pulltrace.agent.labels" -}}
{{ include "pulltrace.labels" . }}
{{ include "pulltrace.agent.selectorLabels" . }}
app.kubernetes.io/component: agent
app.kubernetes.io/version: {{ .Values.agent.image.tag | quote }}
{{- end }}

{{/*
Server selector labels
*/}}
{{- define "pulltrace.server.selectorLabels" -}}
app.kubernetes.io/name: {{ include "pulltrace.name" . }}-server
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Server labels
*/}}
{{- define "pulltrace.server.labels" -}}
{{ include "pulltrace.labels" . }}
{{ include "pulltrace.server.selectorLabels" . }}
app.kubernetes.io/component: server
app.kubernetes.io/version: {{ .Values.server.image.tag | quote }}
{{- end }}

{{/*
Agent service account name
*/}}
{{- define "pulltrace.agent.serviceAccountName" -}}
{{ include "pulltrace.fullname" . }}-agent
{{- end }}

{{/*
Server service account name
*/}}
{{- define "pulltrace.server.serviceAccountName" -}}
{{ include "pulltrace.fullname" . }}-server
{{- end }}

{{/*
Validate runtime socket configuration.
Fails helm install/upgrade if runtimeSocket.enabled=true but risksAcknowledged!=true.
*/}}
{{- define "pulltrace.validateRuntimeSocket" -}}
{{- if and .Values.agent.runtimeSocket.enabled (not .Values.agent.runtimeSocket.risksAcknowledged) -}}
{{- fail "\n\nSECURITY ERROR: agent.runtimeSocket.enabled=true requires agent.runtimeSocket.risksAcknowledged=true.\n\nMounting the containerd socket grants the agent read access to all container\nmetadata on the node. Set agent.runtimeSocket.risksAcknowledged=true to confirm\nyou understand the risks.\n\nSee SECURITY.md for the full threat model.\n" -}}
{{- end -}}
{{- end -}}

{{/*
Validate service type.
Warns if service type is not ClusterIP — LoadBalancer/NodePort expose the
unauthenticated API to the network.
*/}}
{{- define "pulltrace.validateServiceType" -}}
{{- if and (ne .Values.server.service.type "ClusterIP") (not .Values.server.service.exposureAcknowledged) -}}
{{- fail (printf "\n\nSECURITY ERROR: server.service.type=%s exposes the Pulltrace API without authentication.\n\nPulltrace has no built-in auth. Anyone with network access can read cluster\ninventory data (node names, pod names, image references).\n\nIf this is intentional, set server.service.exposureAcknowledged=true.\nOtherwise, keep the default ClusterIP and use ingress with an auth proxy.\n" .Values.server.service.type) -}}
{{- end -}}
{{- end -}}
