package rest

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	appsresmartbot "github.com/srikarm/image-factory/internal/application/sresmartbot"
	"github.com/srikarm/image-factory/internal/domain/sresmartbot"
	"github.com/srikarm/image-factory/internal/domain/systemconfig"
	"github.com/srikarm/image-factory/internal/infrastructure/middleware"
	"go.uber.org/zap"
)

type SRESmartBotHandler struct {
	repo                      sresmartbot.Repository
	actionService             *appsresmartbot.ActionService
	demoService               *appsresmartbot.DemoService
	detectorSuggestionService *appsresmartbot.DetectorRuleSuggestionService
	workspaceService          *appsresmartbot.WorkspaceService
	mcpService                *appsresmartbot.MCPService
	agentService              *appsresmartbot.AgentService
	probeService              *appsresmartbot.AgentRuntimeProbeService
	interpretService          *appsresmartbot.InterpretationService
	logger                    *zap.Logger
}

func NewSRESmartBotHandler(
	repo sresmartbot.Repository,
	actionService *appsresmartbot.ActionService,
	demoService *appsresmartbot.DemoService,
	detectorSuggestionService *appsresmartbot.DetectorRuleSuggestionService,
	workspaceService *appsresmartbot.WorkspaceService,
	mcpService *appsresmartbot.MCPService,
	agentService *appsresmartbot.AgentService,
	probeService *appsresmartbot.AgentRuntimeProbeService,
	interpretService *appsresmartbot.InterpretationService,
	logger *zap.Logger,
) *SRESmartBotHandler {
	return &SRESmartBotHandler{
		repo:                      repo,
		actionService:             actionService,
		demoService:               demoService,
		detectorSuggestionService: detectorSuggestionService,
		workspaceService:          workspaceService,
		mcpService:                mcpService,
		agentService:              agentService,
		probeService:              probeService,
		interpretService:          interpretService,
		logger:                    logger,
	}
}

type listSREIncidentsResponse struct {
	Incidents []*sresmartbot.Incident `json:"incidents"`
	Total     int                     `json:"total"`
	Limit     int                     `json:"limit"`
	Offset    int                     `json:"offset"`
}

type sreApprovalQueueItem struct {
	Approval *sresmartbot.Approval      `json:"approval"`
	Incident *sresmartbot.Incident      `json:"incident,omitempty"`
	Action   *sresmartbot.ActionAttempt `json:"action,omitempty"`
}

type listSREApprovalsResponse struct {
	Approvals []*sreApprovalQueueItem `json:"approvals"`
	Total     int                     `json:"total"`
	Limit     int                     `json:"limit"`
	Offset    int                     `json:"offset"`
}

type listSREDetectorRuleSuggestionsResponse struct {
	Suggestions []*sresmartbot.DetectorRuleSuggestion `json:"suggestions"`
	Limit       int                                   `json:"limit"`
	Offset      int                                   `json:"offset"`
}

type getSREIncidentResponse struct {
	Incident       *sresmartbot.Incident        `json:"incident"`
	Findings       []*sresmartbot.Finding       `json:"findings"`
	Evidence       []*sresmartbot.Evidence      `json:"evidence"`
	ActionAttempts []*sresmartbot.ActionAttempt `json:"action_attempts"`
	Approvals      []*sresmartbot.Approval      `json:"approvals"`
}

type getSREIncidentWorkspaceResponse = appsresmartbot.IncidentWorkspace

type listSREMCPToolsResponse struct {
	Tools []appsresmartbot.MCPToolDescriptor `json:"tools"`
}

type invokeSREMCPToolRequest struct {
	ServerID string `json:"server_id"`
	ToolName string `json:"tool_name"`
}

type getSREAgentDraftResponse = appsresmartbot.AgentDraftResponse
type getSREAgentTriageResponse = appsresmartbot.AgentTriageResponse
type getSREAgentInterpretationResponse = appsresmartbot.AgentInterpretationResponse
type probeSREAgentRuntimeResponse = appsresmartbot.AgentRuntimeProbeResponse

type probeSREAgentRuntimeRequest struct {
	AgentRuntime systemconfig.RobotSREAgentRuntimeConfig `json:"agent_runtime"`
}

type sreDemoScenario struct {
	ID                     string `json:"id"`
	Name                   string `json:"name"`
	Summary                string `json:"summary"`
	RecommendedWalkthrough string `json:"recommended_walkthrough"`
}

type listSREDemoScenariosResponse struct {
	Scenarios []sreDemoScenario `json:"scenarios"`
}

type generateSREDemoIncidentRequest struct {
	ScenarioID string `json:"scenario_id"`
}

type sreApprovalRequest struct {
	ChannelProviderID string `json:"channel_provider_id"`
	RequestMessage    string `json:"request_message"`
}

type sreApprovalDecisionRequest struct {
	Decision string `json:"decision"`
	Comment  string `json:"comment"`
}

type sreMutationResponse struct {
	Incident *sresmartbot.Incident      `json:"incident,omitempty"`
	Action   *sresmartbot.ActionAttempt `json:"action,omitempty"`
	Approval *sresmartbot.Approval      `json:"approval,omitempty"`
}

type detectorRuleSuggestionDecisionRequest struct {
	Reason string `json:"reason"`
}

func (h *SRESmartBotHandler) ListDemoScenarios(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, r.Context(), MethodNotAllowed("Method not allowed"))
		return
	}
	if h.demoService == nil {
		WriteError(w, r.Context(), InternalServer("Demo service is not configured"))
		return
	}

	scenarios := h.demoService.ListScenarios()
	response := make([]sreDemoScenario, 0, len(scenarios))
	for _, scenario := range scenarios {
		response = append(response, sreDemoScenario{
			ID:                     scenario.ID,
			Name:                   scenario.Name,
			Summary:                scenario.Summary,
			RecommendedWalkthrough: scenario.RecommendedWalkthrough,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(listSREDemoScenariosResponse{Scenarios: response})
}

func (h *SRESmartBotHandler) GenerateDemoIncident(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, r.Context(), MethodNotAllowed("Method not allowed"))
		return
	}
	if h.demoService == nil {
		WriteError(w, r.Context(), InternalServer("Demo service is not configured"))
		return
	}

	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		WriteError(w, r.Context(), Unauthorized("Authentication required"))
		return
	}

	var req generateSREDemoIncidentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r.Context(), BadRequest("Invalid request body"))
		return
	}
	req.ScenarioID = strings.TrimSpace(req.ScenarioID)
	if req.ScenarioID == "" {
		WriteError(w, r.Context(), BadRequest("scenario_id is required"))
		return
	}

	tenantID := authCtx.TenantID
	incident, err := h.demoService.GenerateIncident(r.Context(), &tenantID, req.ScenarioID)
	if err != nil {
		h.logger.Error("Failed to generate SRE demo incident", zap.Error(err), zap.String("scenario_id", req.ScenarioID), zap.String("user_id", authCtx.UserID.String()))
		WriteError(w, r.Context(), BadRequest("Failed to generate demo incident").WithCause(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(sreMutationResponse{Incident: incident})
}

func (h *SRESmartBotHandler) ListIncidents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, r.Context(), MethodNotAllowed("Method not allowed"))
		return
	}

	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		WriteError(w, r.Context(), Unauthorized("Authentication required"))
		return
	}

	limit := 50
	offset := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("offset")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			offset = parsed
		}
	}

	filter := sresmartbot.IncidentFilter{
		Domain:   strings.TrimSpace(r.URL.Query().Get("domain")),
		Status:   strings.TrimSpace(r.URL.Query().Get("status")),
		Severity: strings.TrimSpace(r.URL.Query().Get("severity")),
		Search:   strings.TrimSpace(r.URL.Query().Get("search")),
		Limit:    limit,
		Offset:   offset,
	}
	if !authCtx.IsSystemAdmin && !isAllTenantsScopeRequested(r, authCtx) {
		filter.TenantID = &authCtx.TenantID
	}
	if rawTenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id")); rawTenantID != "" {
		tenantID, err := uuid.Parse(rawTenantID)
		if err != nil {
			WriteError(w, r.Context(), BadRequest("Invalid tenant_id"))
			return
		}
		filter.TenantID = &tenantID
	}

	incidents, err := h.repo.ListIncidents(r.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to list SRE incidents", zap.Error(err))
		WriteError(w, r.Context(), InternalServer("Failed to list incidents").WithCause(err))
		return
	}
	total, err := h.repo.CountIncidents(r.Context(), filter)
	if err != nil {
		h.logger.Warn("Failed to count SRE incidents", zap.Error(err))
		total = len(incidents)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(listSREIncidentsResponse{
		Incidents: incidents,
		Total:     total,
		Limit:     limit,
		Offset:    offset,
	})
}

func (h *SRESmartBotHandler) ListDetectorRuleSuggestions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, r.Context(), MethodNotAllowed("Method not allowed"))
		return
	}
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		WriteError(w, r.Context(), Unauthorized("Authentication required"))
		return
	}
	if h.detectorSuggestionService == nil {
		WriteError(w, r.Context(), InternalServer("Detector suggestion service is not configured"))
		return
	}
	limit := 50
	offset := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("offset")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			offset = parsed
		}
	}
	filter := sresmartbot.DetectorRuleSuggestionFilter{
		Status: strings.TrimSpace(r.URL.Query().Get("status")),
		Search: strings.TrimSpace(r.URL.Query().Get("search")),
		Limit:  limit,
		Offset: offset,
	}
	if !authCtx.IsSystemAdmin && !isAllTenantsScopeRequested(r, authCtx) {
		filter.TenantID = &authCtx.TenantID
	}
	if rawTenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id")); rawTenantID != "" {
		tenantID, err := uuid.Parse(rawTenantID)
		if err != nil {
			WriteError(w, r.Context(), BadRequest("Invalid tenant_id"))
			return
		}
		filter.TenantID = &tenantID
	}
	suggestions, err := h.detectorSuggestionService.ListSuggestions(r.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to list detector rule suggestions", zap.Error(err))
		WriteError(w, r.Context(), InternalServer("Failed to list detector rule suggestions").WithCause(err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(listSREDetectorRuleSuggestionsResponse{
		Suggestions: suggestions,
		Limit:       limit,
		Offset:      offset,
	})
}

func (h *SRESmartBotHandler) ProposeDetectorRuleSuggestion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, r.Context(), MethodNotAllowed("Method not allowed"))
		return
	}
	if h.detectorSuggestionService == nil {
		WriteError(w, r.Context(), InternalServer("Detector suggestion service is not configured"))
		return
	}
	incidentID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
	if err != nil {
		WriteError(w, r.Context(), BadRequest("Invalid incident id"))
		return
	}
	authCtx, _ := middleware.GetAuthContext(r)
	suggestion, err := h.detectorSuggestionService.ProposeFromIncident(r.Context(), incidentID, authCtx.UserID.String())
	if err != nil {
		h.logger.Error("Failed to propose detector rule suggestion", zap.Error(err))
		WriteError(w, r.Context(), BadRequest(err.Error()).WithCause(err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(suggestion)
}

func (h *SRESmartBotHandler) AcceptDetectorRuleSuggestion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, r.Context(), MethodNotAllowed("Method not allowed"))
		return
	}
	if h.detectorSuggestionService == nil {
		WriteError(w, r.Context(), InternalServer("Detector suggestion service is not configured"))
		return
	}
	suggestionID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "suggestionId")))
	if err != nil {
		WriteError(w, r.Context(), BadRequest("Invalid suggestion id"))
		return
	}
	authCtx, _ := middleware.GetAuthContext(r)
	suggestion, err := h.detectorSuggestionService.AcceptSuggestion(r.Context(), suggestionID, authCtx.UserID.String())
	if err != nil {
		h.logger.Error("Failed to accept detector rule suggestion", zap.Error(err))
		WriteError(w, r.Context(), BadRequest(err.Error()).WithCause(err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(suggestion)
}

func (h *SRESmartBotHandler) RejectDetectorRuleSuggestion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, r.Context(), MethodNotAllowed("Method not allowed"))
		return
	}
	if h.detectorSuggestionService == nil {
		WriteError(w, r.Context(), InternalServer("Detector suggestion service is not configured"))
		return
	}
	suggestionID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "suggestionId")))
	if err != nil {
		WriteError(w, r.Context(), BadRequest("Invalid suggestion id"))
		return
	}
	var req detectorRuleSuggestionDecisionRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	authCtx, _ := middleware.GetAuthContext(r)
	suggestion, err := h.detectorSuggestionService.RejectSuggestion(r.Context(), suggestionID, authCtx.UserID.String(), req.Reason)
	if err != nil {
		h.logger.Error("Failed to reject detector rule suggestion", zap.Error(err))
		WriteError(w, r.Context(), BadRequest(err.Error()).WithCause(err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(suggestion)
}

func (h *SRESmartBotHandler) GetIncident(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, r.Context(), MethodNotAllowed("Method not allowed"))
		return
	}

	incidentID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
	if err != nil {
		WriteError(w, r.Context(), BadRequest("Invalid incident id"))
		return
	}

	incident, err := h.repo.GetIncident(r.Context(), incidentID)
	if err != nil {
		h.logger.Error("Failed to get SRE incident", zap.Error(err), zap.String("incident_id", incidentID.String()))
		WriteError(w, r.Context(), NotFound("Incident not found").WithCause(err))
		return
	}
	findings, _ := h.repo.ListFindingsByIncident(r.Context(), incidentID)
	evidence, _ := h.repo.ListEvidenceByIncident(r.Context(), incidentID)
	actions, _ := h.repo.ListActionAttemptsByIncident(r.Context(), incidentID)
	approvals, _ := h.repo.ListApprovalsByIncident(r.Context(), incidentID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(getSREIncidentResponse{
		Incident:       incident,
		Findings:       findings,
		Evidence:       evidence,
		ActionAttempts: actions,
		Approvals:      approvals,
	})
}

func (h *SRESmartBotHandler) GetIncidentWorkspace(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, r.Context(), MethodNotAllowed("Method not allowed"))
		return
	}
	if h.workspaceService == nil {
		WriteError(w, r.Context(), InternalServer("Incident workspace service is not configured"))
		return
	}

	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		WriteError(w, r.Context(), Unauthorized("Authentication required"))
		return
	}

	incidentID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
	if err != nil {
		WriteError(w, r.Context(), BadRequest("Invalid incident id"))
		return
	}

	var tenantID *uuid.UUID
	if !isAllTenantsScopeRequested(r, authCtx) {
		tenantID = &authCtx.TenantID
	}

	workspace, err := h.workspaceService.BuildIncidentWorkspace(r.Context(), tenantID, incidentID)
	if err != nil {
		h.logger.Error("Failed to build SRE incident workspace", zap.Error(err), zap.String("incident_id", incidentID.String()))
		WriteError(w, r.Context(), InternalServer("Failed to build incident workspace").WithCause(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(getSREIncidentWorkspaceResponse(*workspace))
}

func (h *SRESmartBotHandler) ListMCPTools(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, r.Context(), MethodNotAllowed("Method not allowed"))
		return
	}
	if h.mcpService == nil {
		WriteError(w, r.Context(), InternalServer("MCP service is not configured"))
		return
	}
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		WriteError(w, r.Context(), Unauthorized("Authentication required"))
		return
	}
	incidentID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
	if err != nil {
		WriteError(w, r.Context(), BadRequest("Invalid incident id"))
		return
	}
	var tenantID *uuid.UUID
	if !isAllTenantsScopeRequested(r, authCtx) {
		tenantID = &authCtx.TenantID
	}
	tools, err := h.mcpService.ListAvailableTools(r.Context(), tenantID, incidentID)
	if err != nil {
		h.logger.Error("Failed to list MCP tools", zap.Error(err), zap.String("incident_id", incidentID.String()))
		WriteError(w, r.Context(), InternalServer("Failed to list MCP tools").WithCause(err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(listSREMCPToolsResponse{Tools: tools})
}

func (h *SRESmartBotHandler) InvokeMCPTool(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, r.Context(), MethodNotAllowed("Method not allowed"))
		return
	}
	if h.mcpService == nil {
		WriteError(w, r.Context(), InternalServer("MCP service is not configured"))
		return
	}
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		WriteError(w, r.Context(), Unauthorized("Authentication required"))
		return
	}
	incidentID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
	if err != nil {
		WriteError(w, r.Context(), BadRequest("Invalid incident id"))
		return
	}
	var req invokeSREMCPToolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r.Context(), BadRequest("Invalid request body"))
		return
	}
	var tenantID *uuid.UUID
	if !isAllTenantsScopeRequested(r, authCtx) {
		tenantID = &authCtx.TenantID
	}
	result, err := h.mcpService.InvokeTool(r.Context(), tenantID, appsresmartbot.MCPToolInvocationRequest{
		IncidentID: incidentID,
		ServerID:   strings.TrimSpace(req.ServerID),
		ToolName:   strings.TrimSpace(req.ToolName),
	})
	if err != nil {
		h.logger.Error("Failed to invoke MCP tool", zap.Error(err), zap.String("incident_id", incidentID.String()), zap.String("server_id", req.ServerID), zap.String("tool_name", req.ToolName))
		WriteError(w, r.Context(), InternalServer("Failed to invoke MCP tool").WithCause(err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(result)
}

func (h *SRESmartBotHandler) GetAgentDraft(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, r.Context(), MethodNotAllowed("Method not allowed"))
		return
	}
	if h.agentService == nil {
		WriteError(w, r.Context(), InternalServer("Agent service is not configured"))
		return
	}
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		WriteError(w, r.Context(), Unauthorized("Authentication required"))
		return
	}
	incidentID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
	if err != nil {
		WriteError(w, r.Context(), BadRequest("Invalid incident id"))
		return
	}
	var tenantID *uuid.UUID
	if !isAllTenantsScopeRequested(r, authCtx) {
		tenantID = &authCtx.TenantID
	}
	draft, err := h.agentService.BuildDraft(r.Context(), tenantID, incidentID)
	if err != nil {
		h.logger.Error("Failed to build SRE agent draft", zap.Error(err), zap.String("incident_id", incidentID.String()))
		WriteError(w, r.Context(), InternalServer("Failed to build agent draft").WithCause(err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(getSREAgentDraftResponse(*draft))
}

func (h *SRESmartBotHandler) GetAgentTriage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, r.Context(), MethodNotAllowed("Method not allowed"))
		return
	}
	if h.agentService == nil {
		WriteError(w, r.Context(), InternalServer("Agent service is not configured"))
		return
	}
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		WriteError(w, r.Context(), Unauthorized("Authentication required"))
		return
	}
	incidentID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
	if err != nil {
		WriteError(w, r.Context(), BadRequest("Invalid incident id"))
		return
	}
	var tenantID *uuid.UUID
	if !isAllTenantsScopeRequested(r, authCtx) {
		tenantID = &authCtx.TenantID
	}
	triage, err := h.agentService.BuildTriage(r.Context(), tenantID, incidentID)
	if err != nil {
		h.logger.Error("Failed to build SRE agent triage", zap.Error(err), zap.String("incident_id", incidentID.String()))
		WriteError(w, r.Context(), InternalServer("Failed to build agent triage").WithCause(err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(getSREAgentTriageResponse(*triage))
}

func (h *SRESmartBotHandler) GetAgentInterpretation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, r.Context(), MethodNotAllowed("Method not allowed"))
		return
	}
	if h.interpretService == nil {
		WriteError(w, r.Context(), InternalServer("Interpretation service is not configured"))
		return
	}
	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		WriteError(w, r.Context(), Unauthorized("Authentication required"))
		return
	}
	incidentID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
	if err != nil {
		WriteError(w, r.Context(), BadRequest("Invalid incident id"))
		return
	}
	var tenantID *uuid.UUID
	if !isAllTenantsScopeRequested(r, authCtx) {
		tenantID = &authCtx.TenantID
	}
	resp, err := h.interpretService.BuildInterpretation(r.Context(), tenantID, incidentID)
	if err != nil {
		h.logger.Error("Failed to build SRE agent interpretation", zap.Error(err), zap.String("incident_id", incidentID.String()))
		WriteError(w, r.Context(), InternalServer("Failed to build agent interpretation").WithCause(err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(getSREAgentInterpretationResponse(*resp))
}

func (h *SRESmartBotHandler) ProbeAgentRuntime(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, r.Context(), MethodNotAllowed("Method not allowed"))
		return
	}
	if h.probeService == nil {
		WriteError(w, r.Context(), InternalServer("Agent runtime probe service is not configured"))
		return
	}

	var req probeSREAgentRuntimeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r.Context(), BadRequest("Invalid request body").WithCause(err))
		return
	}

	resp, err := h.probeService.Probe(r.Context(), req.AgentRuntime)
	if err != nil {
		h.logger.Error("Failed to probe SRE agent runtime", zap.Error(err))
		WriteError(w, r.Context(), InternalServer("Failed to probe agent runtime").WithCause(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(probeSREAgentRuntimeResponse(*resp))
}

func (h *SRESmartBotHandler) ListApprovals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, r.Context(), MethodNotAllowed("Method not allowed"))
		return
	}

	authCtx, ok := middleware.GetAuthContext(r)
	if !ok {
		WriteError(w, r.Context(), Unauthorized("Authentication required"))
		return
	}

	limit := 50
	offset := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("offset")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			offset = parsed
		}
	}

	filter := sresmartbot.ApprovalFilter{
		Status:            strings.TrimSpace(r.URL.Query().Get("status")),
		ChannelProviderID: strings.TrimSpace(r.URL.Query().Get("channel_provider_id")),
		Search:            strings.TrimSpace(r.URL.Query().Get("search")),
		Limit:             limit,
		Offset:            offset,
	}
	if !authCtx.IsSystemAdmin && !isAllTenantsScopeRequested(r, authCtx) {
		filter.TenantID = &authCtx.TenantID
	}
	if rawTenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id")); rawTenantID != "" {
		tenantID, err := uuid.Parse(rawTenantID)
		if err != nil {
			WriteError(w, r.Context(), BadRequest("Invalid tenant_id"))
			return
		}
		filter.TenantID = &tenantID
	}

	approvals, err := h.repo.ListApprovals(r.Context(), filter)
	if err != nil {
		h.logger.Error("Failed to list SRE approvals", zap.Error(err))
		WriteError(w, r.Context(), InternalServer("Failed to list approvals").WithCause(err))
		return
	}
	total, err := h.repo.CountApprovals(r.Context(), filter)
	if err != nil {
		h.logger.Warn("Failed to count SRE approvals", zap.Error(err))
		total = len(approvals)
	}

	items := make([]*sreApprovalQueueItem, 0, len(approvals))
	for _, approval := range approvals {
		if approval == nil {
			continue
		}
		item := &sreApprovalQueueItem{Approval: approval}
		incident, incidentErr := h.repo.GetIncident(r.Context(), approval.IncidentID)
		if incidentErr == nil {
			item.Incident = incident
			if approval.ActionAttemptID != nil {
				actions, actionsErr := h.repo.ListActionAttemptsByIncident(r.Context(), approval.IncidentID)
				if actionsErr == nil {
					for _, action := range actions {
						if action != nil && action.ID == *approval.ActionAttemptID {
							item.Action = action
							break
						}
					}
				}
			}
		}
		items = append(items, item)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(listSREApprovalsResponse{
		Approvals: items,
		Total:     total,
		Limit:     limit,
		Offset:    offset,
	})
}

func (h *SRESmartBotHandler) RequestApproval(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, r.Context(), MethodNotAllowed("Method not allowed"))
		return
	}
	if h.actionService == nil {
		WriteError(w, r.Context(), InternalServer("SRE action service not configured"))
		return
	}
	incidentID, actionID, ok := parseIncidentAndActionIDs(w, r)
	if !ok {
		return
	}
	var req sreApprovalRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	authCtx, _ := middleware.GetAuthContext(r)
	approval, action, err := h.actionService.RequestApproval(r.Context(), incidentID, actionID, authCtx.UserID.String(), strings.TrimSpace(req.ChannelProviderID), strings.TrimSpace(req.RequestMessage))
	if err != nil {
		h.logger.Error("Failed to request SRE approval", zap.Error(err))
		WriteError(w, r.Context(), BadRequest(err.Error()).WithCause(err))
		return
	}
	incident, _ := h.repo.GetIncident(r.Context(), incidentID)
	writeSREMutationResponse(w, incident, action, approval)
}

func (h *SRESmartBotHandler) DecideApproval(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, r.Context(), MethodNotAllowed("Method not allowed"))
		return
	}
	if h.actionService == nil {
		WriteError(w, r.Context(), InternalServer("SRE action service not configured"))
		return
	}
	incidentID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
	if err != nil {
		WriteError(w, r.Context(), BadRequest("Invalid incident id"))
		return
	}
	approvalID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "approvalId")))
	if err != nil {
		WriteError(w, r.Context(), BadRequest("Invalid approval id"))
		return
	}
	var req sreApprovalDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, r.Context(), BadRequest("Invalid request body"))
		return
	}
	authCtx, _ := middleware.GetAuthContext(r)
	approval, action, actionErr := h.actionService.DecideApproval(r.Context(), incidentID, approvalID, req.Decision, authCtx.UserID.String(), req.Comment)
	if actionErr != nil {
		h.logger.Error("Failed to decide SRE approval", zap.Error(actionErr))
		WriteError(w, r.Context(), BadRequest(actionErr.Error()).WithCause(actionErr))
		return
	}
	incident, _ := h.repo.GetIncident(r.Context(), incidentID)
	writeSREMutationResponse(w, incident, action, approval)
}

func (h *SRESmartBotHandler) ExecuteAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, r.Context(), MethodNotAllowed("Method not allowed"))
		return
	}
	if h.actionService == nil {
		WriteError(w, r.Context(), InternalServer("SRE action service not configured"))
		return
	}
	incidentID, actionID, ok := parseIncidentAndActionIDs(w, r)
	if !ok {
		return
	}
	authCtx, _ := middleware.GetAuthContext(r)
	action, err := h.actionService.ExecuteAction(r.Context(), incidentID, actionID, authCtx.UserID.String(), &authCtx.TenantID)
	if err != nil {
		h.logger.Error("Failed to execute SRE action", zap.Error(err))
		WriteError(w, r.Context(), BadRequest(err.Error()).WithCause(err))
		return
	}
	incident, _ := h.repo.GetIncident(r.Context(), incidentID)
	writeSREMutationResponse(w, incident, action, nil)
}

func (h *SRESmartBotHandler) EmailIncidentSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, r.Context(), MethodNotAllowed("Method not allowed"))
		return
	}
	if h.actionService == nil {
		WriteError(w, r.Context(), InternalServer("SRE action service not configured"))
		return
	}
	incidentID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
	if err != nil {
		WriteError(w, r.Context(), BadRequest("Invalid incident id"))
		return
	}
	authCtx, _ := middleware.GetAuthContext(r)
	action, actionErr := h.actionService.TriggerIncidentSummaryEmail(r.Context(), incidentID, authCtx.UserID.String(), &authCtx.TenantID)
	if actionErr != nil {
		h.logger.Error("Failed to email SRE incident summary", zap.Error(actionErr))
		WriteError(w, r.Context(), BadRequest(actionErr.Error()).WithCause(actionErr))
		return
	}
	incident, _ := h.repo.GetIncident(r.Context(), incidentID)
	writeSREMutationResponse(w, incident, action, nil)
}

func parseIncidentAndActionIDs(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	incidentID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
	if err != nil {
		WriteError(w, r.Context(), BadRequest("Invalid incident id"))
		return uuid.Nil, uuid.Nil, false
	}
	actionID, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "actionId")))
	if err != nil {
		WriteError(w, r.Context(), BadRequest("Invalid action id"))
		return uuid.Nil, uuid.Nil, false
	}
	return incidentID, actionID, true
}

func writeSREMutationResponse(w http.ResponseWriter, incident *sresmartbot.Incident, action *sresmartbot.ActionAttempt, approval *sresmartbot.Approval) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(sreMutationResponse{
		Incident: incident,
		Action:   action,
		Approval: approval,
	})
}
