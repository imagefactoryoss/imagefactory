# SRE Smart Bot Implementation Plan

## Purpose

Turn the SRE Smart Bot design into an incremental delivery plan that fits the current Image Factory backend, admin APIs, and small-cluster operating model.

## Delivery Principle

Build the system in this order:

1. policy and storage
2. incidents and evidence
3. safe actions and approvals
4. operator channels
5. richer agent and MCP capabilities

That keeps the deterministic control plane ahead of the persona layer.

## Architecture Guardrail

SRE Smart Bot must remain modular.

Specifically:

- `backend/cmd/server/main.go` should only compose dependencies and start background workers
- incident correlation, signal mapping, policy evaluation, channel delivery, and action execution should live in dedicated packages
- each signal source should have a small adapter boundary rather than embedding business rules directly in startup code
- new SRE Smart Bot slices should prefer `service + repository + adapter` structure over inline logic in `main.go`

This is important because the backend already has a large startup surface, and SRE Smart Bot will grow quickly if we do not enforce boundaries early.

## Current Checkpoint

Status: `2026-03-14 checkpoint reached`

Completed since kickoff:

- persisted SRE Smart Bot policy config and admin API
- incident ledger schema, repository, and read APIs
- initial watcher signal wiring for runtime dependency and cluster metrics ingester incidents
- modularization checkpoint:
  - runtime dependency watcher extracted
  - cluster metrics ingester extracted
  - dispatcher runner extracted
  - workflow runner extracted
  - stale execution watchdog extracted
  - provider readiness watcher extracted
  - tenant asset drift watcher extracted
  - quarantine release compliance watcher extracted
  - build notification subscriber health reporter extracted
- first product-facing admin incident page route added for `Operations > SRE Smart Bot`
- normalized SRE ledger events now publish through the existing event bus for:
  - `sre.finding.observed`
  - `sre.incident.resolved`
  - `sre.evidence.added`
  - `sre.action.proposed`
- backend ingestion path now exists for detector-published findings through:
  - `sre.detector.finding.observed`
- local observability bootstrap now exists for development:
  - local Loki config
  - local log shipper for `logs/*.log`
  - local Grafana provisioning pointing to Loki

Current emphasis:

- keep `main.go` as a composition root
- continue product work on top of the extracted runner boundaries
- prioritize operator visibility and actionability before richer persona/channel work
- make the local observability workflow usable enough to iterate on detector rules quickly
- start treating golden signals as first-class SRE inputs, beginning with low-risk saturation detection from existing cluster metrics snapshots

## Phase 0: Policy Foundation

Status: `done`

Scope:

- persisted `robot_sre_policy` config
- environment posture (`demo`, `staging`, `production`)
- configurable channel providers through API contract
- operator-defined rule metadata and validation
- implementation backlog and epic alignment

Current slice:

- add backend config model and admin endpoints for SRE Smart Bot policy
- keep rules declarative and bounded

Exit criteria:

- admin can read and update SRE Smart Bot policy
- invalid policy payloads are rejected deterministically
- defaults are safe for demo environment

## Phase 1: Incident Ledger

Status: `in_progress`

Scope:

- findings table
- incidents table
- incident evidence records
- action attempts
- approval requests and decisions
- correlation keys and incident lifecycle transitions

First incident classes:

- `infrastructure.node_disk_pressure`
- `runtime_services.runtime_dependency_outage`
- `release_configuration.registry_auth_or_mirror_failure`
- `identity_security.identity_provider_unreachable`
- `application_services.application_service_degraded`

Exit criteria:

- repeated watcher signals fold into a stable incident record
- incidents expose state: `observed`, `triaged`, `contained`, `recovering`, `resolved`, `suppressed`, `escalated`
- all automated actions leave an auditable trail

Progress now:

- incident, finding, evidence, action-attempt, and approval tables exist
- incident list/detail admin APIs exist
- initial admin incident page exists
- runtime dependency watcher and cluster metrics snapshot ingester already create/resolve incidents

Next for Phase 1:

- wire provider readiness, tenant asset drift, and release compliance watcher results into incident findings
- add evidence capture helpers for watcher-specific detail snapshots
- extend incident UI with filters, counts, and approval/action timeline polish

## Phase 2: Guarded Remediation Engine

Status: `planned`

Scope:

- policy evaluator
- cooldown enforcement
- allowlisted containment actions
- approval gate for recover/disruptive actions
- action runner abstraction for Kubernetes, OCI, and config mutations

V1 allowed actions:

- notify
- delete succeeded/failed pods
- suspend allowlisted CronJobs
- scale allowlisted noncritical workloads to zero

Approval-required actions:

- rollout restart deployment
- patch config
- Helm reconcile
- cordon node
- OCI reboot/replace node

Exit criteria:

- no disruptive action can execute without approval state
- repeated failures do not thrash the same remediation
- runtime dependency and disk-pressure incidents can be contained automatically in demo mode

## Phase 3: Admin UI And Operator Workflow

Status: `in_progress`

Scope:

- admin page for incidents
- admin page for SRE Smart Bot policy/rules
- approval inbox
- incident timeline with evidence and action history

Exit criteria:

- operator can inspect incident evidence and approve/reject actions from UI
- operator-defined rules can be added without code changes
- built-in rules remain protected from unsafe mutation

Progress now:

- incidents workspace, approvals inbox, settings page, and detector-rules review page all exist
- the incident drawer now has tabbed summary / AI workspace / signals / actions views
- the AI workspace renders structured HTTP golden-signal MCP output instead of raw JSON

Next for Phase 3:

- add trend/history visuals driven by persisted `http_signals.history`
- add direct queue/backlog signal summaries now that async-worker backlog sources are landing

## Phase 4: Provider-Based Operator Channels

Status: `planned`

Scope:

- provider-based channel integration
- incident summaries and action prompts
- approval/reject commands
- thread-safe mapping between provider messages and incident/action IDs

Exit criteria:

- operator receives incident updates through a configured provider
- operator can approve a remediation through a provider that supports interaction
- provider delivery failures fall back to in-app notifications cleanly

## Phase 4.5: Loki And Alloy Ingestion Baseline

Status: `planned`

Scope:

- Alloy collection for pod logs and Kubernetes events
- Loki monolithic deployment profile
- bounded retention and low-cardinality labeling
- detector-friendly namespace selection and labels

Exit criteria:

- logs are queryable for recent incident windows without overloading the cluster
- Loki/Alloy footprint fits the current small-cluster budget
- detector services can query Loki instead of tailing raw node logs directly

## Phase 5: Log Intelligence And NATS Findings

Status: `planned`

Scope:

- lightweight detector service
- NATS subject for normalized findings
- log signature detection
- correlation with metrics snapshots and runtime health
- reuse the new SRE event-bus contracts so detector and ledger flows speak the same language

Exit criteria:

- log-derived findings enrich incidents instead of bypassing policy
- findings are normalized and replayable
- remediation remains gated by corroborating evidence

## Phase 5.5: Golden Signals And Metric Correlation

Status: `in_progress`

Scope:

- derive signal findings from cluster and service metrics
- normalize the common golden signal categories:
  - latency
  - traffic
  - errors
  - saturation
- correlate metric findings with logs, runtime health, and incident evidence
- keep the first slice lightweight by building on the existing metrics snapshot ingester

Current first slice:

- node CPU saturation detection from cluster snapshots
- node memory saturation detection from cluster snapshots
- pod restart pressure detection from pod status snapshots
- pod eviction pressure detection from pod status snapshots
- app-level 5xx burst detection from backend access logs
- app-level panic detection from backend logs
- app-level request volume windows from HTTP middleware
- app-level server-error rate windows from HTTP middleware
- app-level average latency windows from HTTP middleware
- read-only MCP access to recent HTTP golden-signal windows
- recommendation-only action proposal:
  - `review_cluster_capacity`
  - `review_workload_stability`

Immediate next slice:

- expose those windows directly in the incident summary UX
- add queue-depth / backlog golden signals for asynchronous workloads
- compare persisted HTTP trends with recent logs when drafting hypotheses

Exit criteria:

- metric-backed findings create or enrich incident threads without requiring log signatures
- saturation findings carry evidence snapshots and bounded recommendation actions
- thresholds are configurable through env and later promotable into policy/admin settings
- the agent/MCP layer can eventually inspect metric trends the same way it inspects logs

## Phase 6: MCP And Agent Runtime

Status: `planned`

Scope:

- MCP tool interfaces for Kubernetes, OCI, database, release state, and chat
- agent explanation layer
- bounded investigation flows

Exit criteria:

- agent can summarize incidents and evidence through MCP tools
- policy layer remains the final action authority
- human approvals remain explicit and auditable

## Phase 7: AI Operator Experience

Status: `planned`

Scope:

- AI-generated operator summaries
- suggested next investigative steps
- configurable provider delivery contracts
- standalone SRE Smart Bot worker/service extraction

Exit criteria:

- operators receive useful summaries grounded in stored findings/evidence
- provider delivery remains configurable through API contracts
- standalone extraction does not require changing the incident or approval model

## Next Build Slice

Recommended next implementation sequence:

1. add explicit observability/intelligence epics to backlog and handover
2. publish normalized SRE ledger events on the existing event bus so NATS/detector consumers have a stable contract
3. define and land the first detector/NATS subject contract using those event types
4. add Loki/Alloy deployment manifests or chart values sized for the small OKE cluster
5. then move into MCP tool contracts and AI operator features
## MCP And AI Feature Layer

- `In progress`: `robot_sre_policy` now includes MCP server definitions and bounded agent-runtime controls.
- `In progress`: incident workspace API/UI now exposes an MCP/AI-ready bundle with executive summaries, recommended questions, enabled MCP servers, and tooling guidance.
- `In progress`: concrete read-only MCP adapters now exist for:
  - `observability`
  - `kubernetes`
- `In progress`: read-only tool coverage now includes:
  - `logs.recent`
  - `release_drift.summary`
- `In progress`: log intelligence now also covers notification-delivery failure signatures from worker logs so SRE Smart Bot can open incidents for downstream async action failures.
- `In progress`: detector-rule learning loop now exists in the backend:
  - observed incidents can generate persisted detector-rule suggestions
  - admins can accept or reject suggestions
  - accepted suggestions become active `detector_rules` in `robot_sre_policy`
  - `detector_learning_mode=training_auto_create` can auto-activate learned rules
- `In progress`: the first deterministic agent workflow now exists:
  - build a draft hypothesis set
  - build a bounded investigation plan
  - use only read-only MCP tools
- `In progress`: an optional local-model interpretation layer now exists for:
  - `provider=ollama`
  - local model evaluation on top of the deterministic draft
- `Current default local profile`:
  - `provider=ollama`
  - `base_url=http://127.0.0.1:11434`
  - `model=llama3.2:3b`
- `In progress`: Helm/runtime default wiring now supports env-driven agent runtime defaults:
  - `IF_SRE_AGENT_RUNTIME_BASE_URL`
  - `IF_SRE_AGENT_RUNTIME_MODEL`
  - in-cluster `ollama.enabled=true` deployments can default to the internal service URL automatically
- `Done`: bootstrap/reset can now optionally persist those deployment-aware defaults into the saved global `robot_sre_policy` on first run through `bootstrap.seedRobotSREPolicyDefaults=true`, without overwriting later operator edits
- `Next`: add richer tool coverage, move tool invocation behind a standalone agent runtime seam, and keep mutating actions approval-bound.
- `Next`: add admin UI for detector-rule suggestions and training-mode controls.
