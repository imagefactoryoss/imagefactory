# SRE Smart Bot Requirements And Design

## Purpose

Define a production-minded but demo-friendly "SRE Smart Bot" capability for Image Factory that can:

- watch cluster and application health continuously
- explain incidents in operator-friendly language
- propose or take bounded remediation actions
- notify and converse with operators over chat channels such as Telegram or WhatsApp
- use an AI agent runtime and MCP tools without giving up deterministic control over high-risk actions

This document is intentionally focused on a practical first version that fits the current Image Factory architecture and recent OKE operational failure modes.

## Problem Statement

Recent cluster instability exposed several gaps:

- node `ephemeral-storage` pressure built up before operators had useful early warning
- Docker Hub fallback and image churn amplified recovery pain
- some remediation steps were repetitive and mechanical
- diagnosis required stitching together OCI, Kubernetes, and application state manually
- stale configuration persisted across infrastructure changes

We want an ops capability that behaves like a careful teammate:

- detects trouble early
- narrates what is happening
- recommends safe actions first
- can execute approved actions
- leaves an auditable trail

## Naming

The product-facing name should be `SRE Smart Bot`.

For now, some internal code and document references may still use `Robot SRE` as a technical codename while the implementation is being rolled out.

## Product Vision

SRE Smart Bot is not just a chatbot and not just a cleanup CronJob.

It is a hybrid system with two layers:

1. Deterministic control plane
- watches defined signals
- evaluates explicit policies
- executes allowlisted remediation actions
- records evidence, decisions, cooldowns, and outcomes

2. Conversational operator interface
- translates system state into concise incident updates
- answers operator questions
- asks for approval when actions cross risk thresholds
- uses chat channels and Image Factory admin surfaces as the human interface

The AI persona should improve usability and investigation speed.
The deterministic policy layer should remain the source of truth for what the system is allowed to do.

## Taxonomy Shape

The Robot SRE should organize incidents by operational domain first, then by incident type.

Recommended top-level domains:

- infrastructure
- runtime services
- application services
- network / ingress
- identity / security
- release / configuration
- operator channels

This gives us a cleaner way to expand beyond cluster-only issues and lets operators browse and reason about rules in a way that matches how they already think about incidents.

## Goals

- Detect cluster, runtime dependency, and application degradation before it becomes customer-visible.
- Create a single incident narrative from Kubernetes, OCI, and Image Factory runtime signals.
- Remediate a bounded set of known-safe failure classes automatically.
- Escalate safely for actions with non-obvious blast radius.
- Give operators a chat-first interface for incident follow-up, approvals, and status.
- Reuse existing Image Factory runtime health watcher patterns where possible.
- Keep the design compatible with OKE, Supabase, ingress-nginx, Tekton, and mirrored GitLab images.

## Non-Goals

- Fully autonomous root access with unrestricted shell/tool execution.
- Generic "AI runs the cluster" behavior.
- Replacing Prometheus, Grafana, or normal alerting entirely.
- Performing destructive actions without policy, cooldowns, or approval.
- Solving every SRE use case in v1.

## Key Design Principles

### Modular System Boundaries

SRE Smart Bot should be implemented as a modular subsystem, not as a long chain of special cases inside server startup.

That means:

- startup code composes dependencies and launches workers
- signal adapters translate watcher output into normalized findings
- incident services manage correlation and lifecycle
- policy services decide whether actions are allowed
- channel adapters deliver notifications and approvals through provider contracts

This keeps the system testable, easier to extend, and less likely to turn `main.go` into an unmaintainable control tower.

### Deterministic Before Generative

The system should use explicit policy evaluation for:

- detection
- severity classification
- remediation eligibility
- approvals
- cooldown enforcement
- audit logging

The AI layer should summarize, explain, and help choose between approved actions.

### Safe By Default

The robot should prefer:

- observe
- notify
- suggest
- ask approval
- take low-risk action

It should not jump straight to reboot, replace, delete, or suspend critical services.

### Explainability

Every decision should produce:

- what triggered the action
- what evidence was used
- why this action was chosen
- what was changed
- what happens next

### Persona With Boundaries

The Robot SRE can feel like a teammate, but it must be honest about uncertainty and clearly distinguish:

- observations
- inferences
- recommendations
- executed actions
- required human approvals

## Current Platform Hooks We Can Reuse

The backend already has a useful foundation:

- `runtime_dependency_watcher` in [`backend/cmd/server/main.go`](../../backend/cmd/server/main.go)
- process health store and runtime component status
- notification delivery and websocket updates
- tenant asset drift watcher
- provider readiness watcher
- quarantine/release compliance watcher
- system configuration storage in `system_configs`

This suggests we should not start with a separate standalone bot.
The better path is to add a new remediation/orchestration capability that integrates with the backend runtime and admin APIs.

## Proposed Capability Scope

### V1: Guarded Runtime And Cluster Remediation

The first version should handle a small number of repeatable operational problems:

- runtime dependency unavailable
- ingress broken or DNS/cert mismatch
- stale LDAP/system config after service replacement
- node disk pressure
- repeated image pull failures
- failed or noisy background job buildup
- Tekton or controller churn overwhelming small clusters

V1 should also include a small set of application-service incidents, not only infrastructure incidents, for example:

- backend deployment degraded
- frontend deployment degraded
- dispatcher unavailable
- login error spike
- worker crash-loop with dependency correlation

### V1.5: Operator Conversation

Expose incident updates and approvals over configurable operator channels:

- Image Factory admin notifications / websocket feed
- enterprise webhook or chat gateway integrations
- optionally Telegram for environments that allow it

Support commands like:

- `status`
- `what happened`
- `why did you scale this down`
- `show evidence`
- `approve remediation <id>`
- `pause robot`
- `resume robot`

### V2: Richer Tool Use Via MCP And Agent Runtime

Add:

- MCP-backed tool registry
- agent planning for investigations
- richer runbooks
- multi-step incidents with memory and threads
- provider-based channel integrations through API contract

## Target Users

- system administrators
- demo environment owner/operator
- platform engineer
- on-call engineer

## Functional Requirements

## Operator Channel Contract

Many enterprise environments will not allow Telegram or WhatsApp directly.

Because of that, channels should not be hardcoded into the product design.
Instead, SRE Smart Bot should treat channels as configurable providers exposed through an API contract.

Recommended shape:

- provider id
- provider kind
- display name
- enabled flag
- approval interaction support
- config reference or endpoint reference

Examples of provider kinds:

- `in_app`
- `email`
- `webhook`
- `slack`
- `teams`
- `telegram`
- `whatsapp`
- `custom`

This makes the product usable in locked-down enterprises where the integration path may be:

- an internal notification broker
- a Teams bot
- a ServiceNow or PagerDuty bridge
- a company webhook gateway

### Detection

The robot must watch at least these signal classes:

- Kubernetes node conditions
- Kubernetes events
- pod restart loops
- image pull failures
- ingress health and certificate coverage
- runtime dependency health
- application process health
- database reachability
- cluster capacity trends
- OCI instance/node pool state

#### Example Signal Inputs

- `NodeDiskPressure=True`
- `FreeDiskSpaceFailed`
- `EvictionThresholdMet`
- `ImagePullBackOff`
- `ErrImagePull`
- repeated `CrashLoopBackOff`
- `Ready=Unknown`
- ingress with missing address
- TLS secret/host mismatch
- Supabase connectivity failures
- LDAP login failures above threshold

### Incident Correlation

The robot must group related low-level signals into a higher-level incident such as:

- `node_disk_pressure`
- `registry_auth_or_mirror_failure`
- `runtime_dependency_outage`
- `ingress_configuration_drift`
- `identity_provider_unreachable`
- `database_connectivity_degraded`

### Recommendation Generation

For each incident class, the robot should be able to produce:

- summary
- likely root cause
- blast radius
- confidence
- recommended next step
- allowed automated actions
- actions requiring approval

### Remediation Execution

The robot should support staged remediation plans.

#### Example stage model

1. Observe
- collect evidence
- classify severity
- notify operator

2. Contain
- reduce churn
- pause noisy jobs
- scale nonessential services down

3. Recover
- restart a component
- rotate a stale config
- reapply a release
- replace or recycle one node

4. Verify
- confirm health recovery
- emit incident resolution update

### Operator Messaging

The robot must be able to:

- send proactive alerts
- answer follow-up questions in-thread
- request approval for gated actions
- post remediation summaries
- expose links to evidence or admin pages where applicable

### Auditability

Every investigation and remediation action must be recorded with:

- incident id
- timestamps
- evidence summary
- actor
- action type
- policy basis
- approval source if any
- outcome

## Non-Functional Requirements

- Low false-positive rate for auto-remediation.
- No destructive remediation without policy and approval.
- Works in small demo clusters with limited resources.
- Degrades gracefully if the AI subsystem is unavailable.
- Keeps secrets out of chat transcripts.
- Supports idempotent retries.
- Supports dry-run mode.

## Architecture Proposal

## High-Level Components

1. Signal Collectors
- Kubernetes collector
- OCI collector
- Image Factory runtime collector
- database/system-config collector

2. Incident Engine
- normalizes signals
- correlates incidents
- scores severity
- opens/closes incidents

3. Policy Engine
- maps incident class + severity + environment to allowed actions
- enforces cooldowns and approvals
 - resolves built-in policy plus operator-defined rule overrides

4. Remediation Executor
- performs allowlisted actions
- tracks action results
- rolls forward or stops

5. Agent Orchestrator
- creates operator-facing narrative
- can investigate through tools
- never bypasses policy engine

6. Chat Gateway
- Telegram integration
- optional WhatsApp integration
- future Slack/email/webhook support

7. Evidence And Audit Store
- incident records
- tool traces
- action journal
- chat thread linkage
 - rule versions and policy decisions

8. Rules And Policy Admin Layer
- stores built-in and custom rules
- supports environment-specific overrides
- validates operator-defined rule safety
- previews effective policy

## Recommended Implementation Shape

### Phase 1 Recommendation

Implement Robot SRE as a backend subsystem inside the existing server process, or as a closely related worker service.

Why:

- existing watchers and health store already live in backend
- existing notifications and websocket pathways already exist
- app-level configs and runtime signals are already available there
- simplest way to create admin APIs and UI for incidents

Within that backend subsystem, keep three clear separations:

- built-in taxonomy and hard safety constraints in code
- operator-defined rules in persisted config/data
- conversational agent orchestration as a consumer of incident/policy state

### Phase 2 Recommendation

Split out a dedicated `ops-agent` service when:

- tool execution volume grows
- remediation policies become complex
- chat integrations need independent scaling
- operator interactions require long-lived workflows

## MCP + Agent Runtime Design

## Why MCP Fits

MCP is useful here because the robot needs structured, permissioned access to tools such as:

- Kubernetes read and write actions
- OCI instance/node operations
- Helm operations
- Supabase or Postgres inspection
- Image Factory admin APIs
- notification and chat senders

Instead of embedding all integrations directly into prompts, MCP gives us:

- explicit tool contracts
- better permission boundaries
- server-side tool reuse
- testability

## Proposed MCP Servers

### `if-ops-kubernetes`

Capabilities:

- read nodes, pods, events, ingress, deployments
- allowed write operations for allowlisted namespaces/resources
- scale deployments
- suspend/resume CronJobs
- delete completed/failed pods
- cordon/drain selected nodes

### `if-ops-oci`

Capabilities:

- inspect node pools
- inspect instances
- reboot worker
- replace worker
- query boot volume size

### `if-ops-database`

Capabilities:

- read system config
- update selected config keys through validated operations
- inspect health metadata

### `if-ops-release`

Capabilities:

- inspect Helm release values/status
- apply approved release reconciliations

### `if-ops-chat`

Capabilities:

- send Telegram/WhatsApp messages
- create approval prompts
- thread replies to incidents

### `if-ops-observability`

Capabilities:

- query app runtime health
- query watcher health
- query incident and remediation history

## Agent SDK Role

The agent runtime should be used for:

- evidence gathering from MCP servers
- summarization
- hypothesis ranking
- operator conversation
- action plan drafting

The agent runtime should not decide action legality by itself.
That must stay in policy code.

## Policy And Safety Model

## Action Classes

### Auto-Allowed

- send notification
- open incident
- collect evidence
- mark component degraded
- delete completed/failed pods
- suspend specific noisy CronJobs
- scale down specific noncritical demo workloads

### Approval Required

- scale down shared controllers
- restart backend/frontend
- patch system config
- reconcile Helm release
- cordon or drain a node
- reboot an OCI worker

### Human Only

- database-destructive actions
- credential rotation without workflow
- deleting tenant data
- disabling security controls
- bulk namespace deletion

## Operator-Defined Rules

Operators should be able to extend the robot from the admin UI without code changes.

Recommended supported customizations:

- incident threshold overrides
- severity escalation rules
- environment-specific enable/disable
- notification routing
- suppression windows
- cooldown tuning
- allowlisted resource selection within approved action families

Recommended restrictions:

- operators cannot make destructive actions auto-allowed
- operators cannot introduce arbitrary shell commands
- operators cannot bypass approval for disruptive classes
- operators cannot store raw secrets in rule definitions

This makes the system flexible without turning it into an unsafe "run anything" automation tool.

## Required Safeguards

- allowlist of resources and actions
- per-incident cooldown
- per-action rate limits
- environment awareness (`demo`, `staging`, `prod`)
- approval tokens with expiry
- dry-run mode
- audit log with before/after evidence
- rollback or fallback instructions for every action class

## Channel Strategy

## Telegram For MVP

Telegram is the best first chat channel because:

- easier bot onboarding
- lower business/legal friction
- supports threaded-ish operator flows well enough
- good for approval and incident updates

## WhatsApp For Phase 2

WhatsApp is attractive for operator reach, but it introduces more platform overhead:

- Meta business onboarding
- template/message policy constraints
- channel approval and delivery complexity

Recommendation:

- build a channel abstraction first
- ship Telegram first
- add WhatsApp once workflows and policy prompts are stable

## Example Operator Experience

Example alert:

`Robot SRE: Disk pressure detected on worker 10.0.10.202. Evidence: EvictionThresholdMet, FreeDiskSpaceFailed, 12 pod evictions in 9m. Suggested actions: 1) pause nonessential workloads 2) clean completed/failed pods 3) reboot node if pressure persists for 15m. Reply APPROVE 1 to proceed.`

Example follow-up:

`Why did this happen?`

Example response:

`The node is low on ephemeral storage. Primary contributors appear to be image churn, repeated pod restarts, and unreclaimed runtime artifacts. Confidence medium.`

## Incident Classes And V1 Remediations

## 1. Node Disk Pressure

Signals:

- node `DiskPressure=True`
- `FreeDiskSpaceFailed`
- evictions

Automations:

- notify
- delete completed/failed pods
- suspend noisy jobs
- scale down allowlisted demo workloads
- if still degraded beyond threshold, request approval to recycle node

## 2. Registry / Image Pull Degradation

Signals:

- `ImagePullBackOff`
- `ErrImagePull`
- auth failures
- rate limit signals

Automations:

- identify image source
- classify `dockerhub_rate_limit` vs `gitlab_auth` vs `tag_missing`
- suggest mirror or pull-secret remediation
- if live drift exists, reconcile release or image refs with approval

## 3. Runtime Dependency Failure

Signals:

- NATS/Redis/MinIO/registry/glauth health failures
- backend logs showing dependency timeout

Automations:

- verify service, endpoints, and health
- restart dependency if allowed
- verify dependent services recover

## 4. Config Drift

Signals:

- ingress missing expected hosts
- TLS secret mismatch
- LDAP host points to old service IP
- runtime config disagrees with Helm values

Automations:

- detect mismatch
- suggest or apply targeted config correction
- reopen incident if drift reappears after release change

## Data Model Proposal

Add new persisted entities:

- `ops_incidents`
- `ops_incident_events`
- `ops_remediation_actions`
- `ops_approvals`
- `ops_channel_threads`
- `ops_policy_bindings`

Suggested incident fields:

- `id`
- `incident_type`
- `status`
- `severity`
- `summary`
- `evidence_json`
- `current_signature`
- `opened_at`
- `resolved_at`
- `environment`
- `tenant_scope` if relevant
- `resource_scope`

## Admin UX Proposal

Add an admin area such as:

- `Operations > Robot SRE`

Core views:

- active incidents
- incident detail with evidence timeline
- pending approvals
- action history
- policy configuration
- channel configuration
- simulation / dry-run console

## Suggested Rollout Plan

## Phase 0: Design And Safety

- define incident taxonomy
- define action allowlist
- define approval rules
- create data model and APIs
- define operator-defined rule boundaries and validation

## Phase 1: Deterministic Watcher

- implement incident engine
- implement first remediations for disk pressure and runtime dependency failure
- add audit log
- add admin UI for incident visibility
 - add rules UI for threshold/routing overrides

## Phase 2: Telegram Channel

- add outbound incident notifications
- add approval workflow
- add simple command handlers

## Phase 3: Agent + MCP Integration

- expose MCP servers for approved tools
- add agent summarization and investigation mode
- add evidence-aware operator Q&A

## Phase 4: Expanded Remediation

- OCI node recycle
- Helm drift reconciliation
- config drift correction
- richer release recovery workflows

## Open Questions

- Should Robot SRE live in the backend process or its own deployable from day one?
- Should approvals be required in demo environments for node recycle, or only in staging/prod?
- Should Telegram be the only MVP chat channel?
- How much authority should the agent have to propose Helm changes vs execute them?
- Do we want the robot to reason over historical incidents for pattern detection in v1, or wait for v2?
 - How much of the application-service taxonomy should ship in v1 vs v1.5?
 - Should operator-defined rules be stored in `system_configs` first, or in dedicated ops tables from day one?

## Recommended MVP Decisions

- Start with a deterministic remediation engine inside the backend runtime.
- Reuse the existing watcher/process health patterns already present in `backend/cmd/server/main.go`.
- Use MCP for tool integration boundaries, not as the policy engine.
- Use an agent runtime for operator conversation and evidence summarization only.
- Ship Telegram first.
- Treat WhatsApp as a later channel adapter.
- Restrict v1 automatic actions to low-risk containment and cleanup.
- Require explicit operator approval for node recycle, release reconciliation, and config mutation.

## Immediate Next Deliverables

1. Incident taxonomy and policy matrix.
2. Data model and API design for incidents, actions, approvals, and operator-defined rules.
3. MCP server interface definitions for Kubernetes, OCI, database, and chat.
4. Telegram bot interaction design.
5. V1 implementation plan for `node_disk_pressure`, `runtime_dependency_failure`, and first application-service incidents.
