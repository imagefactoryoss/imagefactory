package build

// BuildStatus represents the status of a build.
type BuildStatus string

const (
	BuildStatusPending   BuildStatus = "pending"
	BuildStatusQueued    BuildStatus = "queued"
	BuildStatusRunning   BuildStatus = "running"
	BuildStatusCompleted BuildStatus = "completed"
	BuildStatusFailed    BuildStatus = "failed"
	BuildStatusCancelled BuildStatus = "cancelled"
)

var allowedBuildTransitions = map[BuildStatus]map[BuildStatus]struct{}{
	BuildStatusPending: {
		BuildStatusQueued:    {},
		BuildStatusFailed:    {},
		BuildStatusCancelled: {},
	},
	BuildStatusQueued: {
		BuildStatusRunning:   {},
		BuildStatusFailed:    {},
		BuildStatusCancelled: {},
	},
	BuildStatusRunning: {
		BuildStatusCompleted: {},
		BuildStatusFailed:    {},
		BuildStatusCancelled: {},
	},
	BuildStatusFailed: {
		BuildStatusRunning: {},
	},
	BuildStatusCancelled: {
		BuildStatusRunning: {},
	},
	BuildStatusCompleted: {},
}

// BuildType represents the type of build.
type BuildType string

const (
	BuildTypeContainer BuildType = "container"
	BuildTypeVM        BuildType = "vm"
	BuildTypeCloud     BuildType = "cloud"
	BuildTypePacker    BuildType = "packer"
	BuildTypePaketo    BuildType = "paketo"
	BuildTypeKaniko    BuildType = "kaniko"
	BuildTypeBuildx    BuildType = "buildx"
	BuildTypeNix       BuildType = "nix"
)

// SBOMTool represents the SBOM generation tool.
type SBOMTool string

const (
	SBOMToolSyft  SBOMTool = "syft"
	SBOMToolGrype SBOMTool = "grype"
	SBOMToolTrivy SBOMTool = "trivy"
)

// ScanTool represents the security scanning tool.
type ScanTool string

const (
	ScanToolTrivy ScanTool = "trivy"
	ScanToolClair ScanTool = "clair"
	ScanToolGrype ScanTool = "grype"
	ScanToolSnyk  ScanTool = "snyk"
)

// RegistryType represents the container registry backend.
type RegistryType string

const (
	RegistryTypeS3          RegistryType = "s3"
	RegistryTypeHarbor      RegistryType = "harbor"
	RegistryTypeQuay        RegistryType = "quay"
	RegistryTypeArtifactory RegistryType = "artifactory"
)

// SecretManagerType represents the secret management backend.
type SecretManagerType string

const (
	SecretManagerVault   SecretManagerType = "vault"
	SecretManagerAWSSM   SecretManagerType = "aws_secretsmanager"
	SecretManagerAzureKV SecretManagerType = "azure_keyvault"
	SecretManagerGCP     SecretManagerType = "gcp_secretmanager"
)
