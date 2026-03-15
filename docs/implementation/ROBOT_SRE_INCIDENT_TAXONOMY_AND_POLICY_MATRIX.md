# Robot SRE Incident Taxonomy And Policy Matrix

## Purpose

Define the operational incident classes, evidence rules, severity model, and remediation policy boundaries for the Robot SRE / Ops Persona.

This document is the safety contract for the system.
The AI layer may explain, summarize, and help choose actions, but it must stay inside the policy boundaries defined here.

## Design Intent

This taxonomy is optimized for:

- small OKE clusters
- demo and staging environments first
- recent real failure modes in Image Factory
- gradual automation with strong guardrails

It is not intended to be exhaustive on day one.
It should start narrow and expand only when each class has good evidence, clear rollback posture, and proven low false-positive behavior.

## Domain Categories

The taxonomy should be organized first by operational domain, then by incident class.
This keeps the system easier to reason about, easier to extend, and friendlier in the admin UI.

### `infrastructure`

Cluster and cloud substrate concerns:

- nodes
- capacity
- storage
- OCI worker lifecycle
- cluster scheduling

### `runtime_services`

Shared in-cluster dependencies and control-plane helpers:

- Redis
- NATS
- MinIO
- internal registry
- GLAuth
- background workers and watcher processes

### `application_services`

Image Factory user-facing and business-critical services:

- backend API
- frontend UI
- docs service
- dispatcher
- notification worker
- email worker
- external tenant service

### `network_ingress`

Traffic routing and reachability:

- ingress
- DNS
- TLS
- load balancer
- service-to-service resolution when customer-visible

### `golden_signals`

Cross-cutting service health and capacity indicators:

- latency
- traffic
- errors
- saturation
- queue backlog and throughput when they behave like service health signals

### `identity_security`

Authentication, authorization, and trust path concerns:

- LDAP / identity provider connectivity
- auth-provider drift
- secret and certificate mismatches
- security control disablement or auth-path failures

### `release_configuration`

Intended vs actual system shape:

- Helm drift
- stale config in system tables
- image source drift
- missing pull secrets

### `operator_channels`

How the robot reaches humans:

- Telegram delivery
- WhatsApp delivery
- in-app admin notifications
- approval channel reachability

## Incident Lifecycle

Each incident moves through these states:

1. `observed`
2. `triaged`
3. `contained`
4. `recovering`
5. `resolved`
6. `suppressed`
7. `escalated`

## Severity Model

### `info`

- no customer-visible impact
- advisory only
- no automated write action needed

### `warning`

- localized degradation
- blast radius is narrow or slow-moving
- low-risk containment may be automatic

### `critical`

- customer-visible or imminent outage
- core dependency unavailable
- data loss, auth failure, or control-plane instability risk
- severe golden-signal exhaustion such as sustained saturation, runaway error rate, or dramatic latency collapse

## Environment Modes

Policy must vary by environment:

### `demo`

- more automation allowed
- faster containment acceptable
- human approval still required for destructive actions

### `staging`

- moderate automation allowed
- prefer approval for anything beyond low-risk cleanup

### `production`

- conservative mode
- notify and recommend first
- auto-remediation restricted to clearly safe idempotent actions

## Action Classes

### `observe`

Read-only evidence collection:

- query Kubernetes
- query OCI
- query app runtime health
- query release state
- query system config

### `notify`

- open incident
- send chat alert
- send in-app/admin notification
- update incident thread

### `contain`

Low-risk actions to reduce churn or blast radius:

- delete completed/failed pods
- suspend allowlisted CronJobs
- scale allowlisted noncritical workloads to zero
- mark incident suppressed for cooldown

### `recover`

Actions that alter service topology or runtime behavior:

- rollout restart deployment
- patch targeted system config
- reconcile Helm release
- resume paused jobs/workloads
- cordon one node

### `disruptive`

Higher-risk recovery actions:

- drain node
- reboot worker
- replace worker
- scale shared controllers
- disable subsystems

## Approval Policy

### Auto-Allowed

- all `observe`
- all `notify`
- low-risk `contain` actions on explicitly allowlisted resources

### Approval Required

- any `recover`
- any `disruptive`
- any config mutation
- any Helm reconciliation
- any OCI instance or node-pool operation

### Human Only

- delete persistent data
- rotate secrets without runbook
- remove namespaces with tenant data
- disable auth or security controls

## Cooldown Policy

Unless overridden per incident class:

- observation polling: 60s
- duplicate alert suppression: 15m
- same remediation action retry: 15m
- disruptive action retry: 60m

## Evidence Confidence Bands

### `high`

- multiple corroborating signals
- direct error from failing component
- repeated signal across two or more checks

### `medium`

- one strong signal plus one weak signal
- inferred root cause but not directly proven

### `low`

- single ambiguous signal
- no corroborating data

The robot may auto-act only on `high` confidence incidents in auto-allowed categories.

## Incident Taxonomy

## Taxonomy Structure

Each incident should be represented as:

- `domain`
- `incident_type`
- `display_name`
- `description`
- `default_severity`
- `evidence_rules`
- `policy_binding`

Example:

- `domain`: `runtime_services`
- `incident_type`: `runtime_dependency_outage`
- `display_name`: `Runtime Dependency Outage`

## Domain: `infrastructure`

## 1. `node_disk_pressure`

### Description

Node ephemeral storage pressure causing evictions, scheduling failures, and runtime instability.

### Primary Signals

- Kubernetes node condition `DiskPressure=True`
- node taint `node.kubernetes.io/disk-pressure`
- events:
  - `EvictionThresholdMet`
  - `FreeDiskSpaceFailed`
- burst of evicted pods

### Secondary Signals

- repeated image pulls
- large backlog of completed/failed pods
- image-pull retries after churn

### Severity Rules

- `warning`
  - one node in pressure for <10m and cluster remains functional
- `critical`
  - multiple nodes in pressure
  - or one node in pressure with customer-facing pod failures
  - or pressure persists >10m

### Root Cause Hypotheses

- image churn
- failed runtime garbage collection
- noisy CronJobs / Tekton backlog
- pod log buildup
- hostPath growth

### Auto-Allowed Actions

- notify operator
- collect top evidence
- delete `Succeeded` and `Failed` pods
- suspend allowlisted noisy CronJobs
- scale allowlisted demo workloads down

### Approval-Required Actions

- cordon node
- reboot worker
- replace worker
- scale shared controllers down

### Cooldown

- same containment set once per 15m
- reboot/replace once per 60m per node

### Rollback / Exit Criteria

- `DiskPressure=False`
- no disk-pressure taint
- no new evictions for 10m

## 1.1 `node_unreachable_or_notready`

### Description

Worker becomes unreachable, `NotReady`, or stops reporting heartbeats.

### Primary Signals

- node `Ready=False` or `Ready=Unknown`
- `node.kubernetes.io/unreachable`
- OCI instance state mismatch with Kubernetes node state

### Auto-Allowed Actions

- notify
- collect node, kubelet, and OCI evidence

### Approval-Required Actions

- cordon
- reboot worker
- replace worker

## Domain: `release_configuration`

## 2. `registry_pull_failure`

### Description

Pods fail to pull images due to registry auth issues, missing tags, or external registry rate limits.

### Primary Signals

- `ErrImagePull`
- `ImagePullBackOff`
- error text containing:
  - `toomanyrequests`
  - `unauthorized`
  - `manifest unknown`
  - `403 Forbidden`

### Secondary Signals

- runtime dependency outage
- rollout stuck
- fresh replacement nodes unable to hydrate

### Severity Rules

- `warning`
  - one noncritical workload affected
- `critical`
  - runtime dependency image failing
  - or multiple workloads affected
  - or recovery blocked

### Auto-Allowed Actions

- classify failure subtype
- notify with exact repo/tag/error
- recommend mirror or pull-secret repair

### Approval-Required Actions

- patch live deployment image
- patch service account pull secret
- reconcile release to GitLab source of truth

### Cooldown

- no repeated auto recommendation alert inside 15m unless signature changes

### Exit Criteria

- deployment available
- no new pull failures for 10m

## 2.1 `configuration_drift`

### Description

Persisted or live config disagrees with expected release intent.

### Primary Signals

- system config points to stale service IP
- release values and live objects disagree
- app env/config and persisted system config disagree

### Auto-Allowed Actions

- notify
- diff desired vs actual config

### Approval-Required Actions

- patch targeted system config
- reconcile release

## Domain: `runtime_services`

## 3. `runtime_dependency_outage`

### Description

A required dependency like Redis, NATS, MinIO, registry, GLAuth, or internal worker health is unavailable.

### Primary Signals

- service unavailable
- health endpoint fails
- backend log shows dependency connection failure
- process health store marks component down

### Severity Rules

- `warning`
  - degraded optional dependency
- `critical`
  - required dependency down
  - auth/login blocked
  - build execution blocked

### Auto-Allowed Actions

- notify
- verify service/endpoints/pod health
- restart allowlisted dependency deployment in demo/staging only if confidence is high

### Approval-Required Actions

- restart shared dependency in production
- patch runtime config
- reconcile release

### Exit Criteria

- dependency health passes
- dependent workloads recover

## 3.1 `background_watcher_degraded`

### Description

One of Image Factory's internal background watchers or process-health components stops running or reports degraded state.

### Primary Signals

- process health store marks watcher disabled or not running
- watcher counters stop advancing
- repeated watcher-specific failures

### Auto-Allowed Actions

- notify
- collect runtime health evidence

### Approval-Required Actions

- rollout restart backend or worker that owns the watcher
- patch runtime-services config

## Domain: `network_ingress`

## 4. `ingress_configuration_drift`

### Description

Ingress resources or TLS coverage drift from expected hostnames, classes, or endpoints.

### Primary Signals

- expected ingress missing
- ingress has no address after grace period
- TLS secret missing or SAN mismatch
- host-header test fails while services are healthy

### Severity Rules

- `warning`
  - docs or secondary host broken
- `critical`
  - primary UI/API host unavailable

### Auto-Allowed Actions

- notify
- gather ingress/controller/service evidence

### Approval-Required Actions

- enable ingress
- patch host list / TLS host coverage
- reconcile release

### Exit Criteria

- ingress admitted
- address assigned
- host-header HTTPS checks succeed

## 4.1 `dns_or_lb_mismatch`

### Description

DNS or external load balancer target does not match the currently active ingress controller.

### Primary Signals

- host resolves to unexpected IP
- ingress controller service IP differs from expected public record
- host-header test succeeds against load balancer IP but public hostname fails

### Auto-Allowed Actions

- notify
- present expected DNS target and evidence

### Approval-Required Actions

- none inside cluster

### Human Only

- external DNS record changes

## Domain: `identity_security`

## 5. `identity_provider_unreachable`

### Description

LDAP or other auth provider reachable in configuration but unavailable or misconfigured.

### Primary Signals

- login attempts timeout/fail above threshold
- backend log shows LDAP connect/bind/search errors
- stored provider host points to stale IP or dead endpoint

### Severity Rules

- `warning`
  - auth method degraded but local admin login still works
- `critical`
  - primary login path broken for operators or demo users

### Auto-Allowed Actions

- notify
- compare stored system config with live service DNS
- verify TCP reachability and bind/search behavior

### Approval-Required Actions

- patch persisted LDAP system config
- restart backend to pick up auth env defaults

### Exit Criteria

- login succeeds within SLO
- no new LDAP timeout events for 10m

## 5.1 `authn_or_authz_regression`

### Description

Core authentication or authorization paths fail even if the identity provider itself is reachable.

### Primary Signals

- spike in login failures
- JWT validation failures
- RBAC checks failing unexpectedly
- admin-only flows inaccessible

### Auto-Allowed Actions

- notify
- gather auth logs and config evidence

### Approval-Required Actions

- restart backend
- revert or patch auth-related config

## Domain: `application_services`

## 6. `application_service_degraded`

### Description

A user-facing or business-critical Image Factory service is crash-looping, unavailable, or serving errors.

### Candidate Services

- backend
- frontend
- docs
- dispatcher
- notification worker
- email worker
- external tenant service

### Primary Signals

- `CrashLoopBackOff`
- deployment unavailable
- readiness failures
- repeated 5xx responses
- worker health endpoint unavailable

### Severity Rules

- `warning`
  - secondary service degraded with narrow impact
- `critical`
  - backend, frontend, or dispatcher degraded
  - or repeated failures across multiple services

### Auto-Allowed Actions

- notify
- gather logs, pod events, and dependency health

### Approval-Required Actions

- rollout restart targeted service
- scale service temporarily
- reconcile release for affected service

### Exit Criteria

- deployment available
- no new crash loops for 10m

## 6.1 `application_error_rate_spike`

### Description

User-visible API/UI errors rise without an obvious pod-level failure.

### Primary Signals

- sustained 5xx rate
- login or core workflow failures above threshold
- health endpoints remain green while transaction-level errors rise

### Auto-Allowed Actions

- notify
- gather recent logs and runtime metrics

### Approval-Required Actions

- restart targeted service
- activate containment feature flags if defined

## Domain: `database_connectivity`

## 7. `database_connectivity_degraded`

### Description

The application cannot reliably reach the configured database.

### Primary Signals

- DB ping failure from runtime dependency watcher
- app startup/connect errors
- migration/bootstrap failures

### Severity Rules

- always `critical`

### Auto-Allowed Actions

- notify
- gather DB health evidence

### Approval-Required Actions

- switch configured DB target
- run migration/reconcile jobs
- patch system configs referencing DB

### Human-Only Actions

- data restore
- schema reset
- PVC deletion

## Domain: `release_configuration`

## 8. `release_drift_or_partial_apply`

### Description

Helm or runtime state partially diverges from intended release state.

### Primary Signals

- Helm `failed` or `pending-*`
- live images differ from desired values
- missing `imagePullSecrets` or stale field ownership

### Severity Rules

- `warning`
  - workloads healthy but metadata inconsistent
- `critical`
  - rollout blocked and workloads unhealthy

### Auto-Allowed Actions

- detect diff
- notify operator with exact drift

### Approval-Required Actions

- run Helm reconcile
- force conflict ownership
- restart impacted workloads

## Domain: `runtime_services`

## 9. `background_job_buildup`

### Description

Completed, failed, or noisy background workloads build up and degrade small-cluster stability.

### Primary Signals

- large counts of `Succeeded`/`Failed` pods
- repeating CronJobs with no user value
- Tekton history growth

### Severity Rules

- `warning`
  - backlog exceeds threshold but no node impact yet
- `critical`
  - backlog contributing to disk pressure or control-plane churn

### Auto-Allowed Actions

- delete completed/failed pods
- suspend allowlisted CronJobs
- post cleanup summary

### Approval-Required Actions

- pause shared controllers
- bulk cleanup outside allowlist

## Domain: `operator_channels`

## 10. `chatops_delivery_failure`

### Description

The robot cannot reliably reach operators through configured channels.

### Primary Signals

- Telegram/WhatsApp send failures
- repeated delivery retries

### Severity Rules

- `warning`
  - one channel unavailable
- `critical`
  - all operator channels unavailable during active incident

### Auto-Allowed Actions

- fail over to alternate channel
- surface alert in admin UI

## Extensible Rule Model

The taxonomy should support two rule types:

### Built-In Rules

Shipped by engineering and versioned in code:

- canonical incident types
- default evidence rules
- default policies
- default remediations

### Operator-Defined Rules

Created from the admin UI:

- additional signal thresholds
- custom environment-specific incident variants
- routing and notification rules
- approval requirements and overrides
- suppression windows

Operator-defined rules should extend built-in rules, not replace core safety constraints.

## Operator-Defined Rule Boundaries

Operators should be able to add or change:

- thresholds
- severity escalation conditions
- channel routing
- cooldown values
- enable/disable per incident type per environment
- allowlisted resources inside pre-approved action families

Operators should not be able to change from the UI:

- human-only action classes into auto actions
- destructive actions into auto-allowed
- secret values directly in incident rules
- unrestricted shell command definitions

## Suggested Rule Schema

Each rule should contain:

- `id`
- `name`
- `enabled`
- `domain`
- `incident_type`
- `environment_scope`
- `signal_selector`
- `threshold_expression`
- `severity`
- `notification_policy`
- `allowed_action_profile`
- `cooldown_seconds`
- `suppression_schedule`
- `owner`
- `version`

## Admin UI Requirements For Rules

Add an operator-facing rules interface under:

- `Operations > Robot SRE > Rules`

Minimum capabilities:

- list built-in and custom rules
- clone built-in rule into custom override
- enable/disable rule per environment
- edit thresholds and routing
- preview policy outcome
- test rule against recent evidence
- audit who changed what and when

## Policy Resolution Order

When evaluating a potential incident:

1. built-in taxonomy definition
2. built-in policy defaults
3. environment policy overlay
4. operator-defined rule override
5. hard safety constraints

Hard safety constraints must always win.

## Expanded V1 Recommendation

V1 should still implement a narrow set of incident classes, but the taxonomy should be structured by domain from the start so we can add:

- more infrastructure incidents
- application-service incidents
- security incidents
- operator-defined rules

without redesigning the system later.

## Policy Matrix

| Incident Class | Auto Observe | Auto Notify | Auto Contain | Approval Required | Human Only |
|---|---|---|---|---|---|
| `node_disk_pressure` | yes | yes | yes, allowlist only | cordon, reboot, replace, shared scale-down | persistent data deletion |
| `registry_pull_failure` | yes | yes | no | image patch, pull-secret patch, release reconcile | registry credential rotation outside runbook |
| `runtime_dependency_outage` | yes | yes | limited restart in demo/staging | shared dependency restart, config patch, release reconcile | none by default |
| `ingress_configuration_drift` | yes | yes | no | ingress enable/patch, TLS patch, release reconcile | certificate/key replacement outside runbook |
| `identity_provider_unreachable` | yes | yes | no | patch LDAP config, backend restart | auth disablement |
| `database_connectivity_degraded` | yes | yes | no | DB target patch, recovery workflow start | restore/reset/delete data |
| `release_drift_or_partial_apply` | yes | yes | no | Helm reconcile, restart workloads | uninstall/delete release resources |
| `background_job_buildup` | yes | yes | yes, allowlist only | shared controller pause, bulk cleanup | namespace/data deletion |
| `chatops_delivery_failure` | yes | yes | alternate-channel retry | channel credential/config changes | none |

## Allowlist Proposal For V1

### Auto-Contain Workloads

- `headlamp`
- `tekton-pipelines` deployments
- `tekton-pipelines-resolvers` deployments
- `image-factory` demo app deployments
- `trivy-db-warmup`

### Auto-Contain Actions

- delete `Succeeded` and `Failed` pods cluster-wide
- suspend `trivy-db-warmup`
- scale `image-factory` app workloads to zero in demo mode

### Never Auto-Contain

- Supabase config
- ingress controller
- cert-manager
- namespace deletion
- PVC deletion
- database reset

## Required Incident Evidence Schema

Each incident record should capture:

- `incident_type`
- `severity`
- `confidence`
- `signal_sources`
- `resource_scope`
- `environment`
- `evidence_summary`
- `raw_evidence_refs`
- `recommended_actions`
- `executed_actions`
- `approval_state`

## Approval Prompt Requirements

When the robot asks for approval, the prompt must include:

- action
- target
- why now
- expected impact
- rollback or next fallback
- expiry time

Example:

`Robot SRE requests approval: reboot worker 10.0.10.202. Reason: disk pressure persists after cleanup for 17m. Expected impact: pods on that node will reschedule. Rollback/fallback: replace worker from node pool if reboot does not clear pressure. Approval expires in 10m.`

## Metrics To Add

- `ops_incidents_open_total`
- `ops_incidents_resolved_total`
- `ops_incidents_by_type`
- `ops_auto_remediations_total`
- `ops_auto_remediations_failed_total`
- `ops_approvals_requested_total`
- `ops_approvals_granted_total`
- `ops_approvals_denied_total`
- `ops_policy_suppressions_total`
- `ops_chat_delivery_failures_total`

## MVP Recommendation

Implement first:

1. `node_disk_pressure`
2. `runtime_dependency_outage`
3. `registry_pull_failure`
4. `identity_provider_unreachable`
5. `release_drift_or_partial_apply`

These cover most of the recent real operational pain while staying small enough to validate safely.

## Next Design Step

Use this taxonomy to define:

- data model and APIs
- operator approval workflow
- MCP tool contracts
- Telegram conversation flows
