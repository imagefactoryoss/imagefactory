package build

func (e *MethodTektonExecutor) getDockerPipelineTemplate() string {
	return `
apiVersion: tekton.dev/v1beta1
kind: PipelineRun
metadata:
  generateName: image-factory-docker-run-
  labels:
    build-id: "{{.BuildID}}"
    tenant-id: "{{.TenantID}}"
spec:
  pipelineRef:
    name: image-factory-build-v1-docker
  params:
  - name: git-url
    value: "{{ default \"\" .GitURL }}"
  - name: dockerfile-path
    value: "{{ default \"Dockerfile\" .DockerfilePath }}"
  - name: build-context
    value: "{{ default \".\" .BuildContext }}"
  - name: image-name
    value: "{{ default \"\" .ImageName }}"
  - name: scan-tool
    value: "{{ default \"trivy\" .ScanTool }}"
  - name: sbom-tool
    value: "{{ default \"syft\" .SBOMTool }}"
  - name: enable-scan
    value: "{{ default \"true\" .EnableScan }}"
  - name: enable-sbom
    value: "{{ default \"true\" .EnableSBOM }}"
  - name: enable-sign
    value: "{{ default \"false\" .EnableSign }}"
  - name: sign-key-path
    value: "{{ default \"/workspace/signing-key/cosign.key\" .SignKeyPath }}"
  - name: trivy-cache-mode
    value: "{{ default \"shared\" .TrivyCacheMode }}"
  - name: trivy-db-repository
    value: "{{ default \"image-factory-registry:5000/security/trivy-db:2,mirror.gcr.io/aquasec/trivy-db:2\" .TrivyDBRepository }}"
  - name: trivy-java-db-repository
    value: "{{ default \"image-factory-registry:5000/security/trivy-java-db:1,mirror.gcr.io/aquasec/trivy-java-db:1\" .TrivyJavaDBRepository }}"
  - name: enable-temp-scan-stage
    value: "{{ default \"false\" .EnableTempScanStage }}"
  - name: temp-scan-image-name
    value: "{{ default \"\" .TempScanImageName }}"
  - name: scan-source-image-ref
    value: "{{ default \"\" .ScanSourceImageRef }}"
  - name: sbom-source
    value: "{{ default \"docker-archive:/workspace/source/.image/image.tar\" .SBOMSource }}"
  workspaces:
  - name: source
    volumeClaimTemplate:
      spec:
        accessModes:
        - ReadWriteOnce
        resources:
          requests:
            storage: 1Gi
  - name: dockerconfig
    secret:
      secretName: docker-config
  {{- if .IncludeGitAuth }}
  - name: git-auth
    secret:
      secretName: git-auth
  {{- end }}
  {{- if .IncludeSignKey }}
  - name: signing-key
    secret:
      secretName: "{{ .SignKeySecretName }}"
  {{- end }}
`
}

func (e *MethodTektonExecutor) getBuildxPipelineTemplate() string {
	return `
apiVersion: tekton.dev/v1beta1
kind: PipelineRun
metadata:
  generateName: image-factory-buildx-run-
  labels:
    build-id: "{{.BuildID}}"
spec:
  pipelineRef:
    name: image-factory-build-v1-buildx
  params:
  - name: git-url
    value: "{{ default \"\" .GitURL }}"
  - name: dockerfile-path
    value: "{{ default \"Dockerfile\" .DockerfilePath }}"
  - name: platforms
    value: "{{ default \"linux/amd64\" .Platforms }}"
  - name: build-context
    value: "{{ default \".\" .BuildContext }}"
  - name: image-name
    value: "{{ default \"\" .ImageName }}"
  - name: scan-tool
    value: "{{ default \"trivy\" .ScanTool }}"
  - name: sbom-tool
    value: "{{ default \"syft\" .SBOMTool }}"
  - name: enable-scan
    value: "{{ default \"true\" .EnableScan }}"
  - name: enable-sbom
    value: "{{ default \"true\" .EnableSBOM }}"
  - name: enable-sign
    value: "{{ default \"false\" .EnableSign }}"
  - name: sign-key-path
    value: "{{ default \"/workspace/signing-key/cosign.key\" .SignKeyPath }}"
  - name: trivy-cache-mode
    value: "{{ default \"shared\" .TrivyCacheMode }}"
  - name: trivy-db-repository
    value: "{{ default \"image-factory-registry:5000/security/trivy-db:2,mirror.gcr.io/aquasec/trivy-db:2\" .TrivyDBRepository }}"
  - name: trivy-java-db-repository
    value: "{{ default \"image-factory-registry:5000/security/trivy-java-db:1,mirror.gcr.io/aquasec/trivy-java-db:1\" .TrivyJavaDBRepository }}"
  - name: enable-temp-scan-stage
    value: "{{ default \"false\" .EnableTempScanStage }}"
  - name: temp-scan-image-name
    value: "{{ default \"\" .TempScanImageName }}"
  - name: scan-source-image-ref
    value: "{{ default \"\" .ScanSourceImageRef }}"
  - name: sbom-source
    value: "{{ default \"docker-archive:/workspace/source/.image/image.tar\" .SBOMSource }}"
  workspaces:
  - name: source
    volumeClaimTemplate:
      spec:
        accessModes:
        - ReadWriteOnce
        resources:
          requests:
            storage: 1Gi
  - name: dockerconfig
    secret:
      secretName: docker-config
  {{- if .IncludeGitAuth }}
  - name: git-auth
    secret:
      secretName: git-auth
  {{- end }}
  {{- if .IncludeSignKey }}
  - name: signing-key
    secret:
      secretName: "{{ .SignKeySecretName }}"
  {{- end }}
`
}

func (e *MethodTektonExecutor) getKanikoPipelineTemplate() string {
	return `
apiVersion: tekton.dev/v1beta1
kind: PipelineRun
metadata:
  generateName: image-factory-kaniko-run-
  labels:
    build-id: "{{.BuildID}}"
spec:
  pipelineRef:
    name: image-factory-build-v1-kaniko
  params:
  - name: git-url
    value: "{{ default \"\" .GitURL }}"
  - name: dockerfile-path
    value: "{{ default \"Dockerfile\" .DockerfilePath }}"
  - name: build-context
    value: "{{ default \".\" .BuildContext }}"
  - name: image-name
    value: "{{ default \"\" .ImageName }}"
  - name: scan-tool
    value: "{{ default \"trivy\" .ScanTool }}"
  - name: sbom-tool
    value: "{{ default \"syft\" .SBOMTool }}"
  - name: enable-scan
    value: "{{ default \"true\" .EnableScan }}"
  - name: enable-sbom
    value: "{{ default \"true\" .EnableSBOM }}"
  - name: enable-sign
    value: "{{ default \"false\" .EnableSign }}"
  - name: sign-key-path
    value: "{{ default \"/workspace/signing-key/cosign.key\" .SignKeyPath }}"
  - name: dockerfile-inline-base64
    value: "{{ default \"\" .DockerfileInlineBase64 }}"
  - name: trivy-cache-mode
    value: "{{ default \"shared\" .TrivyCacheMode }}"
  - name: trivy-db-repository
    value: "{{ default \"image-factory-registry:5000/security/trivy-db:2,mirror.gcr.io/aquasec/trivy-db:2\" .TrivyDBRepository }}"
  - name: trivy-java-db-repository
    value: "{{ default \"image-factory-registry:5000/security/trivy-java-db:1,mirror.gcr.io/aquasec/trivy-java-db:1\" .TrivyJavaDBRepository }}"
  - name: enable-temp-scan-stage
    value: "{{ default \"false\" .EnableTempScanStage }}"
  - name: temp-scan-image-name
    value: "{{ default \"\" .TempScanImageName }}"
  - name: scan-source-image-ref
    value: "{{ default \"\" .ScanSourceImageRef }}"
  - name: sbom-source
    value: "{{ default \"docker-archive:/workspace/source/.image/image.tar\" .SBOMSource }}"
  workspaces:
  - name: source
    volumeClaimTemplate:
      spec:
        accessModes:
        - ReadWriteOnce
        resources:
          requests:
            storage: 1Gi
  - name: dockerconfig
    secret:
      secretName: docker-config
  {{- if .IncludeGitAuth }}
  - name: git-auth
    secret:
      secretName: git-auth
  {{- end }}
  {{- if .IncludeSignKey }}
  - name: signing-key
    secret:
      secretName: "{{ .SignKeySecretName }}"
  {{- end }}
`
}

func (e *MethodTektonExecutor) getPackerPipelineTemplate() string {
	return `
apiVersion: tekton.dev/v1beta1
kind: PipelineRun
metadata:
  generateName: image-factory-packer-run-
  labels:
    build-id: "{{.BuildID}}"
spec:
  pipelineRef:
    name: image-factory-build-v1-packer
  params:
  - name: git-url
    value: "{{ default \"\" .GitURL }}"
  - name: packer-template
    value: "{{ default \"\" .PackerTemplate }}"
  - name: vars
    value:
    {{- if .PackerVars }}
    {{- range .PackerVars }}
    - "{{ . }}"
    {{- end }}
    {{- else }}
    - ""
    {{- end }}
  workspaces:
  - name: source
    volumeClaimTemplate:
      spec:
        accessModes:
        - ReadWriteOnce
        resources:
          requests:
            storage: 1Gi
  {{- if .IncludeGitAuth }}
  - name: git-auth
    secret:
      secretName: git-auth
  {{- end }}
`
}
