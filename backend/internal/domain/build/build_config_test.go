package build

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildConfig_NewEnums(t *testing.T) {
	t.Run("BuildType enum values", func(t *testing.T) {
		assert.Equal(t, BuildType("packer"), BuildTypePacker)
		assert.Equal(t, BuildType("paketo"), BuildTypePaketo)
		assert.Equal(t, BuildType("kaniko"), BuildTypeKaniko)
		assert.Equal(t, BuildType("buildx"), BuildTypeBuildx)
	})

	t.Run("SBOMTool enum values", func(t *testing.T) {
		assert.Equal(t, SBOMTool("syft"), SBOMToolSyft)
		assert.Equal(t, SBOMTool("grype"), SBOMToolGrype)
		assert.Equal(t, SBOMTool("trivy"), SBOMToolTrivy)
	})

	t.Run("ScanTool enum values", func(t *testing.T) {
		assert.Equal(t, ScanTool("trivy"), ScanToolTrivy)
		assert.Equal(t, ScanTool("clair"), ScanToolClair)
		assert.Equal(t, ScanTool("grype"), ScanToolGrype)
		assert.Equal(t, ScanTool("snyk"), ScanToolSnyk)
	})

	t.Run("RegistryType enum values", func(t *testing.T) {
		assert.Equal(t, RegistryType("s3"), RegistryTypeS3)
		assert.Equal(t, RegistryType("harbor"), RegistryTypeHarbor)
		assert.Equal(t, RegistryType("quay"), RegistryTypeQuay)
		assert.Equal(t, RegistryType("artifactory"), RegistryTypeArtifactory)
	})

	t.Run("SecretManagerType enum values", func(t *testing.T) {
		assert.Equal(t, SecretManagerType("vault"), SecretManagerVault)
		assert.Equal(t, SecretManagerType("aws_secretsmanager"), SecretManagerAWSSM)
		assert.Equal(t, SecretManagerType("azure_keyvault"), SecretManagerAzureKV)
		assert.Equal(t, SecretManagerType("gcp_secretmanager"), SecretManagerGCP)
	})
}

func TestBuildConfig_Validation(t *testing.T) {
	tenantID := uuid.New()
	projectID := uuid.New()

	t.Run("valid BuildConfig with all tools", func(t *testing.T) {
		manifest := BuildManifest{
			Name:      "test-multi-tool-build",
			Type:      BuildTypeContainer,
			BaseImage: "ubuntu:20.04",
			Instructions: []string{
				"RUN apt-get update",
				"RUN apt-get install -y curl",
			},
			BuildConfig: &BuildConfig{
				BuildType:         BuildTypeKaniko,
				SBOMTool:          SBOMToolSyft,
				ScanTool:          ScanToolTrivy,
				RegistryType:      RegistryTypeHarbor,
				SecretManagerType: SecretManagerVault,
				Dockerfile:        "Dockerfile",
				BuildContext:      ".",
				RegistryRepo:      "registry.example.com/team/test:latest",
				BuildArgs: map[string]string{
					"VERSION": "1.0.0",
				},
				Variables: map[string]interface{}{
					"env": "production",
				},
			},
		}

		build, err := NewBuild(tenantID, projectID, manifest, nil)
		require.NoError(t, err)
		assert.Equal(t, "test-multi-tool-build", build.Manifest().Name)
		assert.Equal(t, BuildTypeContainer, build.Manifest().Type)
		assert.NotNil(t, build.Manifest().BuildConfig)
		assert.Equal(t, BuildTypeKaniko, build.Manifest().BuildConfig.BuildType)
		assert.Equal(t, SBOMToolSyft, build.Manifest().BuildConfig.SBOMTool)
		assert.Equal(t, ScanToolTrivy, build.Manifest().BuildConfig.ScanTool)
		assert.Equal(t, RegistryTypeHarbor, build.Manifest().BuildConfig.RegistryType)
		assert.Equal(t, SecretManagerVault, build.Manifest().BuildConfig.SecretManagerType)
	})

	t.Run("BuildConfig with invalid BuildType should fail", func(t *testing.T) {
		manifest := BuildManifest{
			Name:         "test-invalid-build-type",
			Type:         BuildTypeContainer,
			BaseImage:    "ubuntu:20.04",
			Instructions: []string{"RUN echo hello"},
			BuildConfig: &BuildConfig{
				BuildType: BuildType("invalid"),
				SBOMTool:  SBOMToolSyft,
			},
		}

		_, err := NewBuild(tenantID, projectID, manifest, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid build type")
	})

	t.Run("BuildConfig with invalid SBOMTool should fail", func(t *testing.T) {
		manifest := BuildManifest{
			Name:         "test-invalid-sbom-tool",
			Type:         BuildTypeContainer,
			BaseImage:    "ubuntu:20.04",
			Instructions: []string{"RUN echo hello"},
			BuildConfig: &BuildConfig{
				BuildType: BuildTypeKaniko,
				SBOMTool:  SBOMTool("invalid"),
			},
		}

		_, err := NewBuild(tenantID, projectID, manifest, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid SBOM tool")
	})

	t.Run("BuildConfig with invalid ScanTool should fail", func(t *testing.T) {
		manifest := BuildManifest{
			Name:         "test-invalid-scan-tool",
			Type:         BuildTypeContainer,
			BaseImage:    "ubuntu:20.04",
			Instructions: []string{"RUN echo hello"},
			BuildConfig: &BuildConfig{
				BuildType: BuildTypeKaniko,
				SBOMTool:  SBOMToolSyft,
				ScanTool:  ScanTool("invalid"),
			},
		}

		_, err := NewBuild(tenantID, projectID, manifest, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid scan tool")
	})

	t.Run("BuildConfig with invalid RegistryType should fail", func(t *testing.T) {
		manifest := BuildManifest{
			Name:         "test-invalid-registry-type",
			Type:         BuildTypeContainer,
			BaseImage:    "ubuntu:20.04",
			Instructions: []string{"RUN echo hello"},
			BuildConfig: &BuildConfig{
				BuildType:    BuildTypeKaniko,
				SBOMTool:     SBOMToolSyft,
				ScanTool:     ScanToolTrivy,
				RegistryType: RegistryType("invalid"),
			},
		}

		_, err := NewBuild(tenantID, projectID, manifest, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid registry type")
	})

	t.Run("BuildConfig with invalid SecretManagerType should fail", func(t *testing.T) {
		manifest := BuildManifest{
			Name:         "test-invalid-secret-manager",
			Type:         BuildTypeContainer,
			BaseImage:    "ubuntu:20.04",
			Instructions: []string{"RUN echo hello"},
			BuildConfig: &BuildConfig{
				BuildType:         BuildTypeKaniko,
				SBOMTool:          SBOMToolSyft,
				ScanTool:          ScanToolTrivy,
				RegistryType:      RegistryTypeHarbor,
				SecretManagerType: SecretManagerType("invalid"),
			},
		}

		_, err := NewBuild(tenantID, projectID, manifest, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid secret manager type")
	})
}

func TestBuildConfig_UnmarshalJSON(t *testing.T) {
	t.Run("snake_case with dockerfile string", func(t *testing.T) {
		payload := []byte(`{
			"build_type":"kaniko",
			"sbom_tool":"syft",
			"scan_tool":"trivy",
			"registry_type":"s3",
			"secret_manager_type":"aws_secretsmanager",
			"dockerfile":"Dockerfile",
			"build_context":"."
		}`)

		var cfg BuildConfig
		err := json.Unmarshal(payload, &cfg)
		require.NoError(t, err)
		assert.Equal(t, BuildTypeKaniko, cfg.BuildType)
		assert.Equal(t, SBOMToolSyft, cfg.SBOMTool)
		assert.Equal(t, ScanToolTrivy, cfg.ScanTool)
		assert.Equal(t, RegistryTypeS3, cfg.RegistryType)
		assert.Equal(t, SecretManagerAWSSM, cfg.SecretManagerType)
		assert.Equal(t, "Dockerfile", cfg.Dockerfile)
		assert.Equal(t, ".", cfg.BuildContext)
	})

	t.Run("snake_case with dockerfile content object", func(t *testing.T) {
		payload := []byte(`{
			"build_type":"kaniko",
			"dockerfile":{"source":"content","content":"FROM alpine"},
			"build_context":"."
		}`)

		var cfg BuildConfig
		err := json.Unmarshal(payload, &cfg)
		require.NoError(t, err)
		assert.Equal(t, BuildTypeKaniko, cfg.BuildType)
		assert.Equal(t, "FROM alpine", cfg.Dockerfile)
		assert.Equal(t, ".", cfg.BuildContext)
	})

	t.Run("camelCase with dockerfile path object", func(t *testing.T) {
		payload := []byte(`{
			"buildType":"kaniko",
			"dockerfile":{"source":"path","path":"Dockerfile.custom"},
			"buildContext":"./src"
		}`)

		var cfg BuildConfig
		err := json.Unmarshal(payload, &cfg)
		require.NoError(t, err)
		assert.Equal(t, BuildTypeKaniko, cfg.BuildType)
		assert.Equal(t, "Dockerfile.custom", cfg.Dockerfile)
		assert.Equal(t, "./src", cfg.BuildContext)
	})

	t.Run("camelCase with dockerfile content object", func(t *testing.T) {
		payload := []byte(`{
			"buildType":"kaniko",
			"dockerfile":{"source":"upload","content":"FROM golang:1.21"},
			"buildContext":"."
		}`)

		var cfg BuildConfig
		err := json.Unmarshal(payload, &cfg)
		require.NoError(t, err)
		assert.Equal(t, BuildTypeKaniko, cfg.BuildType)
		assert.Equal(t, "FROM golang:1.21", cfg.Dockerfile)
		assert.Equal(t, ".", cfg.BuildContext)
	})
}

func TestBuildConfig_PackerSpecific(t *testing.T) {
	tenantID := uuid.New()
	projectID := uuid.New()

	t.Run("valid Packer BuildConfig", func(t *testing.T) {
		manifest := BuildManifest{
			Name:      "test-packer-build",
			Type:      BuildTypePacker,
			BaseImage: "ubuntu-20.04",
			BuildConfig: &BuildConfig{
				BuildType:         BuildTypePacker,
				SBOMTool:          SBOMToolSyft,
				ScanTool:          ScanToolTrivy,
				RegistryType:      RegistryTypeS3,
				SecretManagerType: SecretManagerVault,
				PackerTemplate:    `{"builders": [{"type": "amazon-ebs"}]}`,
				Variables: map[string]interface{}{
					"aws_region": "us-east-1",
				},
				Builders: []PackerBuilder{
					{
						Type: "amazon-ebs",
						Config: map[string]interface{}{
							"region": "{{user `aws_region`}}",
						},
					},
				},
			},
		}

		build, err := NewBuild(tenantID, projectID, manifest, nil)
		require.NoError(t, err)
		assert.Equal(t, BuildTypePacker, build.Manifest().BuildConfig.BuildType)
		assert.NotEmpty(t, build.Manifest().BuildConfig.PackerTemplate)
		assert.Len(t, build.Manifest().BuildConfig.Builders, 1)
	})
}

func TestBuildConfig_PaketoSpecific(t *testing.T) {
	tenantID := uuid.New()
	projectID := uuid.New()

	t.Run("valid Paketo BuildConfig", func(t *testing.T) {
		manifest := BuildManifest{
			Name:      "test-paketo-build",
			Type:      BuildTypePaketo,
			BaseImage: "paketobuildpacks/builder:base",
			BuildConfig: &BuildConfig{
				BuildType:         BuildTypePaketo,
				SBOMTool:          SBOMToolSyft,
				ScanTool:          ScanToolTrivy,
				RegistryType:      RegistryTypeHarbor,
				SecretManagerType: SecretManagerVault,
				PaketoConfig: &PaketoConfig{
					Builder: "paketobuildpacks/builder:base",
					Buildpacks: []string{
						"paketobuildpacks/java",
					},
					Env: map[string]string{
						"BP_JVM_VERSION": "17",
					},
				},
			},
		}

		build, err := NewBuild(tenantID, projectID, manifest, nil)
		require.NoError(t, err)
		assert.Equal(t, BuildTypePaketo, build.Manifest().BuildConfig.BuildType)
		assert.NotNil(t, build.Manifest().BuildConfig.PaketoConfig)
		assert.Equal(t, "paketobuildpacks/builder:base", build.Manifest().BuildConfig.PaketoConfig.Builder)
	})
}

func TestBuildConfig_KanikoSpecific(t *testing.T) {
	tenantID := uuid.New()
	projectID := uuid.New()

	t.Run("valid Kaniko BuildConfig", func(t *testing.T) {
		manifest := BuildManifest{
			Name:      "test-kaniko-build",
			Type:      BuildTypeKaniko,
			BaseImage: "ubuntu:20.04",
			BuildConfig: &BuildConfig{
				BuildType:         BuildTypeKaniko,
				SBOMTool:          SBOMToolSyft,
				ScanTool:          ScanToolTrivy,
				RegistryType:      RegistryTypeHarbor,
				SecretManagerType: SecretManagerVault,
				Dockerfile:        "Dockerfile",
				BuildContext:      ".",
				BuildArgs: map[string]string{
					"VERSION": "1.0.0",
				},
				Target:       "production",
				Cache:        true,
				CacheRepo:    "my-registry/cache",
				RegistryRepo: "123456789012.dkr.ecr.us-east-1.amazonaws.com/my-app",
			},
		}

		build, err := NewBuild(tenantID, projectID, manifest, nil)
		require.NoError(t, err)
		assert.Equal(t, BuildTypeKaniko, build.Manifest().BuildConfig.BuildType)
		assert.Equal(t, "Dockerfile", build.Manifest().BuildConfig.Dockerfile)
		assert.Equal(t, ".", build.Manifest().BuildConfig.BuildContext)
		assert.True(t, build.Manifest().BuildConfig.Cache)
	})

	t.Run("kaniko BuildConfig without registry repo should fail", func(t *testing.T) {
		manifest := BuildManifest{
			Name:      "test-kaniko-build-missing-registry",
			Type:      BuildTypeKaniko,
			BaseImage: "ubuntu:20.04",
			BuildConfig: &BuildConfig{
				BuildType:         BuildTypeKaniko,
				SBOMTool:          SBOMToolSyft,
				ScanTool:          ScanToolTrivy,
				RegistryType:      RegistryTypeHarbor,
				SecretManagerType: SecretManagerVault,
				Dockerfile:        "Dockerfile",
				BuildContext:      ".",
			},
		}

		_, err := NewBuild(tenantID, projectID, manifest, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "registry_repo is required for kaniko builds")
	})
}

func TestBuildConfig_BuildxSpecific(t *testing.T) {
	tenantID := uuid.New()
	projectID := uuid.New()

	t.Run("valid Buildx BuildConfig", func(t *testing.T) {
		manifest := BuildManifest{
			Name:      "test-buildx-build",
			Type:      BuildTypeBuildx,
			BaseImage: "ubuntu:20.04",
			BuildConfig: &BuildConfig{
				BuildType:         BuildTypeBuildx,
				SBOMTool:          SBOMToolSyft,
				ScanTool:          ScanToolTrivy,
				RegistryType:      RegistryTypeHarbor,
				SecretManagerType: SecretManagerVault,
				Dockerfile:        "Dockerfile",
				BuildContext:      ".",
				BuildArgs: map[string]string{
					"VERSION": "1.0.0",
				},
				RegistryRepo: "registry.example.com/team/buildx-app:latest",
				Target:    "production",
				Platforms: []string{"linux/amd64", "linux/arm64"},
				Cache:     true,
				CacheTo:   "type=registry,ref=my-registry/cache",
				CacheFrom: []string{"my-registry/cache"},
				Secrets: map[string]string{
					"git_token": "GIT_TOKEN",
				},
			},
		}

		build, err := NewBuild(tenantID, projectID, manifest, nil)
		require.NoError(t, err)
		assert.Equal(t, BuildTypeBuildx, build.Manifest().BuildConfig.BuildType)
		assert.Contains(t, build.Manifest().BuildConfig.Platforms, "linux/amd64")
		assert.Contains(t, build.Manifest().BuildConfig.Platforms, "linux/arm64")
		assert.True(t, build.Manifest().BuildConfig.Cache)
		assert.Contains(t, build.Manifest().BuildConfig.Secrets, "git_token")
	})

	t.Run("buildx BuildConfig without registry repo should fail", func(t *testing.T) {
		manifest := BuildManifest{
			Name:      "test-buildx-no-registry",
			Type:      BuildTypeBuildx,
			BaseImage: "ubuntu:20.04",
			BuildConfig: &BuildConfig{
				BuildType:         BuildTypeBuildx,
				SBOMTool:          SBOMToolSyft,
				ScanTool:          ScanToolTrivy,
				RegistryType:      RegistryTypeHarbor,
				SecretManagerType: SecretManagerVault,
				Dockerfile:        "Dockerfile",
				BuildContext:      ".",
				Platforms:         []string{"linux/amd64"},
			},
		}

		_, err := NewBuild(tenantID, projectID, manifest, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "registry_repo is required for buildx builds")
	})
}
