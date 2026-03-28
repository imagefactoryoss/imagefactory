# SRE Smart Bot Handover

Last updated: `2026-03-28`
Branch: `feature/sre-smartbot-test-harness`

## Current Status

SRE Smart Bot is now past the architecture-foundation stage and back into product work.

Completed:

- persisted policy config and admin API
- incident ledger schema, repository, and read APIs
- initial watcher-to-incident wiring for:
  - runtime dependency watcher
  - cluster metrics snapshot ingester
- deeper watcher-to-ledger wiring for:
  - provider readiness watcher -> findings, evidence, proposed action attempt
  - tenant asset drift watcher -> findings, evidence, proposed action attempt
  - quarantine release compliance watcher -> per-record findings, evidence, proposed action attempts, recovery resolution
- modularization checkpoint for startup/runtime loops:
  - dispatcher runner
  - workflow runner
  - stale execution watchdog
  - provider readiness watcher
  - tenant asset drift watcher
  - quarantine release compliance watcher
  - runtime dependency watcher
  - build notification subscriber health reporter
  - cluster metrics ingester runner
- first product-facing admin screen:
  - `Operations > SRE Smart Bot`
  - incident list + detail drawer for findings, evidence, actions, approvals
  - incidents workspace now includes a built-in demo incident generator with realistic scenarios for LDAP timeout, provider connectivity degradation, and release drift walkthroughs
- dedicated approval inbox:
  - `Operations > SRE Approvals`
  - pending/recent approval queue with inline approve/reject actions
- responsive/full-width pass for the admin incidents screen
- policy/settings admin screen:
  - `Operations > SRE Bot Settings`
  - policy editing for identity, automation posture, enabled domains, channel providers, and operator-defined rules
- first guarded executable action flow:
  - `reconcile_tenant_assets` can now be approved and executed through the incident ledger
- operator summary and notification improvements:
  - incident drawer now includes an executive summary section
  - operators can queue `Email Summary to Admins` directly from an incident
- additional low-risk executable remediation:
  - `review_provider_connectivity` now triggers an on-demand provider readiness refresh
- first observability/event contract slice:
  - SRE Smart Bot now publishes normalized event-bus records for:
    - `sre.finding.observed`
    - `sre.incident.resolved`
    - `sre.evidence.added`
    - `sre.action.proposed`
  - this is the intended bridge into NATS-backed detectors and later MCP/agent consumers
  - SRE Smart Bot also consumes detector-originated findings through:
    - `sre.detector.finding.observed`
  - this is the intended ingestion contract for a future Loki-query detector service
  - a first built-in detector runner now exists behind env flags, using a Loki-compatible query client and publishing observed/recovered detector events
  - local observability bootstrap now exists for development:
    - local Loki config and VS Code tasks
    - local repo-owned log shipper for `logs/*.log`
    - local Grafana provisioning with a `Loki Local` datasource and starter log dashboard
  - first golden-signal slice now exists:
    - cluster metrics snapshot ingestion derives node CPU and memory saturation findings
    - pod status snapshots now also derive pod restart pressure and pod eviction pressure findings
    - built-in log detection now also emits first app-level signals for API 5xx bursts and backend panic signatures
    - lightweight HTTP middleware snapshots now derive app-level request volume, server-error rate, and average latency findings
    - MCP now exposes those HTTP windows through a read-only `http_signals.recent` tool
    - those HTTP windows are now persisted via a dedicated `appsignals` repository and exposed back through MCP as `http_signals.history`
    - the incident drawer now renders that HTTP MCP output as structured signal cards instead of raw JSON
    - those findings write into the incident ledger as `golden_signals.saturation_risk`
    - error-style workload instability findings write as `golden_signals.error_pressure`
    - the current recommendation-only actions are:
      - `review_cluster_capacity`
      - `review_workload_stability`

## Why This Checkpoint Matters

The key goal of the recent slice was to stop growing `backend/cmd/server/main.go`.

That objective is now materially achieved:

- `main.go` is much closer to a composition root
- new SRE Smart Bot product work can attach to extracted runner boundaries
- watcher-specific incident/evidence logic can now evolve without reintroducing startup sprawl

## Recommended Next Steps

0. `AIOps Assistant` PR track is complete through `AIOPS-12`
- backlog tracker: `docs/implementation/ENGINEERING_BACKLOG.md` (`AIOps Assistant v1 PR Track`)
- keep each next-phase slice deterministic and approval-safe, with one validation artifact per slice.

1. Expand golden-signal coverage beyond the first saturation slice
- keep building on the existing metrics snapshot path before introducing a heavier metrics stack
- next likely additions: deeper NATS-specific consumer lag and consumer-pressure coverage now that the async backlog and transport slice is complete
- follow with incident summary cards and trend views so operators do not need to open MCP output just to read service health
- continue teaching drafts and interpretations to compare golden-signal direction with recent logs

Epic suggestion: **NATS consumer lag and pressure signals**
- normalize NATS consumer lag and stalled-progress findings into the incident ledger
- project consumer-pressure context into the workspace bundle and MCP tooling defaults
- correlate consumer lag with transport instability and async backlog growth in the deterministic draft
- add operator-readable summary cards for lagging consumers and stalled progress
- execution doc: `docs/implementation/SRE_SMART_BOT_NATS_CONSUMER_LAG_PRESSURE_EPIC.md`

2. Land Loki/Alloy ingestion baseline
- keep the footprint sized for the current small OKE cluster
- collect pod logs and Kubernetes events first
- local development is now good enough to validate detector rules before cluster rollout

3. Add the first detector + NATS findings flow
- detector should consume from Loki queries, not raw NATS log firehoses
- reuse the new SRE event contract instead of inventing a separate envelope

4. Define MCP tool contracts and standalone extraction seam
- keep the bot embedded for now
- make Kubernetes, OCI, database, release-state, and channel interactions contract-driven

5. Add richer AI/operator experiences only on top of the stable signal layer
- summaries, hypotheses, and guided investigation are good next candidates
- action authority must remain deterministic and approval-bound

## Validation Baseline

Recent validation completed:

```bash
cd /Users/srikarm/projects/image-factory/backend && go test ./internal/application/sresmartbot -count=1
cd /Users/srikarm/projects/image-factory/backend && go test ./internal/adapters/primary/rest -run SRESmartBot -count=1
cd /Users/srikarm/projects/image-factory/backend && go test ./internal/application/sresmartbot -run 'Test(BuildDraft|DemoService|ObserveAsyncBacklogSignals_|ObserveNATSConsumerLagSignals_|BuildIncidentWorkspace_IncludesMessagingConsumerSummaryAndBundle)' -count=1
cd /Users/srikarm/projects/image-factory/frontend && npm test -- --run src/pages/admin/__tests__/SRESmartBotIncidentsPage.test.tsx src/pages/admin/__tests__/sreSmartBotAsyncSummary.test.ts
cd /Users/srikarm/projects/image-factory && make qa-sre-smartbot-regression
```

Known test-harness noise:

- frontend SRE incident tests pass but currently emit repeated React `act(...)` warnings from async state updates in test flows.
- this is now tracked as a focused test-harness hardening follow-up; the new regression runner keeps failures visible while that warning cleanup is addressed.

## Notes For Next Contributor

- Prefer extending `backend/internal/application/bootstrap` and `backend/internal/application/sresmartbot` rather than adding new inline loops in `main.go`.
- Keep SRE Smart Bot action authority deterministic; the persona/channel layer should explain and request approval, not bypass policy.
- The incidents UI is no longer read-only. Operators can now request approval, approve/reject, and execute the first allowlisted action path, but the action authority remains intentionally narrow.
- SRE summaries now exist in two forms: a human-readable executive summary block in the incident drawer and an explicit admin email summary action that uses the existing email queue.
- The first stored action-attempt flow is intentionally conservative: most watcher-generated actions are still proposals, with only a small allowlist executable (`reconcile_tenant_assets`, `email_incident_summary`, `review_provider_connectivity`).
- Current deployment model is embedded, not standalone. Treat the settings/API/ledger contracts as the stable boundary so extraction to a worker/service is a later operational decision, not a product rewrite.
- The new SRE event-bus contracts are the intended handoff point into detector services and future standalone runtimes. Prefer consuming/publishing those structured events over scraping DB changes directly.
- The newest app-level signal path is intentionally lightweight and middleware-based. It is a good product wedge, but the next contributor should expect to add persistence/history before attempting fancy charts or trend logic.
## MCP And AI Layer

- `robot_sre_policy` now persists:
  - `mcp_servers[]`
  - `agent_runtime`
- Admin settings UI now exposes those controls.
- Incident workspace now has a dedicated MCP/AI bundle via `GET /api/v1/admin/sre/incidents/{id}/workspace`.
- Read-only MCP tool contracts now exist via:
  - `GET /api/v1/admin/sre/incidents/{id}/mcp/tools`
  - `POST /api/v1/admin/sre/incidents/{id}/mcp/invoke`
- The incidents drawer now surfaces:
  - executive summary
  - recommended investigation questions
  - suggested tooling guidance
  - enabled MCP servers for the current policy scope
  - runnable read-only MCP tools with inline output
  - deterministic draft hypotheses and investigation plan generated from the workspace + MCP tools

### Current boundary

- The AI layer is still deterministic and embedded.
- The first agent workflow now exists via:
  - `GET /api/v1/admin/sre/incidents/{id}/agent/draft`
- A deterministic triage workflow now exists via:
  - `GET /api/v1/admin/sre/incidents/{id}/agent/triage`
- A deterministic severity correlation workflow now exists via:
  - `GET /api/v1/admin/sre/incidents/{id}/agent/severity`
- A deterministic advisory suggested-action workflow now exists via:
  - `GET /api/v1/admin/sre/incidents/{id}/agent/suggested-action`
- An optional local-model interpretation layer now exists via:
  - `GET /api/v1/admin/sre/incidents/{id}/agent/interpretation`
- The interpretation endpoint now includes bounded small-LLM outputs:
  - `timeline_summary`
  - `change_detection_15m`
  - `operator_handoff_note`
  - plus `evidence_hash`, `cache_hit`, `fallback_reason`, and `citations[]` metadata
- That draft flow:
  - gathers a bounded set of read-only MCP tool results
  - ranks a few likely hypotheses
  - proposes an investigation plan
  - does not create or execute actions
- The interpretation flow:
  - stays downstream of the deterministic draft
  - now caches local-model output by `incident + evidence hash` to reduce repeated inference cost
  - now returns grounded deterministic fallback summaries when local runtime is unavailable
  - now enforces citation validation (runbook + evidence) before returning interpretation output
  - runbook citations are constrained to an allowlisted retrieval index over approved SRE docs/runbooks
  - currently supports `provider=ollama`
  - uses `agent_runtime.base_url` plus the configured local model name
  - does not add any action authority
- Suggested-action reasoning remains advisory:
  - suggestion output includes action + justification + blast-radius category
  - response explicitly marks advisory-only and approval-required execution guardrails
  - execution still requires the existing deterministic action + approval workflow
  - current default local model recommendation: `llama3.2:3b`
  - recommended local profile: Ollama at `http://127.0.0.1:11434` with `provider=ollama`
- The settings page now includes a local model probe:
  - checks daemon reachability
  - checks installed model inventory via Ollama tags
  - distinguishes "runtime reachable" from "selected model not installed"
  - gives explicit air-gapped guidance for pre-seeding model blobs
- Local operator tooling now also includes:
  - `docs/getting-started/LOCAL_OLLAMA_SETUP.md`
  - `scripts/verify-local-ollama-model.sh`
  - VS Code task: `Verify Local Ollama Model`
- Release tooling now also includes:
  - `scripts/build-baked-ollama-image.sh`
  - `make docker-build-ollama-baked`
  - `make docker-build-ollama-baked-push`
  - optional `release-deploy` wiring through `OLLAMA_ENABLED=true OLLAMA_STORAGE_MODE=baked`
- Helm/runtime defaults now also include:
  - `IF_SRE_AGENT_RUNTIME_BASE_URL`
  - `IF_SRE_AGENT_RUNTIME_MODEL`
  - when `ollama.enabled=true`, the backend policy defaults now point at the in-cluster Ollama service automatically until an operator overrides them in admin settings
- Bootstrap/reset seeding now optionally supports persisted first-run defaults:
  - `bootstrap.seedRobotSREPolicyDefaults=true`
  - only seeds global `robot_sre_policy` when `ollama.enabled=true`
  - only applies on bootstrap/reset when the config does not already exist
  - keeps `agent_runtime.enabled=false` until an operator explicitly enables the AI layer
- Current concrete MCP tool coverage is intentionally read-only:
  - `observability`: `incidents.list`, `incidents.get`, `findings.list`, `evidence.list`, `runtime_health.get`, `logs.recent`
  - `observability`: also now includes `http_signals.recent`
  - `kubernetes`: `cluster_overview.get`, `nodes.list`
  - `release`: `release_drift.summary`
- Log detector coverage now also includes notification-delivery failure signatures:
  - `Failed to enqueue notification email`
  - `Failed to save email` + `email_queue`
  - intended to surface async email/action pipeline failures as first-class SRE incidents
- Detector-rule learning now has a persisted backend loop:
  - new table: `sre_detector_rule_suggestions`
  - suggestions can be listed and reviewed through admin APIs
  - accepted suggestions are written into `robot_sre_policy.detector_rules`
  - `detector_learning_mode` supports:
    - `disabled`
    - `suggest_only`
    - `training_auto_create`
- Log detector runner now merges built-in rules with accepted custom detector rules from policy at runtime.
- NATS consumer lag and stalled-progress signals are now productized:
  - consumer lag findings are written into the incident ledger with correlation keys, evidence snapshots, and recovery behavior.
  - workspace and deterministic draft flows now summarize consumer pressure alongside transport and async backlog context.
- remediation-pack execution seams are now live in admin APIs and incident UX:
  - incident remediation packs support list, dry-run, approval-aware execute, and run-history views.
  - initial pack set includes transport stability, async backlog pressure, and provider connectivity drift.
- The new workspace endpoint is the handoff seam for:
  - future MCP tool registry
  - bounded agent runtime
  - standalone SRE Smart Bot worker extraction
- Planned intelligence direction:
  - SRE Smart Bot may suggest new detector rules from repeated correlated log patterns
  - default mode keeps those suggestions operator-reviewable before activation
  - training mode can auto-create active rules when explicitly enabled
  - golden signals should become a co-equal evidence source with logs, especially for saturation, latency, traffic, and error-rate reasoning

### Next recommended step

- Shift the next slice toward observability and intelligence plumbing:
  - tighten Loki ingestion and detector operational parity for staging/external cluster workflows.
  - keep detector contracts and event payloads stable for MCP/agent consumers.
- Continue improving operator readability:
  - enrich incident evidence and action summaries so operators can act without opening raw payload JSON.
- Add the next low-risk allowlisted executable action only when it materially improves operator leverage and remains deterministic + approval-bound.
