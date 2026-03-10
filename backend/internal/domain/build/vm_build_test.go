package build

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVMBuildManifest(t *testing.T) {
	tenantID := uuid.New()
	projectID := uuid.New()

	t.Run("valid VM build manifest", func(t *testing.T) {
		manifest := BuildManifest{
			Name:      "test-vm-build",
			Type:      BuildTypeVM,
			BaseImage: "ubuntu-20.04",
			VMConfig: &VMBuildConfig{
				CloudProvider:  "aws",
				Region:         "us-east-1",
				InstanceType:   "t3.medium",
				OutputFormat:   "ami",
				PackerTemplate: "ubuntu-20.04.json",
				PackerVariables: map[string]interface{}{
					"aws_region":    "us-east-1",
					"instance_type": "t3.medium",
				},
				Provisioners: []VMProvisioner{
					{
						Type: "shell",
						Config: map[string]interface{}{
							"inline": []string{"echo 'VM provisioned'"},
						},
					},
				},
				PostProcessors: []VMPostProcessor{
					{
						Type: "manifest",
						Config: map[string]interface{}{
							"output": "packer-manifest.json",
						},
					},
				},
			},
		}

		build, err := NewBuild(tenantID, projectID, manifest, nil)
		require.NoError(t, err)
		assert.Equal(t, "test-vm-build", build.Manifest().Name)
		assert.Equal(t, BuildTypeVM, build.Manifest().Type)
		assert.NotNil(t, build.Manifest().VMConfig)
		assert.Equal(t, "aws", build.Manifest().VMConfig.CloudProvider)
	})

	t.Run("VM build without VM config should fail", func(t *testing.T) {
		manifest := BuildManifest{
			Name:      "invalid-vm-build",
			Type:      BuildTypeVM,
			BaseImage: "ubuntu-20.04",
			// Missing VMConfig
		}

		_, err := NewBuild(tenantID, projectID, manifest, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "VM configuration is required")
	})

	t.Run("VM build without cloud provider should fail", func(t *testing.T) {
		manifest := BuildManifest{
			Name:      "invalid-vm-build",
			Type:      BuildTypeVM,
			BaseImage: "ubuntu-20.04",
			VMConfig: &VMBuildConfig{
				// Missing CloudProvider
				Region:       "us-east-1",
				InstanceType: "t3.medium",
				OutputFormat: "ami",
			},
		}

		_, err := NewBuild(tenantID, projectID, manifest, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cloud provider is required")
	})

	t.Run("VM build without output format should fail", func(t *testing.T) {
		manifest := BuildManifest{
			Name:      "invalid-vm-build",
			Type:      BuildTypeVM,
			BaseImage: "ubuntu-20.04",
			VMConfig: &VMBuildConfig{
				CloudProvider: "aws",
				Region:        "us-east-1",
				InstanceType:  "t3.medium",
				// Missing OutputFormat
			},
		}

		_, err := NewBuild(tenantID, projectID, manifest, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "output format is required")
	})

	t.Run("VM build without packer template or base image should fail", func(t *testing.T) {
		manifest := BuildManifest{
			Name: "invalid-vm-build",
			Type: BuildTypeVM,
			// Missing both PackerTemplate and BaseImage
			VMConfig: &VMBuildConfig{
				CloudProvider: "aws",
				Region:        "us-east-1",
				InstanceType:  "t3.medium",
				OutputFormat:  "ami",
			},
		}

		_, err := NewBuild(tenantID, projectID, manifest, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "either packer template or base image is required")
	})
}
