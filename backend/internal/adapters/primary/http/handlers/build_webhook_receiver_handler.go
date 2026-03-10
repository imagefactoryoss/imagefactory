package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/srikarm/image-factory/internal/adapters/secondary/postgres"
	"github.com/srikarm/image-factory/internal/domain/build"
	"github.com/srikarm/image-factory/internal/domain/project"
	"go.uber.org/zap"
)

type BuildWebhookReceiverHandler struct {
	buildService *build.Service
	projectSvc   *project.Service
	sourceRepo   *postgres.ProjectSourceRepository
	receiptRepo  *postgres.WebhookReceiptRepository
	logger       *zap.Logger
}

func NewBuildWebhookReceiverHandler(
	buildService *build.Service,
	projectSvc *project.Service,
	sourceRepo *postgres.ProjectSourceRepository,
	receiptRepo *postgres.WebhookReceiptRepository,
	logger *zap.Logger,
) *BuildWebhookReceiverHandler {
	return &BuildWebhookReceiverHandler{
		buildService: buildService,
		projectSvc:   projectSvc,
		sourceRepo:   sourceRepo,
		receiptRepo:  receiptRepo,
		logger:       logger,
	}
}

type inboundWebhookEvent struct {
	EventType string
	Ref       string
	Branch    string
	CommitSHA string
	RepoURL   string
	RawBody   []byte
}

func (h *BuildWebhookReceiverHandler) ReceiveProjectWebhook(w http.ResponseWriter, r *http.Request) {
	provider := strings.ToLower(strings.TrimSpace(chi.URLParam(r, "provider")))
	if provider == "" {
		h.respondError(w, http.StatusBadRequest, "provider is required")
		return
	}

	projectID, err := uuid.Parse(chi.URLParam(r, "projectID"))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid project ID")
		return
	}
	projectEntity, err := h.projectSvc.GetProject(r.Context(), projectID)
	if err != nil || projectEntity == nil {
		h.respondError(w, http.StatusNotFound, "project not found")
		return
	}

	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "failed to read webhook payload")
		return
	}

	event, err := parseInboundWebhookEvent(provider, rawBody, r.Header)
	if err != nil {
		h.logger.Warn("Webhook payload parse failed", zap.Error(err), zap.String("provider", provider), zap.String("project_id", projectID.String()))
		h.persistReceipt(r, &postgres.WebhookReceipt{
			TenantID:            projectEntity.TenantID(),
			ProjectID:           projectID,
			Provider:            provider,
			DeliveryID:          nullableString(deliveryIDForProvider(provider, r.Header)),
			EventType:           "unknown",
			EventSHA:            nullableString(hashBody(rawBody)),
			SignatureValid:      false,
			Status:              "rejected",
			Reason:              nullableString(err.Error()),
			MatchedTriggerCount: 0,
			TriggeredBuildIDs:   []string{},
			ReceivedAt:          time.Now().UTC(),
		})
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	deliveryID := deliveryIDForProvider(provider, r.Header)
	if strings.TrimSpace(deliveryID) != "" && h.receiptRepo != nil {
		existing, lookupErr := h.receiptRepo.FindByProjectProviderDelivery(r.Context(), projectID, provider, deliveryID)
		if lookupErr == nil && existing != nil {
			h.respondJSON(w, http.StatusAccepted, map[string]interface{}{
				"status":           "accepted",
				"provider":         provider,
				"project_id":       projectID.String(),
				"event_type":       existing.EventType,
				"matched_triggers": existing.MatchedTriggerCount,
				"triggered_builds": existing.TriggeredBuildIDs,
				"deduped":          true,
			})
			return
		}
	}

	triggers, err := h.buildService.GetProjectTriggers(r.Context(), projectID)
	if err != nil {
		h.logger.Error("Failed to load project triggers", zap.Error(err), zap.String("project_id", projectID.String()))
		h.respondError(w, http.StatusInternalServerError, "failed to resolve project triggers")
		return
	}

	matched := make([]*build.BuildTrigger, 0)
	signatureChecked := false
	signatureValid := false
	for _, trigger := range triggers {
		if trigger == nil || !trigger.IsActive {
			continue
		}
		switch trigger.Type {
		case build.TriggerTypeWebhook:
			if !webhookTriggerMatchesEvent(trigger, provider, event.EventType, event.RepoURL) {
				continue
			}
			signatureChecked = true
			if !verifyWebhookSignature(provider, trigger.WebhookSecret, event.RawBody, r.Header) {
				continue
			}
			signatureValid = true
			matched = append(matched, trigger)
		case build.TriggerTypeGitEvent:
			if !gitEventTriggerMatches(trigger, provider, event.Branch, event.RepoURL) {
				continue
			}
			matched = append(matched, trigger)
		}
	}

	triggeredBuilds := make([]string, 0, len(matched))
	for _, trigger := range matched {
		newBuildID, runErr := h.triggerBuildFromEvent(r, trigger, event, provider)
		if runErr != nil {
			h.logger.Warn("Failed to execute webhook trigger", zap.Error(runErr), zap.String("trigger_id", trigger.ID.String()))
			continue
		}
		triggeredBuilds = append(triggeredBuilds, newBuildID.String())
	}
	status := "accepted"
	reason := ""
	if len(matched) == 0 {
		status = "ignored"
		if signatureChecked && !signatureValid {
			reason = "signature_validation_failed"
		} else {
			reason = "no_matching_trigger"
		}
	}
	h.persistReceipt(r, &postgres.WebhookReceipt{
		TenantID:            projectEntity.TenantID(),
		ProjectID:           projectID,
		Provider:            provider,
		DeliveryID:          nullableString(deliveryID),
		EventType:           event.EventType,
		EventRef:            nullableString(event.Ref),
		EventBranch:         nullableString(event.Branch),
		EventCommitSHA:      nullableString(event.CommitSHA),
		RepoURL:             nullableString(event.RepoURL),
		EventSHA:            nullableString(hashBody(rawBody)),
		SignatureValid:      !signatureChecked || signatureValid,
		Status:              status,
		Reason:              nullableString(reason),
		MatchedTriggerCount: len(matched),
		TriggeredBuildIDs:   triggeredBuilds,
		ReceivedAt:          time.Now().UTC(),
	})

	h.respondJSON(w, http.StatusAccepted, map[string]interface{}{
		"status":           status,
		"provider":         provider,
		"project_id":       projectID.String(),
		"event_type":       event.EventType,
		"matched_triggers": len(matched),
		"triggered_builds": triggeredBuilds,
		"delivery_id":      deliveryID,
		"deduped":          false,
	})
}

func (h *BuildWebhookReceiverHandler) triggerBuildFromEvent(r *http.Request, trigger *build.BuildTrigger, event inboundWebhookEvent, provider string) (uuid.UUID, error) {
	sourceBuild, err := h.buildService.GetBuild(r.Context(), trigger.BuildID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("load source build: %w", err)
	}
	if sourceBuild == nil {
		return uuid.Nil, fmt.Errorf("source build not found")
	}

	manifest := sourceBuild.Manifest()
	if manifest.Metadata == nil {
		manifest.Metadata = map[string]interface{}{}
	}

	refPolicy := metadataString(manifest.Metadata, "ref_policy", "refPolicy")
	if refPolicy == "" {
		refPolicy = "source_default"
	}
	fixedRef := metadataString(manifest.Metadata, "fixed_ref", "fixedRef")
	sourceID := metadataString(manifest.Metadata, "source_id", "sourceId")
	resolvedBranch := strings.TrimSpace(event.Branch)

	switch refPolicy {
	case "fixed":
		if strings.TrimSpace(fixedRef) != "" {
			resolvedBranch = strings.TrimSpace(fixedRef)
		}
	case "source_default":
		if strings.TrimSpace(sourceID) != "" && h.sourceRepo != nil {
			if parsed, parseErr := uuid.Parse(sourceID); parseErr == nil {
				source, findErr := h.sourceRepo.FindByID(r.Context(), sourceBuild.ProjectID(), parsed)
				if findErr == nil && source != nil && strings.TrimSpace(source.DefaultBranch) != "" {
					resolvedBranch = strings.TrimSpace(source.DefaultBranch)
				}
			}
		}
		if resolvedBranch == "" {
			if current := metadataString(manifest.Metadata, "git_branch", "gitBranch", "branch"); current != "" {
				resolvedBranch = current
			}
		}
	case "event_ref":
		// already uses event branch
	}

	manifest.Metadata["git_url"] = event.RepoURL
	manifest.Metadata["git_branch"] = resolvedBranch
	manifest.Metadata["git_commit"] = event.CommitSHA
	manifest.Metadata["trigger_provider"] = provider
	manifest.Metadata["trigger_event"] = event.EventType
	manifest.Metadata["trigger_ref"] = event.Ref
	manifest.Metadata["trigger_type"] = "webhook"

	newBuild, err := h.buildService.CreateBuildDraft(r.Context(), sourceBuild.TenantID(), sourceBuild.ProjectID(), manifest, nil)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create build draft: %w", err)
	}
	if err := h.buildService.StartBuild(r.Context(), newBuild.ID()); err != nil {
		return uuid.Nil, fmt.Errorf("start build: %w", err)
	}
	return newBuild.ID(), nil
}

func webhookTriggerMatchesEvent(trigger *build.BuildTrigger, provider, eventType, repoURL string) bool {
	if trigger == nil || trigger.Type != build.TriggerTypeWebhook || !trigger.IsActive {
		return false
	}
	if len(trigger.WebhookEvents) > 0 {
		found := false
		for _, evt := range trigger.WebhookEvents {
			if strings.EqualFold(strings.TrimSpace(evt), strings.TrimSpace(eventType)) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if strings.TrimSpace(trigger.GitRepoURL) != "" && !repoMatches(repoURL, trigger.GitRepoURL) {
		return false
	}
	return provider != ""
}

func gitEventTriggerMatches(trigger *build.BuildTrigger, provider, branch, repoURL string) bool {
	if trigger == nil || trigger.Type != build.TriggerTypeGitEvent || !trigger.IsActive {
		return false
	}
	if !strings.EqualFold(string(trigger.GitProvider), provider) {
		return false
	}
	if !repoMatches(repoURL, trigger.GitRepoURL) {
		return false
	}
	return branchMatches(branch, trigger.GitBranchPattern)
}

func verifyWebhookSignature(provider, secret string, body []byte, headers http.Header) bool {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return true
	}
	switch strings.ToLower(provider) {
	case "github":
		headerSig := strings.TrimSpace(headers.Get("X-Hub-Signature-256"))
		if headerSig == "" {
			return false
		}
		const prefix = "sha256="
		if !strings.HasPrefix(headerSig, prefix) {
			return false
		}
		gotSig, err := hex.DecodeString(strings.TrimPrefix(headerSig, prefix))
		if err != nil {
			return false
		}
		mac := hmac.New(sha256.New, []byte(secret))
		_, _ = mac.Write(body)
		wantSig := mac.Sum(nil)
		return hmac.Equal(gotSig, wantSig)
	case "gitlab":
		token := strings.TrimSpace(headers.Get("X-Gitlab-Token"))
		return token != "" && hmac.Equal([]byte(token), []byte(secret))
	default:
		return true
	}
}

func parseInboundWebhookEvent(provider string, rawBody []byte, headers http.Header) (inboundWebhookEvent, error) {
	var payload map[string]interface{}
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		return inboundWebhookEvent{}, fmt.Errorf("invalid JSON payload")
	}

	eventType := ""
	switch provider {
	case "github":
		eventType = strings.ToLower(strings.TrimSpace(headers.Get("X-GitHub-Event")))
	case "gitlab":
		eventType = strings.ToLower(strings.TrimSpace(headers.Get("X-Gitlab-Event")))
		if strings.Contains(eventType, "push") {
			eventType = "push"
		}
	}
	if eventType == "" {
		eventType = "push"
	}

	ref := metadataString(payload, "ref")
	branch := strings.TrimPrefix(strings.TrimSpace(ref), "refs/heads/")
	commitSHA := metadataString(payload, "after", "checkout_sha")

	repoURL := ""
	if repo, ok := payload["repository"].(map[string]interface{}); ok {
		repoURL = metadataString(repo, "clone_url", "git_http_url", "git_url", "html_url", "ssh_url")
	}
	if repoURL == "" {
		if project, ok := payload["project"].(map[string]interface{}); ok {
			repoURL = metadataString(project, "git_http_url", "git_ssh_url", "web_url")
		}
	}
	if strings.TrimSpace(repoURL) == "" {
		return inboundWebhookEvent{}, fmt.Errorf("repository URL not found in payload")
	}

	return inboundWebhookEvent{
		EventType: eventType,
		Ref:       ref,
		Branch:    branch,
		CommitSHA: commitSHA,
		RepoURL:   repoURL,
		RawBody:   rawBody,
	}, nil
}

func branchMatches(branch, pattern string) bool {
	branch = strings.TrimSpace(strings.TrimPrefix(branch, "refs/heads/"))
	pattern = strings.TrimSpace(strings.TrimPrefix(pattern, "refs/heads/"))
	if pattern == "" {
		return true
	}
	if pattern == branch {
		return true
	}
	match, err := path.Match(pattern, branch)
	return err == nil && match
}

func repoMatches(left, right string) bool {
	return canonicalRepo(left) == canonicalRepo(right)
}

func canonicalRepo(repo string) string {
	clean := strings.ToLower(strings.TrimSpace(repo))
	clean = strings.TrimPrefix(clean, "https://")
	clean = strings.TrimPrefix(clean, "http://")
	clean = strings.TrimPrefix(clean, "ssh://")
	clean = strings.TrimPrefix(clean, "git@")
	clean = strings.ReplaceAll(clean, ":", "/")
	clean = strings.TrimSuffix(clean, ".git")
	clean = strings.TrimSuffix(clean, "/")
	return clean
}

func hashBody(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func deliveryIDForProvider(provider string, headers http.Header) string {
	switch provider {
	case "github":
		return strings.TrimSpace(headers.Get("X-GitHub-Delivery"))
	case "gitlab":
		if value := strings.TrimSpace(headers.Get("X-Gitlab-Event-UUID")); value != "" {
			return value
		}
		return strings.TrimSpace(headers.Get("X-Request-Id"))
	default:
		return strings.TrimSpace(headers.Get("X-Delivery-Id"))
	}
}

func nullableString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	trimmed := strings.TrimSpace(value)
	return &trimmed
}

func (h *BuildWebhookReceiverHandler) persistReceipt(r *http.Request, receipt *postgres.WebhookReceipt) {
	if h.receiptRepo == nil || receipt == nil {
		return
	}
	if err := h.receiptRepo.Save(r.Context(), receipt); err != nil {
		if postgres.IsUniqueViolation(err) {
			return
		}
		h.logger.Warn("Failed to persist webhook receipt", zap.Error(err), zap.String("project_id", receipt.ProjectID.String()))
	}
}

func metadataString(metadata map[string]interface{}, keys ...string) string {
	if metadata == nil {
		return ""
	}
	for _, key := range keys {
		if value, ok := metadata[key]; ok {
			switch typed := value.(type) {
			case string:
				if strings.TrimSpace(typed) != "" {
					return strings.TrimSpace(typed)
				}
			}
		}
	}
	return ""
}

func (h *BuildWebhookReceiverHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *BuildWebhookReceiverHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]interface{}{
		"error":  message,
		"status": status,
	})
}
