package rest

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type WorkflowHandler struct {
	db     *sqlx.DB
	logger *zap.Logger
}

func NewWorkflowHandler(db *sqlx.DB, logger *zap.Logger) *WorkflowHandler {
	return &WorkflowHandler{db: db, logger: logger}
}

type workflowStepRequest struct {
	StepKey string                 `json:"step_key"`
	Payload map[string]interface{} `json:"payload,omitempty"`
}

type createWorkflowRequest struct {
	Name        string                 `json:"name"`
	Version     int                    `json:"version"`
	TenantID    *uuid.UUID             `json:"tenant_id,omitempty"`
	SubjectType string                 `json:"subject_type"`
	SubjectID   *uuid.UUID             `json:"subject_id,omitempty"`
	Definition  map[string]interface{} `json:"definition,omitempty"`
	Steps       []workflowStepRequest  `json:"steps"`
}

type workflowInstanceResponse struct {
	ID           uuid.UUID              `json:"id"`
	DefinitionID uuid.UUID              `json:"definition_id"`
	TenantID     *uuid.UUID             `json:"tenant_id,omitempty"`
	SubjectType  string                 `json:"subject_type"`
	SubjectID    uuid.UUID              `json:"subject_id"`
	Status       string                 `json:"status"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	Steps        []workflowStepResponse `json:"steps"`
}

type workflowStepResponse struct {
	ID          uuid.UUID              `json:"id"`
	StepKey     string                 `json:"step_key"`
	Status      string                 `json:"status"`
	Attempts    int                    `json:"attempts"`
	LastError   *string                `json:"last_error,omitempty"`
	Payload     map[string]interface{} `json:"payload,omitempty"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

func (h *WorkflowHandler) CreateWorkflow(w http.ResponseWriter, r *http.Request) {
	var req createWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.Version <= 0 || req.SubjectType == "" {
		writeJSONError(w, http.StatusBadRequest, "name, version, and subject_type are required")
		return
	}
	if len(req.Steps) == 0 {
		writeJSONError(w, http.StatusBadRequest, "at least one step is required")
		return
	}

	definition := req.Definition
	if definition == nil {
		definition = map[string]interface{}{}
	}

	defBytes, err := json.Marshal(definition)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid definition")
		return
	}

	tx, err := h.db.BeginTxx(r.Context(), nil)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to start transaction")
		return
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var definitionID uuid.UUID
	defQuery := `
		INSERT INTO workflow_definitions (name, version, definition)
		VALUES ($1, $2, $3)
		ON CONFLICT (name, version)
		DO UPDATE SET definition = EXCLUDED.definition, updated_at = now()
		RETURNING id`
	if err = tx.GetContext(r.Context(), &definitionID, defQuery, req.Name, req.Version, defBytes); err != nil {
		_ = tx.Rollback()
		writeJSONError(w, http.StatusInternalServerError, "failed to save workflow definition")
		return
	}

	subjectID := uuid.New()
	if req.SubjectID != nil {
		subjectID = *req.SubjectID
	}

	var instanceID uuid.UUID
	instQuery := `
		INSERT INTO workflow_instances (definition_id, tenant_id, subject_type, subject_id, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`
	if err = tx.GetContext(r.Context(), &instanceID, instQuery, definitionID, req.TenantID, req.SubjectType, subjectID, "running"); err != nil {
		_ = tx.Rollback()
		writeJSONError(w, http.StatusInternalServerError, "failed to create workflow instance")
		return
	}

	stepInsert := `
		INSERT INTO workflow_steps (instance_id, step_key, payload, status)
		VALUES ($1, $2, $3, $4)`
	for _, step := range req.Steps {
		if step.StepKey == "" {
			_ = tx.Rollback()
			writeJSONError(w, http.StatusBadRequest, "step_key is required for each step")
			return
		}
		payloadBytes, err := json.Marshal(step.Payload)
		if err != nil {
			_ = tx.Rollback()
			writeJSONError(w, http.StatusBadRequest, "invalid step payload")
			return
		}
		if _, err = tx.ExecContext(r.Context(), stepInsert, instanceID, step.StepKey, payloadBytes, "pending"); err != nil {
			_ = tx.Rollback()
			writeJSONError(w, http.StatusInternalServerError, "failed to create workflow steps")
			return
		}
	}

	if err = tx.Commit(); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to commit workflow")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"id": instanceID,
	})
}

func (h *WorkflowHandler) GetWorkflow(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	instanceID, err := uuid.Parse(idStr)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid workflow id")
		return
	}

	var instance workflowInstanceResponse
	instQuery := `
		SELECT id, definition_id, tenant_id, subject_type, subject_id, status, created_at, updated_at
		FROM workflow_instances
		WHERE id = $1`
	if err := h.db.GetContext(r.Context(), &instance, instQuery, instanceID); err != nil {
		writeJSONError(w, http.StatusNotFound, "workflow not found")
		return
	}

	stepQuery := `
		SELECT id, step_key, status, attempts, last_error, payload, started_at, completed_at, created_at, updated_at
		FROM workflow_steps
		WHERE instance_id = $1
		ORDER BY created_at ASC`
	rows, err := h.db.QueryxContext(r.Context(), stepQuery, instanceID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to load workflow steps")
		return
	}
	defer rows.Close()

	var steps []workflowStepResponse
	for rows.Next() {
		var (
			step         workflowStepResponse
			payloadBytes []byte
		)
		if err := rows.Scan(&step.ID, &step.StepKey, &step.Status, &step.Attempts, &step.LastError, &payloadBytes, &step.StartedAt, &step.CompletedAt, &step.CreatedAt, &step.UpdatedAt); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to read workflow steps")
			return
		}
		if len(payloadBytes) > 0 {
			_ = json.Unmarshal(payloadBytes, &step.Payload)
		}
		steps = append(steps, step)
	}
	instance.Steps = steps

	writeJSON(w, http.StatusOK, instance)
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
