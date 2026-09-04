{{/*
Expand the name of the chart.
*/}}
{{- define "nats-auth-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "nats-auth-operator.fullname" -}}
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
{{- define "nats-auth-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "nats-auth-operator.labels" -}}
helm.sh/chart: {{ include "nats-auth-operator.chart" . }}
{{ include "nats-auth-operator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "nats-auth-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "nats-auth-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
control-plane: controller-manager
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "nats-auth-operator.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "nats-auth-operator.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Controller manager service account name
*/}}
{{- define "nats-auth-operator.controllerManagerServiceAccountName" -}}
{{- if .Values.controllerManager.serviceAccount }}
{{- default (printf "%s-controller-manager" (include "nats-auth-operator.fullname" .)) .Values.controllerManager.serviceAccount.name }}
{{- else }}
{{- printf "%s-controller-manager" (include "nats-auth-operator.fullname" .) }}
{{- end }}
{{- end }}

{{/*
Name for the objects that stay cluster scoped even in namespaced mode.

Cluster scoped names are global, so two namespaced releases that happen to share a release name
would fight over them and Helm refuses the second install. Qualifying with the namespace makes
several scoped instances installable without the operator having to know about each other. Cluster
wide mode is left exactly as it was, so existing installs see no rename.
*/}}
{{- define "nats-auth-operator.clusterScopedName" -}}
{{- $root := index . 0 -}}
{{- $suffix := index . 1 -}}
{{- if $root.Values.namespaced -}}
{{ include "nats-auth-operator.fullname" $root }}-{{ $suffix }}-{{ $root.Release.Namespace }}
{{- else -}}
{{ include "nats-auth-operator.fullname" $root }}-{{ $suffix }}
{{- end -}}
{{- end }}
