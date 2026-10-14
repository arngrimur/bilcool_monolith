{{/*
Expand the name of the chart.
*/}}
{{- define "bilcool.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "bilcool.fullname" -}}
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
{{- define "bilcool.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "bilcool.labels" -}}
helm.sh/chart: {{ include "bilcool.chart" . }}
{{ include "bilcool.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "bilcool.selectorLabels" -}}
app.kubernetes.io/name: {{ include "bilcool.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "bilcool.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "bilcool.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/* Hook annotations: resources that must exist before the migrate Jobs run. Args: weight */}}
{{- define "bilcool.preHook" -}}
helm.sh/hook: pre-install,pre-upgrade
helm.sh/hook-weight: {{ . | quote }}
helm.sh/hook-delete-policy: before-hook-creation
{{- end }}

{{/* Database URL of a service. Args: (list $ "bookings"). Explicit value wins, else derived from the in-cluster Postgres. */}}
{{- define "bilcool.dbUrl" -}}
{{- $root := index . 0 -}}
{{- $svc := index . 1 -}}
{{- $explicit := (index $root.Values.secrets $svc).databaseUrl -}}
{{- if $explicit -}}
{{- $explicit -}}
{{- else -}}
{{- printf "postgres://%s:%s@postgres:5432/bilcool?sslmode=disable" $svc (index $root.Values.secrets.postgres.passwords $svc) -}}
{{- end -}}
{{- end }}

{{/* Restricted-compatible container security context for the Go services. */}}
{{- define "bilcool.containerSecurity" -}}
allowPrivilegeEscalation: false
capabilities:
  drop: ["ALL"]
{{- end }}
