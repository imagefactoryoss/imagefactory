# Workflow Engine Architecture

This document describes a proposed workflow engine model for approvals, gatekeeping, and policy-driven transitions such as onboarding, role changes, promotions, and vulnerability review.

It is design-oriented reference material, not a statement that every workflow described here is already implemented in the published repository.

## Why This Fits Image Factory

Workflow orchestration is needed for:
- Onboarding approvals
- Role additions
- Image promotion with tag gates
- CVE validation and security policy gates

This design builds on existing patterns:
- Dispatcher + status-based queue
- Event bus for async coordination
- PostgreSQL as source of truth

---

## Design Goals

- **Scalable**: stateless workers, SKIP LOCKED claiming
- **Auditable**: immutable event log for each workflow
- **Idempotent**: safe retries and step replays
- **Pluggable**: step handlers for approvals, validation, build, promotion
- **Integrates cleanly** with current build/dispatcher architecture

---

## Core Data Model

### Tables (Proposed)

```sql
CREATE TABLE workflow_definitions (
  id UUID PRIMARY KEY,
  name VARCHAR(100) NOT NULL,
  version INT NOT NULL,
  definition JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (name, version)
);

CREATE TABLE workflow_instances (
  id UUID PRIMARY KEY,
  definition_id UUID NOT NULL REFERENCES workflow_definitions(id),
  tenant_id UUID,
  subject_type VARCHAR(50) NOT NULL,
  subject_id UUID NOT NULL,
  status VARCHAR(20) NOT NULL, -- running, blocked, failed, completed
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE workflow_steps (
  id UUID PRIMARY KEY,
  instance_id UUID NOT NULL REFERENCES workflow_instances(id),
  step_key VARCHAR(100) NOT NULL,
  status VARCHAR(20) NOT NULL, -- pending, running, succeeded, failed, blocked
  attempts INT NOT NULL DEFAULT 0,
  last_error TEXT,
  started_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ
);

CREATE TABLE workflow_events (
  id UUID PRIMARY KEY,
  instance_id UUID NOT NULL REFERENCES workflow_instances(id),
  step_id UUID,
  type VARCHAR(50) NOT NULL,
  payload JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_workflow_instances_status ON workflow_instances(status);
CREATE INDEX idx_workflow_steps_status ON workflow_steps(status);
```

---

## Execution Model

- **Workflow Orchestrator** polls for runnable steps using `FOR UPDATE SKIP LOCKED`.
- Each step is executed by a **handler** (approval, validation, build, promotion).
- State transitions are written atomically.
- Workflow emits events on each transition.

### Step Claim (Example)

```sql
WITH next_step AS (
  SELECT ws.id
  FROM workflow_steps ws
  JOIN workflow_instances wi ON wi.id = ws.instance_id
  WHERE ws.status = 'pending'
    AND wi.status = 'running'
  ORDER BY wi.created_at ASC
  FOR UPDATE SKIP LOCKED
  LIMIT 1
)
UPDATE workflow_steps
SET status = 'running', started_at = now()
WHERE id IN (SELECT id FROM next_step)
RETURNING *;
```

---

## Interfaces (Go)

```go
type WorkflowOrchestrator interface {
  Run(ctx context.Context) error
}

type WorkflowRepository interface {
  ClaimNextStep(ctx context.Context) (*WorkflowStep, error)
  UpdateStep(ctx context.Context, step *WorkflowStep) error
  UpdateInstance(ctx context.Context, inst *WorkflowInstance) error
  AppendEvent(ctx context.Context, evt *WorkflowEvent) error
}

type StepHandler interface {
  Key() string
  Execute(ctx context.Context, step *WorkflowStep) (StepResult, error)
}

type StepResult struct {
  Status string // succeeded, failed, blocked
  Data   map[string]any
}
```

---

## Integration with Current System

### Build Flow
- Step: `queue_build`
  - Calls existing build service to create build with `status = queued`
  - Dispatcher continues execution as today
- Step: `await_build_completion`
  - Waits on build completion events (from event bus)

### Approvals
- Step: `approval_gate`
  - Creates `approval_requests`
  - Blocks until approval event arrives

### CVE Validation
- Step: `cve_check`
  - Calls vulnerability scan service
  - Fails or blocks based on policy thresholds

### Promotion
- Step: `promote_image`
  - Updates tag in registry and `images` metadata

---

## Suggested First Milestone

1. Add schema tables.
2. Implement orchestrator with:
   - `queue_build`
   - `await_build_completion`
3. Add approval step wired to existing approval tables.
4. Emit events on transitions for observability.

---

## Why This Scales

- Stateless workers and DB claiming
- Idempotent steps with retries
- Event-driven blocking/unblocking
- Clear separation between orchestration and execution
