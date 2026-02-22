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
