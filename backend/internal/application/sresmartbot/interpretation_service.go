package sresmartbot

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/domain/systemconfig"
	"github.com/srikarm/image-factory/internal/infrastructure/llm"
)

type AgentInterpretationResponse struct {
	Draft                *AgentDraftResponse `json:"draft"`
	Provider             string              `json:"provider"`
	Model                string              `json:"model"`
	Generated            bool                `json:"generated"`
	OperatorSummary      string              `json:"operator_summary,omitempty"`
	LikelyRootCause      string              `json:"likely_root_cause,omitempty"`
	Watchouts            []string            `json:"watchouts,omitempty"`
	OperatorMessageDraft string              `json:"operator_message_draft,omitempty"`
	RawResponse          string              `json:"raw_response,omitempty"`
}

type interpretationJSON struct {
	OperatorSummary      string   `json:"operator_summary"`
	LikelyRootCause      string   `json:"likely_root_cause"`
	Watchouts            []string `json:"watchouts"`
	OperatorMessageDraft string   `json:"operator_message_draft"`
}

type InterpretationService struct {
	agentService     *AgentService
	workspaceService *WorkspaceService
}

func NewInterpretationService(agentService *AgentService, workspaceService *WorkspaceService) *InterpretationService {
	return &InterpretationService{
		agentService:     agentService,
		workspaceService: workspaceService,
	}
}

func (s *InterpretationService) BuildInterpretation(ctx context.Context, tenantID *uuid.UUID, incidentID uuid.UUID) (*AgentInterpretationResponse, error) {
	if s == nil || s.agentService == nil || s.workspaceService == nil {
		return nil, fmt.Errorf("interpretation service is not configured")
	}
	draft, err := s.agentService.BuildDraft(ctx, tenantID, incidentID)
	if err != nil {
		return nil, err
	}
	workspace, err := s.workspaceService.BuildIncidentWorkspace(ctx, tenantID, incidentID)
	if err != nil {
		return nil, err
	}

	resp := &AgentInterpretationResponse{
		Draft:    draft,
		Provider: workspace.AgentRuntime.Provider,
		Model:    workspace.AgentRuntime.Model,
	}
	if !workspace.AgentRuntime.Enabled || strings.TrimSpace(workspace.AgentRuntime.Provider) == "" || workspace.AgentRuntime.Provider == "none" || workspace.AgentRuntime.Provider == "custom" {
		return resp, nil
	}
	if workspace.AgentRuntime.Provider != "ollama" {
		return resp, nil
	}

	client := llm.NewOllamaClient(workspace.AgentRuntime.BaseURL, nil)
	raw, err := client.Generate(ctx, workspace.AgentRuntime.Model, buildInterpretationPrompt(draft, workspace.AgentRuntime))
	if err != nil {
		return nil, err
	}
	resp.Generated = true
	resp.RawResponse = raw

	var parsed interpretationJSON
	if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
		resp.OperatorSummary = strings.TrimSpace(parsed.OperatorSummary)
		resp.LikelyRootCause = strings.TrimSpace(parsed.LikelyRootCause)
		resp.Watchouts = parsed.Watchouts
		resp.OperatorMessageDraft = strings.TrimSpace(parsed.OperatorMessageDraft)
		return resp, nil
	}

	resp.OperatorSummary = raw
	return resp, nil
}

func buildInterpretationPrompt(draft *AgentDraftResponse, runtime systemconfig.RobotSREAgentRuntimeConfig) string {
	payload, _ := json.MarshalIndent(draft, "", "  ")
	return strings.TrimSpace(fmt.Sprintf(`
You are SRE Smart Bot's local interpretation layer.
You must stay grounded in the deterministic draft and never invent facts.
Return only valid JSON with this shape:
{
  "operator_summary": "short paragraph",
  "likely_root_cause": "single concise statement",
  "watchouts": ["item 1", "item 2"],
  "operator_message_draft": "short operator-facing update"
}

System prompt ref: %s

Deterministic draft:
%s
`, runtime.SystemPromptRef, string(payload)))
}
