{{- define "ecommerce-api.name" -}}
{{- .Chart.Name }}
{{- end }}

{{- define "ecommerce-api.fullname" -}}
{{- printf "%s-%s" .Release.Name .Chart.Name | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "ecommerce-api.labels" -}}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version }}
{{ include "ecommerce-api.selectorLabels" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "ecommerce-api.selectorLabels" -}}
app.kubernetes.io/name: {{ include "ecommerce-api.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
