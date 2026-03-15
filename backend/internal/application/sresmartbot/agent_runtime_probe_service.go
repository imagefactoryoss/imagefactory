package sresmartbot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/srikarm/image-factory/internal/domain/systemconfig"
	"github.com/srikarm/image-factory/internal/infrastructure/llm"
)

type AgentRuntimeProbeResponse struct {
	Provider        string   `json:"provider"`
	Model           string   `json:"model"`
	BaseURL         string   `json:"base_url,omitempty"`
	Healthy         bool     `json:"healthy"`
	Status          string   `json:"status"`
	Message         string   `json:"message"`
	LatencyMS       int64    `json:"latency_ms,omitempty"`
	SampleResponse  string   `json:"sample_response,omitempty"`
	ModelInstalled  bool     `json:"model_installed"`
	InstalledModels []string `json:"installed_models,omitempty"`
	Guidance        []string `json:"guidance,omitempty"`
}

type AgentRuntimeProbeService struct{}

func NewAgentRuntimeProbeService() *AgentRuntimeProbeService {
	return &AgentRuntimeProbeService{}
}

func (s *AgentRuntimeProbeService) Probe(ctx context.Context, runtime systemconfig.RobotSREAgentRuntimeConfig) (*AgentRuntimeProbeResponse, error) {
	resp := &AgentRuntimeProbeResponse{
		Provider: strings.TrimSpace(runtime.Provider),
		Model:    strings.TrimSpace(runtime.Model),
		BaseURL:  strings.TrimSpace(runtime.BaseURL),
		Status:   "skipped",
		Message:  "Agent runtime probe was skipped.",
	}

	if !runtime.Enabled {
		resp.Message = "Agent runtime is disabled. Enable it to test the configured model path."
		return resp, nil
	}

	if resp.Provider == "" || resp.Provider == "none" || resp.Provider == "custom" {
		resp.Message = "Only concrete model runtimes can be tested. Configure provider=ollama to run a live model probe."
		return resp, nil
	}

	if resp.Provider != "ollama" {
		resp.Message = fmt.Sprintf("Provider %q is not probe-enabled yet. Only Ollama is currently supported for live checks.", resp.Provider)
		return resp, nil
	}

	if resp.Model == "" {
		return nil, fmt.Errorf("agent_runtime.model is required when probing ollama")
	}
	if resp.BaseURL == "" {
		return nil, fmt.Errorf("agent_runtime.base_url is required when probing ollama")
	}

	client := llm.NewOllamaClient(resp.BaseURL, nil)
	models, listErr := client.ListModels(ctx)
	if listErr == nil {
		resp.InstalledModels = models
		for _, model := range models {
			if strings.EqualFold(strings.TrimSpace(model), resp.Model) {
				resp.ModelInstalled = true
				break
			}
		}
	} else {
		resp.Guidance = append(resp.Guidance, "Could not list local Ollama models; if the daemon is reachable but model inventory is blocked, verify the local runtime and permissions.")
	}
	if !resp.ModelInstalled {
		resp.Guidance = append(resp.Guidance,
			fmt.Sprintf("Install the configured model locally before enabling interpretation, for example: ollama pull %s", resp.Model),
			"In air-gapped environments, pre-seed the Ollama model blobs from an internal artifact source or bake them into the image/host before deployment.",
		)
	}
	started := time.Now()
	raw, err := client.Generate(ctx, resp.Model, strings.TrimSpace(`
Return one short sentence confirming that SRE Smart Bot can reach this local model.
Keep the response under 20 words and do not use markdown.
`))
	resp.LatencyMS = time.Since(started).Milliseconds()
	if err != nil {
		resp.Status = "error"
		resp.Message = fmt.Sprintf("Local model probe failed: %v", err)
		return resp, nil
	}

	resp.Healthy = true
	resp.Status = "healthy"
	resp.Message = "Local model responded successfully."
	resp.SampleResponse = strings.TrimSpace(raw)
	return resp, nil
}
