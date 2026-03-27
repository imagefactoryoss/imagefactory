package postgres

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/domain/build"
)

func TestBuildMethodConfigRepository_ConfigRowToMethodConfig_PackerExtendedFields(t *testing.T) {
	buildID := uuid.New()
	metadata, err := json.Marshal(map[string]interface{}{
		"variables": map[string]interface{}{
			"region": "us-east-1",
		},
		"build_vars": map[string]interface{}{
			"image_name": "base-ami",
		},
		"on_error": "abort",
		"parallel": false,
	})
	if err != nil {
		t.Fatalf("failed to marshal metadata: %v", err)
	}

	template := `{"builders":[{"type":"amazon-ebs"}]}`
	row := &buildConfigRow{
		ID:             uuid.New(),
		BuildID:        buildID,
		BuildMethod:    string(build.BuildMethodPacker),
		PackerTemplate: &template,
		Metadata:       metadata,
	}

	repo := &BuildMethodConfigRepository{}
	cfg, err := repo.configRowToMethodConfig(row, build.BuildMethodPacker)
	if err != nil {
		t.Fatalf("expected no error converting row, got %v", err)
	}

	packerCfg, ok := cfg.(*build.PackerConfig)
	if !ok {
		t.Fatalf("expected *build.PackerConfig, got %T", cfg)
	}
	if packerCfg.Template() != template {
		t.Fatalf("expected template to roundtrip")
	}
	if got := packerCfg.BuildVars()["image_name"]; got != "base-ami" {
		t.Fatalf("expected build_vars image_name to roundtrip, got %q", got)
	}
	if got := packerCfg.OnError(); got != "abort" {
		t.Fatalf("expected on_error to roundtrip as abort, got %q", got)
	}
	if packerCfg.Parallel() {
		t.Fatalf("expected parallel false")
	}
}
