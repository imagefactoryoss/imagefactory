# Build Domain Models Architecture

This document describes the current build domain model and execution flow used by Image Factory.

## Core Entities

- **Build**: Top-level request and lifecycle container.
- **BuildConfig**: Method-specific configuration (one per build).
- **BuildExecution**: Execution instance for a build.
- **BuildExecutionLog**: Log stream for an execution.
- **BuildArtifact / BuildMetrics / BuildHistory**: Outputs and analytics.

---

## Status Model

- **Build status**: `pending → queued → running → completed|failed|cancelled`
- **Execution status**: `queued → running → succeeded|failed|cancelled`

Builds are queued via `builds.status = 'queued'` and claimed by the dispatcher.

---

## Execution Flow

1. Build is created with a validated `BuildConfig`.
2. Build is queued (`status = queued`).
3. Dispatcher claims the build and creates a `BuildExecution`.
4. Executor runs locally or via Tekton (based on infrastructure selection).
5. Execution updates logs, metrics, artifacts, and final status.

---

## Infrastructure Selection

Builds can be scoped to:
- **Local execution** (default)
- **Kubernetes/Tekton** (via infrastructure provider selection)

Infrastructure selection is stored on the build and used by the dispatcher to pick the correct executor.

---

## Key Tables

- `builds`
- `build_configs`
- `build_executions`
- `build_execution_logs`
- `build_artifacts`
- `build_metrics`
- `build_history`

---

## Invariants

- Exactly one `build_config` per build.
- A build can have multiple executions over time (retries), but only one running at a time.
- Dispatcher is the only component that transitions `queued → running`.
