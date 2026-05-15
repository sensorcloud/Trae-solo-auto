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

{{- define "edgehub.energy.fullname" -}}
{{- printf "%s-%s" (include "edgehub.fullname" .) "energy" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "edgehub.agent.fullname" -}}
{{- printf "%s-%s" (include "edgehub.fullname" .) "agent" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "edgehub.iot.fullname" -}}
{{- printf "%s-%s" (include "edgehub.fullname" .) "iot" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "edgehub.coordination.fullname" -}}
{{- printf "%s-%s" (include "edgehub.fullname" .) "coordination" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "edgehub.nats.fullname" -}}
{{- printf "%s-%s" (include "edgehub.fullname" .) "nats" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "edgehub.mqtt.fullname" -}}
{{- printf "%s-%s" (include "edgehub.fullname" .) "mqtt" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "edgehub.energy.labels" -}}
app: {{ include "edgehub.energy.fullname" . }}
version: {{ .Values.image.tag | default "latest" | quote }}
{{- if .Values.commonLabels }}
{{ toYaml .Values.commonLabels }}
{{- end }}
{{- end }}

{{- define "edgehub.agent.labels" -}}
app: {{ include "edgehub.agent.fullname" . }}
version: {{ .Values.image.tag | default "latest" | quote }}
{{- if .Values.commonLabels }}
{{ toYaml .Values.commonLabels }}
{{- end }}
{{- end }}

{{- define "edgehub.iot.labels" -}}
app: {{ include "edgehub.iot.fullname" . }}
version: {{ .Values.image.tag | default "latest" | quote }}
{{- if .Values.commonLabels }}
{{ toYaml .Values.commonLabels }}
{{- end }}
{{- end }}

{{- define "edgehub.coordination.labels" -}}
app: {{ include "edgehub.coordination.fullname" . }}
version: {{ .Values.image.tag | default "latest" | quote }}
{{- if .Values.commonLabels }}
{{ toYaml .Values.commonLabels }}
{{- end }}
{{- end }}
