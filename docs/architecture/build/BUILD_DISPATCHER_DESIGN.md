# Build Dispatcher Design

This document describes the dispatcher-driven build execution model used by Image Factory, including queue representation, claiming behavior, and reliability goals.

## 1) Goals
- **Scalable**: Support high concurrency, multi-tenant fairness, and node/pool capacity.
- **Reliable**: Idempotent dispatch, retry semantics, and resilient to worker crashes.
- **Observable**: Clear audit trail across `builds` and `build_executions` with metrics and events.
- **Extensible**: Works with local executors today and Tekton pipelines tomorrow.

## 2) Non-Goals
- Dedicated queue tables.
- Complex job graph orchestration (DAG builds) is out of scope for the current implementation.

---

## 3) Core Data Model (No Queue Tables)

**Existing tables used**
- `builds` (authoritative status)
- `build_configs` (one build = one method config)
- `build_executions` (execution instance + logs + artifacts)

**Queue representation**
- `builds.status = 'queued'` is the queue.

**Key statuses**
- Build: `pending -> queued -> running -> completed/failed/cancelled`
- Execution: `queued -> running -> succeeded/failed/cancelled`

---

## 4) High-Level Flow

```
User submits build
  -> Build + BuildConfig stored
  -> Build status = queued
Dispatcher polls queued builds
  -> Select next eligible build (tenant fairness, limits)
  -> Create build_executions row
  -> Execute (local or Tekton)
  -> Stream logs/status
  -> Update build status
```

---

## 5) Dispatcher Responsibilities

- **Polling**: Find queued builds and claim one atomically.
- **Eligibility**: Enforce concurrency limits and tool availability.
- **Selection**: Tenant fairness + priority + resource fit.
- **Execution**: Create execution record and trigger executor.
- **Recovery**: Requeue orphaned builds if a dispatcher crashes.

---

## 6) Robust Dispatch Mechanics (SQL-first)

### 6.1 Atomic claim (single build)
Use `SELECT ... FOR UPDATE SKIP LOCKED` to avoid double dispatch.

```sql
WITH next_build AS (
  SELECT id
  FROM builds
  WHERE status = 'queued'
  ORDER BY created_at ASC
  FOR UPDATE SKIP LOCKED
  LIMIT 1
)
UPDATE builds
SET status = 'running', started_at = NOW()
WHERE id IN (SELECT id FROM next_build)
RETURNING id;
```

### 6.2 Idempotent execution record
Create exactly one running execution per build.

```sql
INSERT INTO build_executions (id, build_id, config_id, status, created_at, updated_at)
VALUES ($1, $2, $3, 'running', NOW(), NOW())
ON CONFLICT (build_id, status)
DO NOTHING;
```

### 6.3 Orphan recovery
Detect `running` builds with no active execution heartbeat and requeue.

```sql
UPDATE builds
SET status = 'queued'
WHERE status = 'running'
  AND id IN (
    SELECT build_id
    FROM build_executions
    WHERE status = 'running'
      AND updated_at < NOW() - INTERVAL '10 minutes'
  );
```

---

## 7) Scheduler Policy (Robust)

### 7.1 Tenant fairness
- Round-robin by tenant.
- Cap max running per tenant.

### 7.2 Priority (optional, metadata-driven)
- Add optional `priority` into `builds.metadata` or `build_configs.metadata`.

### 7.3 Resource fit
- Match requested CPU/memory to available capacity.
- Use infra selector to choose local vs Tekton.

---

## 8) Interfaces (Go)

### 8.1 Dispatcher
```go
// Dispatcher watches queued builds and dispatches them.
type Dispatcher interface {
    Run(ctx context.Context) error
    Stop() error
}

// DispatcherConfig controls polling and limits.
type DispatcherConfig struct {
    PollInterval time.Duration
    MaxConcurrent int
    MaxPerTenant int
}
```

### 8.2 Scheduler
```go
// Scheduler selects which build to run next.
type Scheduler interface {
    Next(ctx context.Context) (*build.Build, error)
}
```

### 8.3 Execution Orchestrator
```go
// ExecutionOrchestrator creates executions and triggers executors.
type ExecutionOrchestrator interface {
    StartExecution(ctx context.Context, b *build.Build) (*build.BuildExecution, error)
    CancelExecution(ctx context.Context, buildID uuid.UUID) error
}
```

### 8.4 Build Executor
```go
// BuildExecutor runs build logic (local or Tekton).
type BuildExecutor interface {
    Execute(ctx context.Context, build *build.Build) (*build.BuildResult, error)
    Cancel(ctx context.Context, buildID uuid.UUID) error
}
```

### 8.5 Execution Repository
```go
// BuildExecutionRepository persists execution state.
type BuildExecutionRepository interface {
    SaveExecution(ctx context.Context, execution *build.BuildExecution) error
    UpdateExecution(ctx context.Context, execution *build.BuildExecution) error
    GetRunningExecutionForBuild(ctx context.Context, buildID uuid.UUID) (*build.BuildExecution, error)
}
```

---

## 9) Tekton Integration Plan

### 9.1 Executor implementation
- `TektonExecutor` implements `BuildExecutor`
- Creates `PipelineRun`
- Streams logs from Tekton
- Updates `build_executions` status

### 9.2 Mapping method -> pipeline
- Method maps to pipeline template name
- Template rendering uses `build_configs`

---

## 10) Observability

- **Metrics**: queue depth, dispatch latency, running builds, per-tenant concurrency
- **Logs**: dispatcher claims, execution lifecycle, errors
- **Events**: `build.execution.started`, `build.execution.completed`, `build.execution.failed`

---

## 11) Security & Multi-Tenancy

- Dispatcher never cross-tenant data except scheduling.
- Execution uses tenant-scoped namespaces or isolated infra.
- Build config secrets must be resolved at execution time.

---

## 12) Migration Strategy

1. Dispatcher reads `builds.status = queued`, uses available executors.
2. Execution records + log streaming are wired for visibility.
3. Tekton executor support is enabled when Kubernetes infrastructure is selected.

---

## 13) Risks + Mitigations

- **Double dispatch**: use `FOR UPDATE SKIP LOCKED` + execution uniqueness.
- **Stuck builds**: orphan detection + requeue.
- **Queue starvation**: enforce per-tenant round-robin.

---

## 14) Open Questions
- Where to store priority (new column vs metadata)?
- Should we allow manual retry from UI that spawns a new execution?
- When should build status flip to `queued` (immediately on create vs explicit start)?
