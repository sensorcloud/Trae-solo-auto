{{- if .Values.imagePullSecrets -}}
imagePullSecrets:
  {{- toYaml .Values.imagePullSecrets | nindent 4 }}
{{- end }}
nameOverride: {{ .Values.nameOverride }}
fullnameOverride: {{ .Values.fullnameOverride }}

serviceAccount:
  create: false

podSecurityContext: {}

securityContext: {}

service:
  type: {{ .Values.service.type }}
  port: {{ .Values.service.port }}

ingress:
  enabled: {{ .Values.ingress.enabled }}
  className: {{ .Values.ingress.className }}
  annotations: {{- toYaml .Values.ingress.annotations | nindent 4 }}
  hosts: {{- toYaml .Values.ingress.hosts | nindent 4 }}
  tls: {{- toYaml .Values.ingress.tls | nindent 4 }}

resources: {{- toYaml .Values.resources | nindent 4 }}

autoscaling:
  enabled: {{ .Values.autoscaling.enabled }}
  minReplicas: {{ .Values.autoscaling.minReplicas }}
  maxReplicas: {{ .Values.autoscaling.maxReplicas }}
  targetCPUUtilizationPercentage: {{ .Values.autoscaling.targetCPUUtilizationPercentage }}
  targetMemoryUtilizationPercentage: {{ .Values.autoscaling.targetMemoryUtilizationPercentage }}

nodeSelector: {{- toYaml .Values.nodeSelector | nindent 4 }}

tolerations: {{- toYaml .Values.tolerations | nindent 4 }}

affinity: {{- toYaml .Values.affinity | nindent 4 }}