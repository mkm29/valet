{{/*
Retrieve configMap name from the name of the chart or the ConfigMap the user
specified.
*/}}
{{- define "alloy.config-map.name" -}}
{{- $values := (mustMergeOverwrite .Values.alloy (or .Values.agent dict)) -}}
{{- if $values.configMap.name -}}
{{- $values.configMap.name }}
{{- else -}}
{{- include "alloy.fullname" . }}
{{- end }}
{{- end }}

{{/*
The name of the config file is the default or the key the user specified in the
ConfigMap.
*/}}
{{- define "alloy.config-map.key" -}}
{{- $values := (mustMergeOverwrite .Values.alloy (or .Values.agent dict)) -}}
{{- if $values.configMap.key -}}
{{- $values.configMap.key }}
{{- else -}}
config.alloy
{{- end }}
{{- end }}

{{- define "alloy.lokiRemoteConfig" -}}
loki.write "default" {
  external_labels = {
    {{- range $key, $value := .Values.global.extraLabels }}
    {{ $key }} = "{{ $value }}",
    {{- end }}
  }
  endpoint {
    {{- with .Values.global.loki }}
    url = "{{ .service.protocol }}://{{ .service.url }}:{{ .service.port }}/{{ .service.prefix }}"
    headers = {
      "X-Scope-OrgId" = "{{ .orgID }}",
    }
    {{- end }}
  }
}
{{- end -}}

{{- define "alloy.prometheusRemoteConfig" -}}
prometheus.remote_write "mimir" {
  external_labels = {
    {{- range $key, $value := .Values.global.extraLabels }}
    {{ $key }} = "{{ $value }}",
    {{- end }}
  }
  endpoint {
    {{- with .Values.global.prometheus }}
    url = "{{ .service.protocol }}://{{ .service.url }}:{{ .service.port }}/{{ .service.prefix }}"
    headers = {
      "X-Scope-OrgId" = "{{ .orgID }}",
    }
    {{- end }}
  }
}
{{- end -}}