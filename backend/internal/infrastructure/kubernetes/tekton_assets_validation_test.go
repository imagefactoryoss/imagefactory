package kubernetes

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"sigs.k8s.io/yaml"
)

type kustomizationFile struct {
	Resources []string `yaml:"resources"`
}

func TestTektonAssets_KustomizationReferencesAssets(t *testing.T) {
	root := tektonAssetsRoot(t)
	kustomizationPath := filepath.Join(root, "kustomization.yaml")

	raw, err := os.ReadFile(kustomizationPath)
	if err != nil {
		t.Fatalf("failed to read %s: %v", kustomizationPath, err)
	}

	var k kustomizationFile
	if err := yaml.Unmarshal(raw, &k); err != nil {
		t.Fatalf("failed to parse kustomization.yaml: %v", err)
	}

	expected := map[string]bool{
		"tasks/v1/git-clone-task.yaml":                 false,
		"tasks/v1/docker-build-task.yaml":              false,
		"tasks/v1/buildx-task.yaml":                    false,
		"tasks/v1/kaniko-task.yaml":                    false,
		"tasks/v1/kaniko-no-push-task.yaml":            false,
		"tasks/v1/packer-task.yaml":                    false,
		"jobs/v1/tekton-history-cleanup-cronjob.yaml":  false,
		"pipelines/v1/image-factory-build-docker.yaml": false,
		"pipelines/v1/image-factory-build-buildx.yaml": false,
		"pipelines/v1/image-factory-build-kaniko.yaml": false,
		"pipelines/v1/image-factory-build-packer.yaml": false,
	}
	for _, resource := range k.Resources {
		if _, ok := expected[resource]; ok {
			expected[resource] = true
		}
	}
	for resource, found := range expected {
		if !found {
			t.Fatalf("kustomization is missing required resource: %s", resource)
		}
	}
}

func TestTektonAssets_TasksAreValidAndNamed(t *testing.T) {
	root := tektonAssetsRoot(t)
	taskDir := filepath.Join(root, "tasks", "v1")
	entries, err := os.ReadDir(taskDir)
	if err != nil {
		t.Fatalf("failed to read task dir %s: %v", taskDir, err)
	}

	expectedNames := map[string]bool{
		"git-clone":      false,
		"docker-build":   false,
		"buildx":         false,
		"kaniko":         false,
		"kaniko-no-push": false,
		"packer":         false,
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		path := filepath.Join(taskDir, entry.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read %s: %v", path, err)
		}

		var task tektonv1beta1.Task
		if err := yaml.Unmarshal(raw, &task); err != nil {
			t.Fatalf("failed to parse task YAML %s: %v", path, err)
		}
		if task.Kind != "Task" {
			t.Fatalf("expected kind Task in %s, got %q", path, task.Kind)
		}
		if task.Name == "" {
			t.Fatalf("task in %s has empty name", path)
		}
		if len(task.Spec.Steps) == 0 {
			t.Fatalf("task %s has no steps", task.Name)
		}
		if _, ok := expectedNames[task.Name]; ok {
			expectedNames[task.Name] = true
		}
	}

	for name, found := range expectedNames {
		if !found {
			t.Fatalf("missing required task asset %q in tasks/v1", name)
		}
	}
}

func TestTektonAssets_GitCloneTaskSupportsAuthWorkspace(t *testing.T) {
	root := tektonAssetsRoot(t)
	taskPath := filepath.Join(root, "tasks", "v1", "git-clone-task.yaml")
	raw, err := os.ReadFile(taskPath)
	if err != nil {
		t.Fatalf("failed to read %s: %v", taskPath, err)
	}

	var task tektonv1beta1.Task
	if err := yaml.Unmarshal(raw, &task); err != nil {
		t.Fatalf("failed to parse task YAML %s: %v", taskPath, err)
	}

	foundOutput := false
	foundAuth := false
	for _, ws := range task.Spec.Workspaces {
		if ws.Name == "output" {
			foundOutput = true
		}
		if ws.Name == "auth" {
			foundAuth = true
			if !ws.Optional {
				t.Fatalf("expected git-clone auth workspace to be optional")
			}
		}
	}
	if !foundOutput {
		t.Fatalf("git-clone task must declare output workspace")
	}
	if !foundAuth {
		t.Fatalf("git-clone task must declare auth workspace")
	}

	if len(task.Spec.Steps) == 0 {
		t.Fatalf("git-clone task has no steps")
	}
	script := task.Spec.Steps[0].Script
	if !strings.Contains(script, "/workspace/auth/ssh-privatekey") {
		t.Fatalf("git-clone script must handle ssh-privatekey from auth workspace")
	}
	if !strings.Contains(script, "/workspace/auth/token") && !strings.Contains(script, "/workspace/auth/password") {
		t.Fatalf("git-clone script must handle token/password from auth workspace")
	}
}

func TestTektonAssets_PipelinesAreValidAndNamed(t *testing.T) {
	root := tektonAssetsRoot(t)
	pipelineDir := filepath.Join(root, "pipelines", "v1")
	entries, err := os.ReadDir(pipelineDir)
	if err != nil {
		t.Fatalf("failed to read pipeline dir %s: %v", pipelineDir, err)
	}

	if len(entries) == 0 {
		t.Fatalf("no pipeline assets found in %s", pipelineDir)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		path := filepath.Join(pipelineDir, entry.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read %s: %v", path, err)
		}

		var p tektonv1beta1.Pipeline
		if err := yaml.Unmarshal(raw, &p); err != nil {
			t.Fatalf("failed to parse pipeline YAML %s: %v", path, err)
		}

		if p.Kind != "Pipeline" {
			t.Fatalf("expected kind Pipeline in %s, got %q", path, p.Kind)
		}
		if !strings.HasPrefix(p.Name, "image-factory-") {
			t.Fatalf("pipeline name in %s must start with image-factory-, got %q", path, p.Name)
		}
		if len(p.Spec.Tasks) == 0 {
			t.Fatalf("pipeline %s has no tasks", p.Name)
		}
		for _, task := range p.Spec.Tasks {
			// Either taskRef or taskSpec is valid in Tekton.
			if task.TaskRef != nil {
				if task.TaskRef.Name == "" {
					t.Fatalf("pipeline %s has task %q with empty taskRef.name", p.Name, task.Name)
				}
				continue
			}
			if task.TaskSpec == nil {
				t.Fatalf("pipeline %s has task %q with neither taskRef nor taskSpec", p.Name, task.Name)
			}
		}
	}
}

func TestTektonAssets_PipelinesWireOptionalGitAuthWorkspace(t *testing.T) {
	root := tektonAssetsRoot(t)
	pipelineDir := filepath.Join(root, "pipelines", "v1")
	entries, err := os.ReadDir(pipelineDir)
	if err != nil {
		t.Fatalf("failed to read pipeline dir %s: %v", pipelineDir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		path := filepath.Join(pipelineDir, entry.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read %s: %v", path, err)
		}

		var p tektonv1beta1.Pipeline
		if err := yaml.Unmarshal(raw, &p); err != nil {
			t.Fatalf("failed to parse pipeline YAML %s: %v", path, err)
		}

		cloneTaskFound := false
		cloneAuthBindingFound := false
		for _, task := range p.Spec.Tasks {
			if task.TaskRef == nil || task.TaskRef.Name != "git-clone" {
				continue
			}
			cloneTaskFound = true
			for _, ws := range task.Workspaces {
				if ws.Name == "auth" && ws.Workspace == "git-auth" {
					cloneAuthBindingFound = true
					break
				}
			}
		}
		if !cloneTaskFound {
			continue
		}

		gitAuthOptional := false
		for _, ws := range p.Spec.Workspaces {
			if ws.Name == "git-auth" {
				gitAuthOptional = ws.Optional
				break
			}
		}
		if !gitAuthOptional {
			t.Fatalf("pipeline %s must declare optional git-auth workspace", p.Name)
		}
		if !cloneAuthBindingFound {
			t.Fatalf("pipeline %s must bind clone auth workspace to git-auth", p.Name)
		}
	}
}

func TestTektonAssets_NoScriptOnShelllessExecutors(t *testing.T) {
	root := tektonAssetsRoot(t)
	taskDir := filepath.Join(root, "tasks", "v1")
	entries, err := os.ReadDir(taskDir)
	if err != nil {
		t.Fatalf("failed to read task dir %s: %v", taskDir, err)
	}

	// images considered shell-less; scripts MUST NOT be used with these images
	shelllessSubstr := []string{"kaniko-project/executor"}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		path := filepath.Join(taskDir, entry.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read %s: %v", path, err)
		}

		var task tektonv1beta1.Task
		if err := yaml.Unmarshal(raw, &task); err != nil {
			t.Fatalf("failed to parse task YAML %s: %v", path, err)
		}

		for _, step := range task.Spec.Steps {
			if step.Script == "" {
				continue
			}
			img := step.Image
			for _, sub := range shelllessSubstr {
				if strings.Contains(img, sub) {
					t.Fatalf("task %q step %q uses shell-less image %q but defines script; use command/args instead", task.Name, step.Name, img)
				}
			}
		}
	}
}

func tektonAssetsRoot(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve current file path")
	}
	// backend/internal/infrastructure/kubernetes -> backend/tekton
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "tekton"))
}
