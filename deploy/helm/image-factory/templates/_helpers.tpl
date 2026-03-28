{{- define "image-factory.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "image-factory.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "image-factory.labels" -}}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}
app.kubernetes.io/name: {{ include "image-factory.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- range $k, $v := .Values.commonLabels }}
{{ $k }}: {{ $v | quote }}
{{- end }}
{{- end -}}

{{- define "image-factory.selectorLabels" -}}
app.kubernetes.io/name: {{ include "image-factory.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "image-factory.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "image-factory.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{- define "image-factory.backendServiceName" -}}
{{- printf "%s-backend" (include "image-factory.fullname" .) -}}
{{- end -}}

{{/*
Validate database mode and required values. This intentionally fails fast to
avoid silent fallback between external and in-cluster databases.
*/}}
{{- define "image-factory.validateDatabaseConfig" -}}
{{- $mode := required "database.mode is required and must be one of: incluster, external" .Values.database.mode -}}
{{- if and (ne $mode "incluster") (ne $mode "external") -}}
{{- fail (printf "invalid database.mode=%q (allowed: incluster, external)" $mode) -}}
{{- end -}}

{{- if eq $mode "external" -}}
  {{- if .Values.postgres.enabled -}}
    {{- fail "database.mode=external requires postgres.enabled=false" -}}
  {{- end -}}
  {{- $_ := required "database.host is required when database.mode=external" .Values.database.host -}}
  {{- $_ := required "database.name is required when database.mode=external" .Values.database.name -}}
  {{- $_ := required "database.user is required when database.mode=external" .Values.database.user -}}
  {{- $_ := required "database.password is required when database.mode=external" .Values.database.password -}}
{{- end -}}

{{- if eq $mode "incluster" -}}
  {{- if not .Values.postgres.enabled -}}
    {{- fail "database.mode=incluster requires postgres.enabled=true" -}}
  {{- end -}}
  {{- if ne (default "" .Values.database.host) "" -}}
    {{- fail "database.host must be empty when database.mode=incluster (host is derived from release service name)" -}}
  {{- end -}}
  {{- if ne (default "" .Values.database.name) "" -}}
    {{- fail "database.name must be empty when database.mode=incluster (name comes from postgres.database)" -}}
  {{- end -}}
  {{- if ne (default "" .Values.database.user) "" -}}
    {{- fail "database.user must be empty when database.mode=incluster (user comes from postgres.username)" -}}
  {{- end -}}
  {{- if ne (default "" .Values.database.password) "" -}}
    {{- fail "database.password must be empty when database.mode=incluster (password comes from postgres.password)" -}}
  {{- end -}}
{{- end -}}
{{- end -}}

{{/*
Validate strict component configuration and ban silent fallback behavior.
*/}}
{{- define "image-factory.validateChartConfig" -}}
{{- include "image-factory.validateDatabaseConfig" . -}}

{{- if .Values.backend.enabled -}}
  {{- $_ := required "backend.image.repository is required when backend.enabled=true" .Values.backend.image.repository -}}
  {{- $_ := required "backend.image.tag is required when backend.enabled=true" .Values.backend.image.tag -}}
  {{- $_ := required "backend.image.pullPolicy is required when backend.enabled=true" .Values.backend.image.pullPolicy -}}
{{- end -}}

{{- if .Values.frontend.enabled -}}
  {{- $_ := required "frontend.image.repository is required when frontend.enabled=true" .Values.frontend.image.repository -}}
  {{- $_ := required "frontend.image.tag is required when frontend.enabled=true" .Values.frontend.image.tag -}}
  {{- $_ := required "frontend.image.pullPolicy is required when frontend.enabled=true" .Values.frontend.image.pullPolicy -}}
{{- end -}}

{{- if .Values.docs.enabled -}}
  {{- $_ := required "docs.image.repository is required when docs.enabled=true" .Values.docs.image.repository -}}
  {{- $_ := required "docs.image.tag is required when docs.enabled=true" .Values.docs.image.tag -}}
  {{- $_ := required "docs.image.pullPolicy is required when docs.enabled=true" .Values.docs.image.pullPolicy -}}
{{- end -}}

{{- if and .Values.backend.enabled .Values.workers.dispatcher.enabled -}}
  {{- $_ := required "workers.dispatcher.image.repository is required when workers.dispatcher.enabled=true" .Values.workers.dispatcher.image.repository -}}
  {{- $_ := required "workers.dispatcher.image.tag is required when workers.dispatcher.enabled=true" .Values.workers.dispatcher.image.tag -}}
  {{- $_ := required "workers.dispatcher.image.pullPolicy is required when workers.dispatcher.enabled=true" .Values.workers.dispatcher.image.pullPolicy -}}
{{- end -}}

{{- if and .Values.backend.enabled .Values.workers.notification.enabled -}}
  {{- $_ := required "workers.notification.image.repository is required when workers.notification.enabled=true" .Values.workers.notification.image.repository -}}
  {{- $_ := required "workers.notification.image.tag is required when workers.notification.enabled=true" .Values.workers.notification.image.tag -}}
  {{- $_ := required "workers.notification.image.pullPolicy is required when workers.notification.enabled=true" .Values.workers.notification.image.pullPolicy -}}
{{- end -}}

{{- if and .Values.backend.enabled .Values.workers.email.enabled -}}
  {{- $_ := required "workers.email.image.repository is required when workers.email.enabled=true" .Values.workers.email.image.repository -}}
  {{- $_ := required "workers.email.image.tag is required when workers.email.enabled=true" .Values.workers.email.image.tag -}}
  {{- $_ := required "workers.email.image.pullPolicy is required when workers.email.enabled=true" .Values.workers.email.image.pullPolicy -}}
{{- end -}}

{{- if and .Values.backend.enabled .Values.workers.internalRegistryGc.enabled -}}
  {{- $_ := required "workers.internalRegistryGc.image.repository is required when workers.internalRegistryGc.enabled=true" .Values.workers.internalRegistryGc.image.repository -}}
  {{- $_ := required "workers.internalRegistryGc.image.tag is required when workers.internalRegistryGc.enabled=true" .Values.workers.internalRegistryGc.image.tag -}}
  {{- $_ := required "workers.internalRegistryGc.image.pullPolicy is required when workers.internalRegistryGc.enabled=true" .Values.workers.internalRegistryGc.image.pullPolicy -}}
{{- end -}}

{{- if .Values.postgres.enabled -}}
  {{- $storageType := required "postgres.storage.type is required when postgres.enabled=true" .Values.postgres.storage.type -}}
  {{- if and (ne $storageType "emptyDir") (ne $storageType "pvc") (ne $storageType "hostPath") -}}
    {{- fail (printf "invalid postgres.storage.type=%q (allowed: emptyDir, pvc, hostPath)" $storageType) -}}
  {{- end -}}
  {{- if eq $storageType "pvc" -}}
    {{- if not .Values.postgres.persistence.enabled -}}
      {{- fail "postgres.storage.type=pvc requires postgres.persistence.enabled=true" -}}
    {{- end -}}
    {{- $_ := required "postgres.persistence.size is required when postgres.storage.type=pvc" .Values.postgres.persistence.size -}}
  {{- end -}}
  {{- if eq $storageType "hostPath" -}}
    {{- $_ := required "postgres.storage.hostPath.path is required when postgres.storage.type=hostPath" .Values.postgres.storage.hostPath.path -}}
  {{- end -}}
{{- end -}}

{{- if .Values.redis.enabled -}}
  {{- $storageType := required "redis.storage.type is required when redis.enabled=true" .Values.redis.storage.type -}}
  {{- if and (ne $storageType "emptyDir") (ne $storageType "pvc") (ne $storageType "hostPath") -}}
    {{- fail (printf "invalid redis.storage.type=%q (allowed: emptyDir, pvc, hostPath)" $storageType) -}}
  {{- end -}}
  {{- if eq $storageType "pvc" -}}
    {{- if not .Values.redis.persistence.enabled -}}
      {{- fail "redis.storage.type=pvc requires redis.persistence.enabled=true" -}}
    {{- end -}}
    {{- $_ := required "redis.persistence.size is required when redis.storage.type=pvc" .Values.redis.persistence.size -}}
  {{- end -}}
  {{- if eq $storageType "hostPath" -}}
    {{- $_ := required "redis.storage.hostPath.path is required when redis.storage.type=hostPath" .Values.redis.storage.hostPath.path -}}
  {{- end -}}
{{- end -}}

{{- if .Values.nats.enabled -}}
  {{- $storageType := required "nats.storage.type is required when nats.enabled=true" .Values.nats.storage.type -}}
  {{- if and (ne $storageType "emptyDir") (ne $storageType "pvc") (ne $storageType "hostPath") -}}
    {{- fail (printf "invalid nats.storage.type=%q (allowed: emptyDir, pvc, hostPath)" $storageType) -}}
  {{- end -}}
  {{- if eq $storageType "pvc" -}}
    {{- if not .Values.nats.persistence.enabled -}}
      {{- fail "nats.storage.type=pvc requires nats.persistence.enabled=true" -}}
    {{- end -}}
    {{- $_ := required "nats.persistence.size is required when nats.storage.type=pvc" .Values.nats.persistence.size -}}
  {{- end -}}
  {{- if eq $storageType "hostPath" -}}
    {{- $_ := required "nats.storage.hostPath.path is required when nats.storage.type=hostPath" .Values.nats.storage.hostPath.path -}}
  {{- end -}}
{{- end -}}

{{- if .Values.minio.enabled -}}
  {{- $storageType := required "minio.storage.type is required when minio.enabled=true" .Values.minio.storage.type -}}
  {{- if and (ne $storageType "emptyDir") (ne $storageType "pvc") (ne $storageType "hostPath") -}}
    {{- fail (printf "invalid minio.storage.type=%q (allowed: emptyDir, pvc, hostPath)" $storageType) -}}
  {{- end -}}
  {{- if eq $storageType "pvc" -}}
    {{- if not .Values.minio.persistence.enabled -}}
      {{- fail "minio.storage.type=pvc requires minio.persistence.enabled=true" -}}
    {{- end -}}
    {{- $_ := required "minio.persistence.size is required when minio.storage.type=pvc" .Values.minio.persistence.size -}}
  {{- end -}}
  {{- if eq $storageType "hostPath" -}}
    {{- $_ := required "minio.storage.hostPath.path is required when minio.storage.type=hostPath" .Values.minio.storage.hostPath.path -}}
  {{- end -}}
{{- end -}}

{{- if .Values.registry.enabled -}}
  {{- $storageType := required "registry.storage.type is required when registry.enabled=true" .Values.registry.storage.type -}}
  {{- if and (ne $storageType "emptyDir") (ne $storageType "pvc") (ne $storageType "hostPath") -}}
    {{- fail (printf "invalid registry.storage.type=%q (allowed: emptyDir, pvc, hostPath)" $storageType) -}}
  {{- end -}}
  {{- if eq $storageType "pvc" -}}
    {{- if not .Values.registry.persistence.enabled -}}
      {{- fail "registry.storage.type=pvc requires registry.persistence.enabled=true" -}}
    {{- end -}}
    {{- $_ := required "registry.persistence.size is required when registry.storage.type=pvc" .Values.registry.persistence.size -}}
  {{- end -}}
  {{- if eq $storageType "hostPath" -}}
    {{- $_ := required "registry.storage.hostPath.path is required when registry.storage.type=hostPath" .Values.registry.storage.hostPath.path -}}
  {{- end -}}
{{- end -}}

{{- if .Values.ollama.enabled -}}
  {{- $_ := required "ollama.image.repository is required when ollama.enabled=true" .Values.ollama.image.repository -}}
  {{- $_ := required "ollama.image.tag is required when ollama.enabled=true" .Values.ollama.image.tag -}}
  {{- $_ := required "ollama.image.pullPolicy is required when ollama.enabled=true" .Values.ollama.image.pullPolicy -}}
  {{- $storageMode := required "ollama.storage.mode is required when ollama.enabled=true" .Values.ollama.storage.mode -}}
  {{- if and (ne $storageMode "baked") (ne $storageMode "emptyDir") (ne $storageMode "pvc") (ne $storageMode "hostPath") -}}
    {{- fail (printf "invalid ollama.storage.mode=%q (allowed: baked, emptyDir, pvc, hostPath)" $storageMode) -}}
  {{- end -}}
  {{- if eq $storageMode "pvc" -}}
    {{- if not .Values.ollama.persistence.enabled -}}
      {{- fail "ollama.storage.mode=pvc requires ollama.persistence.enabled=true" -}}
    {{- end -}}
    {{- $_ := required "ollama.persistence.size is required when ollama.storage.mode=pvc" .Values.ollama.persistence.size -}}
  {{- end -}}
  {{- if eq $storageMode "hostPath" -}}
    {{- $_ := required "ollama.storage.hostPath.path is required when ollama.storage.mode=hostPath" .Values.ollama.storage.hostPath.path -}}
  {{- end -}}
{{- end -}}

{{- if .Values.loki.enabled -}}
  {{- $_ := required "loki.image.repository is required when loki.enabled=true" .Values.loki.image.repository -}}
  {{- $_ := required "loki.image.tag is required when loki.enabled=true" .Values.loki.image.tag -}}
  {{- $_ := required "loki.image.pullPolicy is required when loki.enabled=true" .Values.loki.image.pullPolicy -}}
  {{- $storageMode := required "loki.storage.mode is required when loki.enabled=true" .Values.loki.storage.mode -}}
  {{- if and (ne $storageMode "emptyDir") (ne $storageMode "pvc") (ne $storageMode "hostPath") -}}
    {{- fail (printf "invalid loki.storage.mode=%q (allowed: emptyDir, pvc, hostPath)" $storageMode) -}}
  {{- end -}}
  {{- if eq $storageMode "pvc" -}}
    {{- if not .Values.loki.persistence.enabled -}}
      {{- fail "loki.storage.mode=pvc requires loki.persistence.enabled=true" -}}
    {{- end -}}
    {{- $_ := required "loki.persistence.size is required when loki.storage.mode=pvc" .Values.loki.persistence.size -}}
  {{- end -}}
  {{- if eq $storageMode "hostPath" -}}
    {{- $_ := required "loki.storage.hostPath.path is required when loki.storage.mode=hostPath" .Values.loki.storage.hostPath.path -}}
  {{- end -}}
{{- end -}}

{{- if .Values.alloy.enabled -}}
  {{- $_ := required "alloy.image.repository is required when alloy.enabled=true" .Values.alloy.image.repository -}}
  {{- $_ := required "alloy.image.tag is required when alloy.enabled=true" .Values.alloy.image.tag -}}
  {{- $_ := required "alloy.image.pullPolicy is required when alloy.enabled=true" .Values.alloy.image.pullPolicy -}}
  {{- $_ := required "alloy.clusterName is required when alloy.enabled=true" .Values.alloy.clusterName -}}
  {{- $pushURL := .Values.alloy.loki.pushUrl -}}
  {{- if and (not .Values.loki.enabled) (eq (trim $pushURL) "") -}}
    {{- fail "alloy.enabled=true requires loki.enabled=true or alloy.loki.pushUrl to be set" -}}
  {{- end -}}
{{- end -}}
{{- end -}}
