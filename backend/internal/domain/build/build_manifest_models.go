package build

import "github.com/google/uuid"

// BuildManifest represents the build specification.
type BuildManifest struct {
	Name         string                 `json:"name"`
	Type         BuildType              `json:"type"`
	BaseImage    string                 `json:"base_image"`
	Instructions []string               `json:"instructions"`
	Environment  map[string]string      `json:"environment"`
	Tags         []string               `json:"tags"`
	Metadata     map[string]interface{} `json:"metadata"`

	InfrastructureType       string     `json:"infrastructure_type,omitempty"`
	InfrastructureProviderID *uuid.UUID `json:"infrastructure_provider_id,omitempty"`

	VMConfig    *VMBuildConfig `json:"vm_config,omitempty"`
	BuildConfig *BuildConfig   `json:"build_config,omitempty"`
}

// VMBuildConfig represents VM-specific build configuration.
type VMBuildConfig struct {
	PackerTemplate  string                 `json:"packer_template,omitempty"`
	PackerVariables map[string]interface{} `json:"packer_variables,omitempty"`

	CloudProvider    string `json:"cloud_provider,omitempty"`
	Region           string `json:"region,omitempty"`
	AvailabilityZone string `json:"availability_zone,omitempty"`

	InstanceType string `json:"instance_type,omitempty"`
	VMSize       string `json:"vm_size,omitempty"`

	StorageConfig *VMStorageConfig `json:"storage_config,omitempty"`
	NetworkConfig *VMNetworkConfig `json:"network_config,omitempty"`

	OutputFormat      string `json:"output_format,omitempty"`
	OutputName        string `json:"output_name,omitempty"`
	OutputDescription string `json:"output_description,omitempty"`

	Provisioners   []VMProvisioner   `json:"provisioners,omitempty"`
	PostProcessors []VMPostProcessor `json:"post_processors,omitempty"`

	BuildTimeout int `json:"build_timeout,omitempty"`
	MaxRetries   int `json:"max_retries,omitempty"`

	SecurityConfig *VMSecurityConfig `json:"security_config,omitempty"`
}

// VMStorageConfig represents VM storage configuration.
type VMStorageConfig struct {
	RootVolumeSizeGB int            `json:"root_volume_size_gb,omitempty"`
	RootVolumeType   string         `json:"root_volume_type,omitempty"`
	DataVolumes      []VMDataVolume `json:"data_volumes,omitempty"`
	Encrypted        bool           `json:"encrypted,omitempty"`
	KmsKeyID         string         `json:"kms_key_id,omitempty"`
}

// VMDataVolume represents additional data volumes.
type VMDataVolume struct {
	SizeGB     int    `json:"size_gb"`
	VolumeType string `json:"volume_type"`
	MountPoint string `json:"mount_point,omitempty"`
	DeviceName string `json:"device_name,omitempty"`
}

// VMNetworkConfig represents VM network configuration.
type VMNetworkConfig struct {
	VpcID             string   `json:"vpc_id,omitempty"`
	SubnetID          string   `json:"subnet_id,omitempty"`
	SecurityGroups    []string `json:"security_groups,omitempty"`
	AssociatePublicIP bool     `json:"associate_public_ip,omitempty"`
}

// VMProvisioner represents a Packer provisioner.
type VMProvisioner struct {
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config"`
	Only   []string               `json:"only,omitempty"`
	Except []string               `json:"except,omitempty"`
}

// VMPostProcessor represents a Packer post-processor.
type VMPostProcessor struct {
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config"`
	Only   []string               `json:"only,omitempty"`
	Except []string               `json:"except,omitempty"`
}

// VMSecurityConfig represents VM security configuration.
type VMSecurityConfig struct {
	EnableSELinux  bool     `json:"enable_selinux,omitempty"`
	EnableFirewall bool     `json:"enable_firewall,omitempty"`
	AllowedPorts   []string `json:"allowed_ports,omitempty"`
	SSHKeys        []string `json:"ssh_keys,omitempty"`
	Users          []VMUser `json:"users,omitempty"`
}

// VMUser represents a user to be created on the VM.
type VMUser struct {
	Username string   `json:"username"`
	Password string   `json:"password,omitempty"`
	SSHKeys  []string `json:"ssh_keys,omitempty"`
	Groups   []string `json:"groups,omitempty"`
	Sudo     bool     `json:"sudo,omitempty"`
}
