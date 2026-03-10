# Build Tables Overview

This document explains the **build-related tables** (the ones that start with `build_`) and how they connect to the core `builds` table. It is meant to help answer: *what each table is for, and how they tie together.*

## Core Anchor Table

**`builds`** (from `001_initial_schema.up.sql`) is the anchor record for a build. All build-specific tables reference it via `build_id`.

Key fields (high level):
- Identity: `id`, `tenant_id`, `project_id`, `build_number`
- Status: `status`, `started_at`, `completed_at`, `error_message`
- Git metadata: `git_commit`, `git_branch`, `git_author_*`

## Build_* Tables (Current) and Their Purpose

### 1) `build_executions`
- **Purpose:** Execution instance of a build (status, duration, output, error, artifacts). One build can have multiple executions if retried.
- **Primary link:** `build_executions.build_id` → `builds.id`.
- **Also links:** `build_executions.config_id` → `build_configs.id`.

### 2) `build_execution_logs`
- **Purpose:** Structured logs for a specific execution (timestamped and leveled).
- **Primary link:** `build_execution_logs.execution_id` → `build_executions.id`.

### 3) `build_configs`
- **Purpose:** **Current** method-specific build configuration. Holds dockerfile, build context, tool selection, cache, etc.
- **Primary link:** `build_configs.build_id` → `builds.id`.
- **Used by:** `build_repository.go` and build creation flow.

### 4) `build_steps`
- **Purpose:** Per-step status and logs for docker/build steps (RUN/COPY/etc).
- **Primary link:** `build_steps.build_id` → `builds.id`.

### 5) `build_metrics`
- **Purpose:** Resource and performance metrics (duration, CPU/memory, layer info).
- **Primary link:** `build_metrics.build_id` → `builds.id`.

### 6) `build_artifacts`
- **Purpose:** Artifacts emitted by a build (image, SBOM, scan results, logs, etc).
- **Primary link:** `build_artifacts.build_id` → `builds.id`.
- **Optional link:** `build_artifacts.image_id` → `images.id`.

### 7) `build_history`
- **Purpose:** Historical metrics for ETA prediction and analytics.
- **Primary link:** `build_history.build_id` → `builds.id`.
- **Also links:** `build_history.tenant_id`, `build_history.project_id` for aggregates.

### 8) `build_triggers`
- **Purpose:** Webhook/schedule/git-event trigger definitions for builds.
- **Primary link:** `build_triggers.build_id` → `builds.id`.
- **Also links:** `tenant_id`, `project_id`, `created_by`.

### 9) `build_policies`
- **Purpose:** Per-tenant policy rules for builds (limits, scheduling rules, approvals).
- **Primary link:** `build_policies.tenant_id` → `tenants.id`.
- **Notes:** Does not link directly to `builds`, but governs behavior.

> You mentioned seeing **13** tables. The list above includes all tables that start with `build_`.

## Related (non build_) Tables

- `images` and `image_*` tables: store image outputs referenced by `build_artifacts`.

## How They Tie Together (Diagram)

```
builds
  ├─ build_configs ──┬─ build_executions ── build_execution_logs
  │                  └─ (execution artifacts in JSON)
  ├─ build_steps
  ├─ build_metrics
  ├─ build_artifacts ── images
  ├─ build_history
  └─ build_triggers
```

## Recommended Mental Model (Current)

1. **A build request is created** → record in `builds`.
2. **Method-specific config is saved** → `build_configs`.
3. **Scheduling** → `builds.status = queued` (no queue tables).
4. **Execution begins** → `build_executions` + `build_execution_logs`.
5. **Outputs**
   - Artifacts: `build_artifacts`.
   - Metrics: `build_metrics`.
   - History: `build_history` (for ETA/analytics).

## Tables Not Used in the Current Build Flow

- **`build_method_configs`** and **`config_versions`** are not used in the current build flow.
- **`build_queue`** and **`build_queue_position`** are not used; queuing is represented by `builds.status = queued`.
- **`build_execution_logs`** is the source of truth for execution logs.

## Where to Look in Code

- `backend/internal/adapters/secondary/postgres/build_repository.go` → `builds`, `build_configs`
- `backend/internal/domain/build/service.go` → execution state transitions

---

If you want, I can add a simplified ER diagram or annotate which tables are actively written in the current runtime path.
