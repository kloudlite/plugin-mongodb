{{- define "name.prefix" -}}
{{- if eq .Chart.Name .Release.Name}}
{{.Chart.Name}}
{{- else -}}
{{.Release.Name}}-{{.Chart.Name}}
{{- end }}
{{- end -}}

{{- define "deployment.name" -}}
{{include "name.prefix" .}}
{{- end -}}

{{- define "serviceaccount.name" -}}
{{- if .Values.serviceAccount.name }}
{{.Values.serviceAccount.name}}
{{- else -}}
{{ include "name.prefix" .}}-sa
{{- end }}
{{- end -}}

