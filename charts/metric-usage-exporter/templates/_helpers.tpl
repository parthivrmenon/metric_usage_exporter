{{/*
Expand the name of the chart.
*/}}
{{- define "metric-usage-exporter.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "metric-usage-exporter.fullname" -}}
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

{{- define "metric-usage-exporter.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "metric-usage-exporter.labels" -}}
helm.sh/chart: {{ include "metric-usage-exporter.chart" . }}
{{ include "metric-usage-exporter.selectorLabels" . }}
app.kubernetes.io/version: {{ .Values.image.tag | default .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "metric-usage-exporter.selectorLabels" -}}
app.kubernetes.io/name: {{ include "metric-usage-exporter.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "metric-usage-exporter.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "metric-usage-exporter.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Name of the Secret holding Grafana credentials.
*/}}
{{- define "metric-usage-exporter.secretName" -}}
{{- default (include "metric-usage-exporter.fullname" .) .Values.grafana.auth.existingSecret }}
{{- end }}

{{/*
Validate auth settings; fails the render with a clear message.
*/}}
{{- define "metric-usage-exporter.validate" -}}
{{- $auth := .Values.grafana.auth }}
{{- if not (has $auth.type (list "apiKey" "basic")) }}
{{- fail (printf "grafana.auth.type must be \"apiKey\" or \"basic\", got %q" $auth.type) }}
{{- end }}
{{- if not $auth.existingSecret }}
{{- if and (eq $auth.type "apiKey") (not $auth.apiKey) }}
{{- fail "grafana.auth.apiKey (or grafana.auth.existingSecret) is required when grafana.auth.type is apiKey" }}
{{- end }}
{{- if and (eq $auth.type "basic") (or (not $auth.user) (not $auth.password)) }}
{{- fail "grafana.auth.user and grafana.auth.password (or grafana.auth.existingSecret) are required when grafana.auth.type is basic" }}
{{- end }}
{{- end }}
{{- end }}
