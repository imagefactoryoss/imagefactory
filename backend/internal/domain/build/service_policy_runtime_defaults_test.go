package build

import (
	"testing"

	systemconfig "github.com/srikarm/image-factory/internal/domain/systemconfig"
)

func TestApplyBuildRuntimeDefaults_SetsTempScanStageWhenMissing(t *testing.T) {
	manifest := &BuildManifest{Metadata: map[string]interface{}{}}
	cfg := &systemconfig.BuildConfig{EnableTempScanStage: true}

	applyBuildRuntimeDefaults(manifest, cfg)

	got, ok := manifest.Metadata["enable_temp_scan_stage"].(bool)
	if !ok {
		t.Fatalf("expected enable_temp_scan_stage bool metadata to be set")
	}
	if !got {
		t.Fatalf("expected enable_temp_scan_stage=true")
	}
}

func TestApplyBuildRuntimeDefaults_DoesNotOverrideExplicitValue(t *testing.T) {
	manifest := &BuildManifest{Metadata: map[string]interface{}{"enable_temp_scan_stage": false}}
	cfg := &systemconfig.BuildConfig{EnableTempScanStage: true}

	applyBuildRuntimeDefaults(manifest, cfg)

	got, ok := manifest.Metadata["enable_temp_scan_stage"].(bool)
	if !ok {
		t.Fatalf("expected enable_temp_scan_stage bool metadata to remain set")
	}
	if got {
		t.Fatalf("expected explicit enable_temp_scan_stage=false to be preserved")
	}
}

func TestApplyBuildRuntimeDefaults_RespectsCamelCaseOverride(t *testing.T) {
	manifest := &BuildManifest{Metadata: map[string]interface{}{"enableTempScanStage": false}}
	cfg := &systemconfig.BuildConfig{EnableTempScanStage: true}

	applyBuildRuntimeDefaults(manifest, cfg)

	if _, exists := manifest.Metadata["enable_temp_scan_stage"]; exists {
		t.Fatalf("expected snake_case key not to be injected when camelCase override exists")
	}
}
