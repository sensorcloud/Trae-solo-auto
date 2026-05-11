{{- define "edgehub.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "edgehub.fullname" -}}
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

{{- define "edgehub.api.fullname" -}}
{{- printf "%s-%s" (include "edgehub.fullname" .) "api" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "edgehub.scheduler.fullname" -}}
{{- printf "%s-%s" (include "edgehub.fullname" .) "scheduler" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "edgehub.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "edgehub.api.labels" -}}
{{- merge (dict "app" "edgehub-api") .Values.commonLabels | noist }}
{{- end }}

{{- define "edgehub.selectorLabels" -}}
{{- merge (dict "app" .Chart.Name "version" .Values.image.tag) .Values.commonLabels | noist }}
{{- end }}
