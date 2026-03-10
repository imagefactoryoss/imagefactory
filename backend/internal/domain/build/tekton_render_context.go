package build

import (
	"encoding/base64"
	"fmt"
	"os"
	"sort"
	"strings"
)

// TektonRenderContext is the typed data passed to PipelineRun templates.
type TektonRenderContext struct {
	BuildID                string
	TenantID               string
	ProviderID             string
	GitURL                 string
	BuildContext           string
	DockerfilePath         string
	DockerfileInlineBase64 string
	ImageName              string
	ScanTool               string
	SBOMTool               string
	EnableScan             string
	EnableSBOM             string
	EnableSign             string
	SignKeyPath            string
	SignKeySecretName      string
	Platforms              string
	PackerTemplate         string
	PackerVars             []string
	IncludeGitAuth         bool
	IncludeSignKey         bool
	TrivyCacheMode         string
	TrivyDBRepository      string
	TrivyJavaDBRepository  string
	EnableTempScanStage    string
	TempScanImageName      string
	ScanSourceImageRef     string
	SBOMSource             string
}

func newTektonRenderContext(build *Build, cfg BuildMethodConfig, method BuildMethod) TektonRenderContext {
	ctx := TektonRenderContext{
		BuildContext:          ".",
		Platforms:             "linux/amd64",
		PackerVars:            []string{},
		EnableScan:            "true",
		EnableSBOM:            "true",
		EnableSign:            "false",
		SignKeyPath:           "/workspace/signing-key/cosign.key",
		EnableTempScanStage:   boolString(defaultTempScanStageEnabled()),
		SBOMSource:            "docker-archive:/workspace/source/.image/image.tar",
		TrivyCacheMode:        "shared",
		TrivyDBRepository:     "image-factory-registry:5000/security/trivy-db:2,mirror.gcr.io/aquasec/trivy-db:2",
		TrivyJavaDBRepository: "image-factory-registry:5000/security/trivy-java-db:1,mirror.gcr.io/aquasec/trivy-java-db:1",
	}
	if build != nil {
		ctx.BuildID = build.ID().String()
		ctx.TenantID = build.TenantID().String()
		if providerID := build.InfrastructureProviderID(); providerID != nil {
			ctx.ProviderID = providerID.String()
		}
		ctx.GitURL = extractGitURL(build.Manifest())
		if build.Manifest().BuildConfig != nil && build.Manifest().BuildConfig.RegistryRepo != "" {
			ctx.ImageName = build.Manifest().BuildConfig.RegistryRepo
			ctx.ScanTool = string(build.Manifest().BuildConfig.ScanTool)
			ctx.SBOMTool = string(build.Manifest().BuildConfig.SBOMTool)
		}
		ctx.EnableScan = boolString(extractOptionalBool(build.Manifest(), true, "enable_scan", "enableScan", "scan_enabled", "scanEnabled"))
		ctx.EnableSBOM = boolString(extractOptionalBool(build.Manifest(), true, "enable_sbom", "enableSbom", "sbom_enabled", "sbomEnabled"))
		ctx.EnableSign = boolString(extractOptionalBool(build.Manifest(), false, "enable_sign", "enableSign", "sign_enabled", "signEnabled"))
		ctx.EnableTempScanStage = boolString(extractOptionalBool(build.Manifest(), defaultTempScanStageEnabled(), "enable_temp_scan_stage", "enableTempScanStage"))
		ctx.TempScanImageName = extractStringMetadata(build.Manifest(), "temp_scan_image_name", "tempScanImageName")
		if secretName := extractStringMetadata(build.Manifest(), "sign_key_secret", "signKeySecret", "cosign_key_secret", "cosignKeySecret"); secretName != "" {
			ctx.SignKeySecretName = secretName
			ctx.IncludeSignKey = true
		}
		if keyPath := extractStringMetadata(build.Manifest(), "sign_key_path", "signKeyPath", "cosign_key_path", "cosignKeyPath"); keyPath != "" {
			ctx.SignKeyPath = keyPath
		}
		if cacheMode := extractStringMetadata(build.Manifest(), "trivy_cache_mode", "trivyCacheMode"); cacheMode != "" {
			ctx.TrivyCacheMode = cacheMode
		}
		if dbRepo := extractStringMetadata(build.Manifest(), "trivy_db_repository", "trivyDbRepository"); dbRepo != "" {
			ctx.TrivyDBRepository = dbRepo
		}
		if javaRepo := extractStringMetadata(build.Manifest(), "trivy_java_db_repository", "trivyJavaDbRepository"); javaRepo != "" {
			ctx.TrivyJavaDBRepository = javaRepo
		}
		if strings.TrimSpace(ctx.ImageName) == "" {
			ctx.ImageName = extractStringMetadata(build.Manifest(), "registry_repo", "registryRepo", "image_name", "imageName", "image_ref", "imageRef")
		}
	}

	switch c := cfg.(type) {
	case *KanikoConfig:
		rawDockerfile := c.Dockerfile()
		rawBuildContext := strings.TrimSpace(c.BuildContext())
		if rawBuildContext != "" {
			ctx.BuildContext = rawBuildContext
		}
		manifest := BuildManifest{}
		if build != nil {
			manifest = build.Manifest()
		}
		inlineDockerfile := extractDockerfileInline(manifest)
		if inlineDockerfile == "" && isLikelyDockerfileContent(rawDockerfile) {
			inlineDockerfile = rawDockerfile
		}
		if inlineDockerfile != "" {
			path := strings.TrimSpace(extractDockerfilePath(manifest))
			if path == "" && !isLikelyDockerfileContent(rawDockerfile) {
				path = strings.TrimSpace(rawDockerfile)
			}
			if path == "" {
				path = "Dockerfile"
			}
			ctx.DockerfilePath = path
			ctx.DockerfileInlineBase64 = base64.StdEncoding.EncodeToString([]byte(inlineDockerfile))
		} else {
			ctx.DockerfilePath = rawDockerfile
		}
		ctx.ImageName = c.RegistryRepo()
	case *BuildxConfig:
		ctx.DockerfilePath = c.Dockerfile()
		rawBuildContext := strings.TrimSpace(c.BuildContext())
		if rawBuildContext != "" {
			ctx.BuildContext = rawBuildContext
		}
		platforms := normalizeBuildxPlatforms(c.Platforms())
		if len(platforms) > 0 {
			ctx.Platforms = strings.Join(platforms, ",")
		}
	case *PackerConfig:
		ctx.PackerTemplate = c.Template()
		ctx.PackerVars = flattenPackerVars(c.Variables())
	}

	if ctx.DockerfilePath == "" {
		ctx.DockerfilePath = "Dockerfile"
	}
	if strings.TrimSpace(ctx.BuildContext) == "" {
		ctx.BuildContext = "."
	}
	if strings.TrimSpace(ctx.ScanTool) == "" {
		ctx.ScanTool = "trivy"
	}
	if strings.TrimSpace(ctx.SBOMTool) == "" {
		ctx.SBOMTool = "syft"
	}
	if strings.EqualFold(strings.TrimSpace(ctx.EnableTempScanStage), "true") {
		if strings.TrimSpace(ctx.TempScanImageName) == "" {
			ctx.TempScanImageName = deriveDefaultTempScanImageName(ctx)
		}
		if strings.TrimSpace(ctx.TempScanImageName) != "" {
			ctx.ScanSourceImageRef = ctx.TempScanImageName
		}
	}
	if strings.TrimSpace(ctx.SBOMSource) == "" {
		ctx.SBOMSource = "docker-archive:/workspace/source/.image/image.tar"
	}

	return ctx
}

func defaultTempScanStageEnabled() bool {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv("IF_ENABLE_TEMP_SCAN_STAGE")))
	switch raw {
	case "0", "false", "no", "n", "off":
		return false
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return true
	}
}

func deriveDefaultTempScanImageName(ctx TektonRenderContext) string {
	registry := strings.TrimSpace(os.Getenv("IF_INTERNAL_TEMP_SCAN_REGISTRY"))
	if registry == "" {
		hosts := internalRegistryHosts()
		if len(hosts) == 0 {
			return ""
		}
		registry = strings.TrimSpace(hosts[0])
	}
	if registry == "" {
		return ""
	}
	repository := strings.TrimSpace(os.Getenv("IF_INTERNAL_TEMP_SCAN_REPOSITORY"))
	if repository == "" {
		repository = "quarantine-temp"
	}
	tenant := compactToken(ctx.TenantID)
	build := compactToken(ctx.BuildID)
	if tenant == "" {
		tenant = "tenant"
	}
	if build == "" {
		build = "build"
	}
	return fmt.Sprintf("%s/%s/%s/%s:scan", registry, repository, tenant, build)
}

func compactToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, "-", "")
	if len(value) > 12 {
		return value[:12]
	}
	return value
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func extractOptionalBool(manifest BuildManifest, fallback bool, keys ...string) bool {
	if manifest.Metadata == nil {
		return fallback
	}
	for _, key := range keys {
		raw, ok := manifest.Metadata[key]
		if !ok {
			continue
		}
		switch typed := raw.(type) {
		case bool:
			return typed
		case string:
			normalized := strings.ToLower(strings.TrimSpace(typed))
			switch normalized {
			case "true", "1", "yes", "y", "on":
				return true
			case "false", "0", "no", "n", "off":
				return false
			}
		case float64:
			return typed != 0
		case float32:
			return typed != 0
		case int:
			return typed != 0
		case int64:
			return typed != 0
		}
	}
	return fallback
}

func extractStringMetadata(manifest BuildManifest, keys ...string) string {
	if manifest.Metadata == nil {
		return ""
	}
	for _, key := range keys {
		if raw, ok := manifest.Metadata[key]; ok {
			if typed, ok := raw.(string); ok && strings.TrimSpace(typed) != "" {
				return strings.TrimSpace(typed)
			}
		}
	}
	return ""
}

func validateTektonRenderContext(ctx TektonRenderContext, method BuildMethod) error {
	if strings.TrimSpace(ctx.GitURL) == "" {
		return fmt.Errorf("git repository URL is required for tekton %s builds", method)
	}

	switch method {
	case BuildMethodDocker, BuildMethodBuildx, BuildMethodKaniko:
		if strings.TrimSpace(ctx.ImageName) == "" {
			return fmt.Errorf("image reference is required for tekton %s builds", method)
		}
		if strings.TrimSpace(ctx.DockerfileInlineBase64) == "" && strings.ContainsAny(ctx.DockerfilePath, "\n\r") {
			return fmt.Errorf("dockerfile for tekton %s builds must be a repository path (for example: Dockerfile), not inline file content", method)
		}
	case BuildMethodPacker:
		if strings.TrimSpace(ctx.PackerTemplate) == "" {
			return fmt.Errorf("packer template is required for tekton %s builds", method)
		}
	}

	return nil
}

func extractDockerfileInline(manifest BuildManifest) string {
	if manifest.Metadata == nil {
		return ""
	}
	keys := []string{"dockerfile_inline", "dockerfileInline", "dockerfile_content", "dockerfileContent"}
	for _, key := range keys {
		if val, ok := manifest.Metadata[key].(string); ok && strings.TrimSpace(val) != "" {
			return val
		}
	}
	return ""
}

func extractDockerfilePath(manifest BuildManifest) string {
	if manifest.Metadata == nil {
		return ""
	}
	keys := []string{"dockerfile_path", "dockerfilePath"}
	for _, key := range keys {
		if val, ok := manifest.Metadata[key].(string); ok && strings.TrimSpace(val) != "" {
			return val
		}
	}
	return ""
}

func isLikelyDockerfileContent(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}
	if strings.ContainsAny(trimmed, "\n\r") {
		return true
	}

	firstLine := strings.ToUpper(strings.TrimSpace(strings.Split(trimmed, "\n")[0]))
	directives := []string{
		"FROM ", "ARG ", "RUN ", "COPY ", "ADD ", "WORKDIR ", "CMD ", "ENTRYPOINT ",
		"ENV ", "EXPOSE ", "USER ", "LABEL ", "SHELL ", "STOPSIGNAL ", "HEALTHCHECK ",
		"ONBUILD ", "VOLUME ", "MAINTAINER ",
	}
	for _, directive := range directives {
		if strings.HasPrefix(firstLine, directive) {
			return true
		}
	}
	return false
}

func extractGitURL(manifest BuildManifest) string {
	if manifest.Metadata == nil {
		return ""
	}
	keys := []string{"git_url", "gitUrl", "repo_url", "repoUrl", "repository_url", "repositoryUrl"}
	for _, key := range keys {
		if val, ok := manifest.Metadata[key].(string); ok && strings.TrimSpace(val) != "" {
			return val
		}
	}
	return ""
}

func flattenPackerVars(vars map[string]interface{}) []string {
	if len(vars) == 0 {
		return []string{}
	}
	keys := make([]string, 0, len(vars))
	for key := range vars {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := make([]string, 0, len(keys))
	for _, key := range keys {
		result = append(result, fmt.Sprintf("%s=%v", key, vars[key]))
	}
	return result
}
