# Engineering Backlog

Last updated: 2026-03-28

## Purpose

Track implementation work that is agreed but not yet completed, with clear ownership of next steps.

## Mandatory Workflow (All Future Slices)

- implement each slice on a private feature branch first.
- run slice validation locally (`go test`, frontend build, `qa-sre-smartbot-aiops-eval`, `qa-sre-smartbot-regression`) before sync.
- sync only the changed slice files to OSS (avoid broad repo copy) on a matching OSS feature branch.
- rerun OSS validation, then `PR -> merge` in OSS immediately after private push.
- record private commit SHA + OSS PR number in handoff updates for traceability.

## New Backlog Entries (2026-03-27)

1. Multi-cloud Packer base OS image builds (VMware, Azure, AWS, GCP)
- Status: `done`
- Priority: `P0`
- Owner: `Backend + Platform/Ops + Frontend`
- Problem:
  - We can run Packer builds today, but provider-specific behavior is split across modern template-driven execution and legacy VM mapping.
  - Tekton Packer variable passing needs hardening, and current config persistence does not fully carry advanced Packer options.
  - We need a clear, deterministic path for base OS image builds across VMware, Azure, AWS, and GCP.
- Outcome target:
  - support deterministic base OS image builds for all four providers through one documented contract.
  - standardize parameter handling (`-var`) across local and Tekton execution.
  - align UI form fields, backend persistence, and runtime executor behavior.
  - define provider credential and readiness preflight checks for each target platform.
- Validation target:
  - reference templates for `vmware-iso`, `azure-arm`, `amazon-ebs`, and `googlecompute` build successfully in local and Tekton modes.
  - build logs show explicit Packer command arguments and provider-specific preflight results.
  - persisted build config round-trips all supported Packer settings without silent drops.
  - unsupported/invalid provider settings fail fast with actionable errors.
- Design reference:
  - `docs/implementation/PACKER_MULTI_CLOUD_BASE_IMAGE_BUILD_DESIGN.md`
- Progress:
  - PR1 contract-hardening slice completed on `feature/packer-builds`:
    - persisted/rehydrated Packer `build_vars`, `on_error`, and `parallel` fields end-to-end.
    - added explicit validation for invalid `on_error` and unsupported `parallel=true`.
    - aligned local executor argument rendering with deterministic `-var` assembly + `-on-error` support.
    - wired frontend build wizard payload mapping for the extended Packer contract.
  - PR2 Tekton parity slice completed on `feature/packer-builds`:
    - updated Tekton Packer task/pipeline contracts to carry `on-error` explicitly.
    - normalized Tekton Packer command assembly to emit repeated `-var` flags.
    - aligned Tekton render context with merged `variables + build_vars` handling and deterministic ordering.
    - added render-context tests for merged vars and `on_error` default/override behavior.
  - PR3 admin target profile backend foundation completed on `feature/packer-builds`:
    - added `packer_target_profiles` schema migration with persisted validation fields (`validation_status`, `last_validated_at`, `last_validation_message`, `last_remediation_hints`).
    - introduced backend domain + postgres repository for profile CRUD and validation-state persistence.
    - added admin APIs for create/edit/list/detail/delete/validate under `/api/v1/admin/packer-target-profiles`.
    - added deterministic config validation contract with remediation hints (non-connectivity validation for this slice).
  - PR3 frontend admin target profile UX completed on `feature/packer-builds`:
    - added admin page `/admin/infrastructure/packer-target-profiles` for list/create/edit/delete/validate.
    - wired `AdminLayout` navigation under `Build Management`.
    - added frontend service/types for `packer-target-profiles` API contract.
  - PR4 tenant profile binding and preflight completed on `feature/packer-builds`:
    - added required `build_config.packer_target_profile_id` contract for Packer builds in backend/frontend payload mapping.
    - enforced fail-fast preflight for create/start/retry when target profile is not tenant-entitled or not `valid`.
    - persisted Packer execution metadata with selected target profile/provider context and derived provider artifact identifiers.
    - added/updated backend and integration tests for the PR4 scope.
  - PR5 tenant profile selector UX completed on `feature/packer-builds`:
    - replaced free-form UUID entry with entitled target-profile selector in tenant wizard.
  - PR6 tenant VM image catalog read path completed on `feature/packer-builds`:
    - added tenant VM image catalog APIs (`/api/v1/images/vm` + `/api/v1/images/vm/{executionId}`) with provider/status/search filters.
    - added tenant VM image catalog UI route (`/images/vm`) with details drawer and source build traceability.
  - PR7 backend scheduler slice completed on `feature/packer-builds`:
    - dispatcher now processes due active schedule triggers and queues packer builds from schedule templates.
    - schedule `next_trigger_at` is now computed from cron expression on create and after each fire.
    - scheduled-origin metadata and default `forbid` concurrency policy are applied to scheduled build queuing.
  - PR7 follow-up (scheduled outcome notification hooks) completed on `feature/packer-builds`:
    - added scheduled trigger outcome IDs (`BN-011`/`BN-012`/`BN-013`) and defaults (`scheduled_failed` includes email).
    - mapped scheduled status updates (`scheduled_queued`, `scheduled_failed`, `scheduled_noop`) into notification routing.
    - schedule runner now emits build status events for queue/fail/no-op outcomes to drive trigger-based delivery.
  - PR7 frontend notification catalog parity completed on `feature/packer-builds`:
    - added `BN-011`/`BN-012`/`BN-013` trigger options in admin tenant defaults UI and tenant details notification drawer.
    - added scheduled trigger rows in project notification matrix with descriptions and payload typing updates.
  - Provider artifact extraction hardening completed on `feature/packer-builds`:
    - improved identifier extraction for nested/non-array artifact payloads in execution metadata enrichment.
    - expanded GCP pattern support for `projects/.../global/images/...` values without leading slash.
    - reduced VMware false positives by requiring identifier-like VMware markers instead of bare labels.
  - PR8 backend lifecycle action foundation completed on `feature/packer-builds`:
    - added tenant VM image lifecycle endpoints:
      - `POST /api/v1/images/vm/{executionId}/promote`
      - `POST /api/v1/images/vm/{executionId}/deprecate`
      - `DELETE /api/v1/images/vm/{executionId}`
    - lifecycle transitions persist metadata overrides under `metadata.packer.lifecycle_state`.
    - list/get VM catalog responses now honor persisted lifecycle overrides (`released`, `deprecated`, `deleted`).
    - added guardrails to block lifecycle transitions for active/failed/cancelled executions.
  - PR8 frontend lifecycle action controls completed on `feature/packer-builds`:
    - VM catalog table now exposes `Promote`, `Deprecate`, and `Delete` actions with confirmation dialog UX.
    - action availability is lifecycle-aware and disables invalid transitions in UI.
    - successful actions refresh both list view and open detail drawer state.
  - PR8 lifecycle guardrail + audit-depth hardening completed on `feature/packer-builds`:
    - `deprecate`/`delete` transitions now require explicit reason payloads.
    - `delete` transition now requires current lifecycle state `deprecated` (policy guardrail).
    - lifecycle metadata now records bounded `lifecycle_history` entries with actor/reason/timestamp.
    - VM catalog responses now expose lifecycle last-action fields and lifecycle history for operator traceability.
  - PR8 lifecycle action contract normalization completed on `feature/packer-builds`:
    - VM catalog API now returns server-calculated `action_permissions` (`can_promote`, `can_deprecate`, `can_delete`).
    - frontend VM action controls now rely on backend policy flags instead of duplicated client-side transition logic.
  - PR8 frontend lifecycle reason-entry UX completed on `feature/packer-builds`:
    - VM catalog now requires operator-entered reason text in an in-app modal for `deprecate` and `delete` actions before confirmation.
    - frontend now forwards typed reason payloads directly to lifecycle APIs instead of generated default reason strings.
  - PR8 lifecycle reason payload hardening completed on `feature/packer-builds`:
    - backend now enforces a 500-character maximum reason length for lifecycle transitions.
    - VM catalog reason modal now mirrors the 500-character limit with inline counter feedback.
  - PR8 lifecycle transition-mode contract clarity completed on `feature/packer-builds`:
    - VM catalog API now returns `lifecycle_transition_mode` for list/detail/action responses.
    - transition mode now reflects runtime execution path (`metadata_only` / `provider_native` / `hybrid`) based on execution mode and provider support.
  - PR8 lifecycle audit mode-depth completed on `feature/packer-builds`:
    - lifecycle history entries now persist `transition_mode` for each recorded transition.
    - VM catalog lifecycle history UI now renders per-entry transition mode for audit clarity.
  - PR8 lifecycle action response contract hardening completed on `feature/packer-builds`:
    - idempotent lifecycle transitions (already in target state) now return `data` + `message`, matching successful transition response shape.
    - VM catalog item mapping now uses a shared backend builder to keep list/detail/action payload fields consistent.
  - PR8 lifecycle action UX parity follow-up completed on `feature/packer-builds`:
    - frontend VM lifecycle toasts now display backend-provided action messages.
    - idempotent transition messaging is now surfaced directly in UI (for example, already-in-state responses).
  - PR8 lifecycle transition-mode normalization hardening completed on `feature/packer-builds`:
    - backend now normalizes transition mode to an allowlist (`metadata_only`, `provider_native`, `hybrid`) with unknown values falling back to `metadata_only`.
  - PR8 lifecycle transition-mode default contract hardening completed on `feature/packer-builds`:
    - VM catalog responses now guarantee non-empty `lifecycle_transition_mode`, defaulting to `metadata_only` for empty/invalid metadata payloads.
  - PR8 lifecycle transition-mode visibility follow-up completed on `feature/packer-builds`:
    - VM catalog table lifecycle column now renders transition mode so operators can assess lifecycle semantics without opening details.
  - PR8 lifecycle transition-mode filterability follow-up completed on `feature/packer-builds`:
    - VM catalog list API now supports filtering by `transition_mode`.
    - VM catalog filter bar now includes transition-mode selector (`All`, `metadata_only`, `provider_native`, `hybrid`).
  - PR8 provider-lifecycle execution seam foundation completed on `feature/packer-builds`:
    - VM lifecycle transitions now route through a lifecycle executor interface before metadata persistence.
    - executor now supports provider-native lifecycle execution across AWS, VMware, Azure, and GCP when execution mode enables it.
  - PR8 provider-lifecycle execution-mode policy gate completed on `feature/packer-builds`:
    - added `IF_VM_LIFECYCLE_EXECUTION_MODE` gate (`metadata_only`, `prefer_provider_native`, `require_provider_native`).
    - `require_provider_native` now fails closed with `501` for unsupported provider/state paths and missing provider execution metadata.
  - PR8 provider-native lifecycle action initial implementation completed on `feature/packer-builds`:
    - AWS `delete` lifecycle transitions now execute provider-native `DeregisterImage` via EC2 when execution mode enables provider-native execution.
    - transition mode persists as `provider_native` on successful AWS native delete path; missing/invalid AWS image metadata now returns `400`.
  - PR8 provider-native lifecycle action expansion completed on `feature/packer-builds`:
    - AWS `deprecate` lifecycle transitions now execute provider-native `EnableImageDeprecation` via EC2 when provider-native execution mode is enabled.
  - PR8 provider-native lifecycle release expansion completed on `feature/packer-builds`:
    - AWS `released` lifecycle transitions now execute provider-native `DisableImageDeprecation` via EC2 when provider-native execution mode is enabled.
  - PR8 provider-native lifecycle metadata-compatibility expansion completed on `feature/packer-builds`:
    - AWS native lifecycle transitions now fall back to execution artifact values when `provider_artifact_identifiers.aws` is absent, improving compatibility with older build metadata payloads.
  - PR8 provider-native lifecycle artifact-shape compatibility expansion completed on `feature/packer-builds`:
    - execution artifact extraction now scans nested object/array payloads for string identifiers, improving native lifecycle fallback for non-array artifact shapes.
  - PR8 VMware provider-native lifecycle execution completed on `feature/packer-builds`:
    - VMware `released` / `deprecated` / `deleted` lifecycle transitions now support provider-native execution through vCenter when execution mode enables provider-native transitions.
    - VMware native lifecycle supports identifier fallback from both provider metadata and execution artifacts; invalid/missing VMware identifiers fail with `400`.
  - PR8 Azure provider-native lifecycle execution completed on `feature/packer-builds`:
    - Azure `released` / `deprecated` / `deleted` lifecycle transitions now support provider-native execution through Azure ARM API when execution mode enables provider-native transitions.
    - Azure native lifecycle supports identifier fallback from both provider metadata and execution artifacts; invalid/missing Azure identifiers fail with `400`.
  - PR8 GCP provider-native lifecycle execution completed on `feature/packer-builds`:
    - GCP `released` / `deprecated` / `deleted` lifecycle transitions now support provider-native execution through Compute API when execution mode enables provider-native transitions.
    - GCP native lifecycle supports identifier fallback from both provider metadata and execution artifacts; invalid/missing GCP identifiers fail with `400`.
  - PR8 provider-native lifecycle audit-depth expansion completed on `feature/packer-builds`:
    - lifecycle metadata/history now persists provider execution audit fields (`provider_action`, `provider_identifier`, `provider_outcome`) for provider-native transitions.
    - VM catalog payloads now expose latest provider execution audit fields for faster operator diagnostics.
  - PR8 provider-native lifecycle smoke tooling completed on `feature/packer-builds`:
    - added provider-native smoke runner script `scripts/packer-lifecycle-provider-native-smoke.sh` with destructive-guard and provider-native assertions.
    - added operator runbook `docs/implementation/PACKER_VM_LIFECYCLE_PROVIDER_NATIVE_SMOKE_RUNBOOK.md` for env setup, execution order, and rollback guidance.
  - PR8 provider-native lifecycle matrix validation tooling completed on `feature/packer-builds`:
    - added matrix runner script `scripts/packer-lifecycle-provider-native-matrix.sh` to orchestrate per-provider smoke runs with consolidated evidence logging.
    - added matrix runbook `docs/implementation/PACKER_VM_LIFECYCLE_PROVIDER_NATIVE_MATRIX_RUNBOOK.md` for provider execution-ID mapping, safety guards, and pass/fail criteria.
  - PR8 provider-native lifecycle no-cloud mock validation mode completed on `feature/packer-builds`:
    - smoke/matrix tooling now supports `SMOKE_MODE=mock_success` for deterministic no-cloud workflow validation.
    - mock mode keeps transition assertion and report contracts intact while skipping provider API calls and external credentials.
  - PR8 provider-native staging evidence closure tooling completed on `feature/packer-builds`:
    - added QA runner `scripts/qa/packer_provider_native_matrix_validate.sh` and `make qa-packer-provider-native-matrix` target for timestamped evidence capture.
    - added staging runbook + validation template: `docs/qa/PACKER_PROVIDER_NATIVE_MATRIX_STAGING_RUNBOOK.md`, `docs/qa/PACKER_PROVIDER_NATIVE_MATRIX_VALIDATION_LOG.md`.
    - closure criteria now explicit: run `SMOKE_MODE=api` matrix in staging/prod-like environment, attach artifacts, and capture Platform/Ops + QA signoff.
  - Engineering closure status:
    - Packer implementation PR track is closed from engineering scope (contract, runtime, UI, and QA tooling shipped).
    - staging/prod-like provider API execution evidence is tracked as operational follow-up using the shipped runbook/template and is not blocking engineering PR closure.

## Backlog Review Summary (2026-03-16)

- SRE Smart Bot policy/config foundation is now materially complete in backend + admin UX; backlog status was lagging behind shipped behavior.
- SRE Smart Bot operator and detector rule flows now include immediate drawer persistence with scoped inline save-state feedback.
- SRE Smart Bot channel providers now support table + drawer management, provider-specific settings blocks, and immediate add/edit/remove persistence.
- Primary remaining functional gap is not basic policy editing; it is turning accepted incident guidance into faster, safer operator execution outcomes.

## Current Sprint Focus (P0)

0. SRE Smart Bot Phase 0: policy foundation, incident model, and admin controls
- Status: `done`
- Priority: `P0`
- Owner: `Backend + Platform/Ops`
- Problem: SRE Smart Bot design exists, but there is no persisted control surface for environment posture, operator-defined rules, channel providers, or approval boundaries.
- Outcome target:
  - add first-class `robot_sre_policy` system config with defaults, validation, channel-provider contract, and admin API support.
  - define implementation epics and phase ordering across detection, incidents, approvals, and channels.
  - keep v1 operator-defined rules bounded to safe metadata and thresholds, not arbitrary code execution.
- Validation target:
  - backend config service exposes deterministic default policy and rejects invalid operator rule payloads.
  - admin route contract exists for reading/updating SRE Smart Bot policy.
  - implementation backlog/plan documents reflect phased delivery from watcher MVP to chat approvals.
- Completion note:
  - global `robot_sre_policy` config, validation, and admin read/update APIs are live.
  - admin settings now support dedicated operator/detector rule pages, drawer-based staged editing, and immediate save persistence.
  - channel provider contract is exposed in admin UX with provider-specific settings and immediate persistence semantics.
- Source: `docs/implementation/ROBOT_SRE_OPS_PERSONA_REQUIREMENTS_AND_DESIGN.md`, `docs/implementation/ROBOT_SRE_INCIDENT_TAXONOMY_AND_POLICY_MATRIX.md`, `docs/implementation/ROBOT_SRE_LOG_INTELLIGENCE_AND_INCIDENT_DETECTION.md`

0. SRE Smart Bot Phase 1: incident ledger and bounded runtime watcher actions
- Status: `in_progress`
- Priority: `P0`
- Owner: `Backend + Platform/Ops`
- Problem: current watchers expose health but do not persist incidents, evidence, action attempts, or approval state in a reusable way.
- Outcome target:
  - introduce incident, finding, action-attempt, and approval persistence.
  - convert watcher outputs into structured incident records with correlation keys.
  - enable safe low-risk containment actions from policy, with cooldowns and audit trail.
- Validation target:
  - repeated runtime/cluster signals correlate into a single incident thread.
  - every automated action has stored evidence, policy reason, actor, and outcome.
  - no disruptive action executes without explicit approval state.
- Progress:
  - incident ledger schema/repository/admin read APIs are in place.
  - runtime dependency watcher and cluster metrics snapshot ingester are already writing/resolving incidents.
  - provider readiness, tenant asset drift, and release compliance watchers now write findings/evidence and proposed action-attempts into the ledger.
  - major watcher/bootstrap modularization checkpoint completed so new SRE product work no longer expands `main.go`.
  - first admin incident experience now exists at `Operations > SRE Smart Bot`.
  - SRE Smart Bot policy/settings admin page now exists at `Operations > SRE Bot Settings`.
  - dedicated operator approval inbox now exists at `Operations > SRE Approvals`.
  - first guarded action path is live for `reconcile_tenant_assets`, including request-approval, approve/reject, and execute flows.
  - incident drawer now includes an executive summary section.
  - operators can now queue `email_incident_summary` directly from an incident, using the existing email queue and admin recipient lookup.
  - `review_provider_connectivity` is now executable and triggers an on-demand provider readiness refresh.
  - focused SRE regression workflow now exists via `make qa-sre-smartbot-regression` (`scripts/qa/sre_smartbot_regression_validate.sh`) to capture backend + frontend evidence in one timestamped artifact.
  - known frontend `act(...)` warning noise remains tracked as test-harness hardening follow-up; regression runner keeps pass/fail drift visible while warning cleanup continues.
- Next step:
  - shift the next slice toward observability and intelligence plumbing: Loki ingestion, structured detectors, MCP integration, and agent-facing tool contracts.
  - continue improving evidence/action summaries so the operator story is readable without opening raw payloads.
  - add the next low-risk allowlisted executable action only if it meaningfully improves operator leverage without becoming destructive.
- Source: Robot SRE implementation kickoff

0. SRE Smart Bot Phase 2: operator channels and conversational approvals
- Status: `in_progress`
- Priority: `P1`
- Owner: `Backend + Frontend + Platform/Ops`
- Problem: operators need a low-friction interface for incident summaries, approvals, and follow-up questions outside the admin UI, and many enterprises require internal gateways instead of consumer chat apps.
- Outcome target:
  - add provider-based operator channel integration.
  - expose incident summaries, evidence links, and approval actions over chat and admin UI.
  - keep final action authorization in deterministic policy code, not the LLM layer.
- Sequencing note:
  - keep the bot embedded in the backend while contracts settle.
  - extract to a standalone worker/service once policy, incident, approval, and first executable-action contracts are stable.
- Validation target:
  - operator can approve or reject a pending remediation from a configured provider and from admin UI.
  - incident thread remains consistent across chat and in-app views.
  - chat delivery failures degrade cleanly back to in-app notifications.
- Progress:
  - provider-based channel policy configuration now supports in-app, email, webhook, slack, teams, telegram, whatsapp, and custom providers.
  - channel providers include kind-specific settings blocks and immediate add/edit/remove persistence with inline save feedback.
  - approvals and incident workflows are operational in admin UI; cross-channel action parity remains the next slice.
- Source: Robot SRE channel design

0. SRE Smart Bot Phase 3: log intelligence and NATS event pipeline
- Status: `in_progress`
- Priority: `P1`
- Owner: `Backend + Platform/Ops`
- Problem: log-derived issues are currently detected manually and not normalized into the same incident pipeline as runtime and cluster signals.
- Outcome target:
  - add lightweight detector service fed by logs and runtime events.
  - publish normalized findings over NATS for Robot SRE ingestion.
  - correlate findings with metrics snapshots and runtime health before remediation.
- Validation target:
  - known log signatures create structured findings with bounded false-positive behavior.
  - detector output flows into the incident ledger through NATS.
  - remediation remains blocked when findings lack corroborating evidence.
- Progress:
  - the SRE Smart Bot ledger now emits normalized event-bus contracts for `sre.finding.observed`, `sre.incident.resolved`, `sre.evidence.added`, and `sre.action.proposed`.
  - this creates the first stable bridge between watcher-ledger activity and future NATS/detector consumers.
  - the backend now also consumes normalized detector events via `sre.detector.finding.observed`, so Loki-driven detectors can publish one contract and let SRE Smart Bot handle incident correlation.
  - a built-in Loki-compatible detector runner now exists, with env-driven config and first rules for Docker Hub rate limits, disk-pressure signatures, and LDAP timeouts.
- Source: `docs/implementation/ROBOT_SRE_LOG_INTELLIGENCE_AND_INCIDENT_DETECTION.md`

0. SRE Smart Bot Phase 7: guided remediation packs and one-click execution (functional value epic)
- Status: `in_progress`
- Priority: `P0`
- Owner: `Backend + Frontend + Platform/Ops`
- Problem: incidents now have strong evidence and policy-driven recommendations, but operators still spend high-effort time translating suggested actions into consistent execution steps.
- Outcome target:
  - introduce deterministic remediation packs for top recurring incident types (for example: NATS transport instability, backlog pressure, provider connectivity drift).
  - add one-click `dry_run` and guarded `execute` paths from incident workspace using existing approval and audit model.
  - attach structured preconditions, blast-radius estimates, and rollback hints to every pack.
- Validation target:
  - operator can run a pack dry-run from incident detail and receive explicit pass/fail precondition output.
  - approved execution records plan, actor, evidence delta, and outcome in the incident ledger.
  - mean time to first safe remediation action decreases for targeted incident categories.
- Scope guardrails:
  - no direct destructive automation without explicit approval state.
  - keep action execution deterministic and policy-bound; AI remains advisory.
- Progress:
  - remediation pack schema and repository seam now exist, including persisted remediation-pack run records.
  - policy-backed remediation pack defaults are now validated and exposed through Robot SRE policy config.
  - admin APIs now expose global and incident-scoped remediation pack listing contracts with focused tests.
- Next step:
  - implement dry-run and execute endpoints with approval-gated execution and incident evidence linkage.
  - add incident workspace remediation-pack panel for dry-run and execute flow completion.
- Source: `docs/implementation/SRE_SMART_BOT_GUIDED_REMEDIATION_PACKS_EPIC.md`

0. SRE Smart Bot Phase 3.5: golden signals and metric-driven incident detection
- Status: `in_progress`
- Priority: `P1`
- Owner: `Backend + Platform/Ops`
- Problem: log evidence is useful, but SRE Smart Bot still lacks first-class golden signal awareness for latency, traffic, errors, and saturation trends.
- Outcome target:
  - treat golden signals as normalized SRE inputs, not dashboard-only metrics.
  - derive metric-backed findings from the existing cluster metrics snapshot ingester first, then expand into app/service latency, traffic, and error signals.
  - correlate metric findings with log findings and runtime health before remediation.
- Validation target:
  - cluster metrics ingestion creates saturation-risk findings with evidence and proposed actions.
  - incident threads can combine metric and log signals under one correlation path.
  - thresholds remain configurable and conservative by default.
- Progress:
  - node CPU and memory saturation are now derived from the existing cluster metrics snapshots and written into the incident ledger as `golden_signals.saturation_risk`.
  - pod restart pressure and pod eviction pressure are now derived from pod status snapshots and written into the incident ledger as `golden_signals.error_pressure`.
  - first app-level log-derived signals now exist for API 5xx bursts and backend panic signatures.
  - lightweight HTTP request signal ingestion now exists for app-level request volume, server-error rate, and average latency windows.
  - read-only MCP coverage now includes recent HTTP golden-signal windows so the AI workspace can cite app traffic, latency, and error context directly.
  - the incidents workspace now renders `http_signals.recent` as a structured operator-readable panel instead of raw JSON.
  - HTTP signal windows are now persisted through a dedicated `appsignals` repository seam, and MCP now exposes recent history via `http_signals.history`.
  - async backlog signal ingestion now exists through a dedicated `asyncsignals` runner seam, using build queue depth, pending email queue depth, and messaging outbox backlog as first async-pressure inputs.
  - NATS transport health signals now exist for disconnects and reconnect storms, with matching MCP coverage through `messaging_transport.recent`.
  - the deterministic draft now reasons over async backlog pressure, messaging transport instability, HTTP trends, and recent logs together.
  - the async backlog + transport pressure epic is now complete across normalized backlog incidents, workspace summaries, summary-tab rendering, deterministic async causality, demo scenarios, and published validation notes.
  - the current first actions are recommendation-only: `review_cluster_capacity` and `review_workload_stability`.
- Next step:
  - expand messaging coverage toward true NATS lag and consumer pressure once those metrics are available.
  - package the current golden-signal story into external-cluster deployment defaults and longer-running soak validation.
- Source: golden-signals expansion follow-up

0. SRE Smart Bot Phase 4: Loki and Alloy ingestion baseline
- Status: `planned`
- Priority: `P1`
- Owner: `Platform/Ops`
- Problem: there is no lightweight in-cluster log ingestion baseline for detector rules, incident evidence enrichment, or future operator investigations.
- Outcome target:
  - deploy Alloy for Kubernetes pod logs and events.
  - deploy Loki in monolithic mode with conservative retention and resource limits.
  - keep raw logs out of NATS and reserve NATS for structured control-plane events.
- Validation target:
  - Alloy is scraping selected app/system namespaces successfully.
  - Loki retention and resource usage stay within the small-cluster budget.
  - detector services can query Loki for recent incident evidence windows.
- Source: `docs/implementation/ROBOT_SRE_LOG_INTELLIGENCE_AND_INCIDENT_DETECTION.md`

0. SRE Smart Bot Phase 5: MCP tool contracts and standalone runtime seam
- Status: `planned`
- Priority: `P1`
- Owner: `Backend + Platform/Ops`
- Problem: agent/runtime integrations will become fragile if tool access is improvised separately for Kubernetes, OCI, database, and release state.
- Outcome target:
  - define MCP-style tool contracts for investigation and evidence gathering.
  - keep deterministic policy and action authority outside the LLM layer.
  - prepare the extraction seam from embedded bot runtime to standalone worker/service.
- Validation target:
  - tool contracts are explicit enough for a dedicated SRE Smart Bot worker to consume.
  - the same incident/action/approval ledger remains the system of record after extraction.
  - privileged actions still require allowlists and approval state.
- Progress:
  - read-only MCP now covers:
    - incidents, findings, evidence, runtime health
    - recent logs from Loki
    - cluster overview and nodes
    - release drift summary
    - recent HTTP golden-signal windows
  - deterministic triage contract now exists via `GET /api/v1/admin/sre/incidents/{id}/agent/triage`, projecting probable cause, confidence, next checks, and approval-safe recommended action from the grounded draft.
  - the incidents workspace can now render specialized output for HTTP golden signals instead of raw JSON blobs.
- Source: `docs/implementation/ROBOT_SRE_OPS_PERSONA_REQUIREMENTS_AND_DESIGN.md`

0. SRE Smart Bot Phase 6: AI operator experiences and summaries
- Status: `planned`
- Priority: `P2`
- Owner: `Backend + Frontend`
- Problem: incident data is becoming rich, but operators still need better summaries, hypotheses, and guided investigation help without giving up control.
- Outcome target:
  - add AI-generated incident summaries and suggested next steps on top of stable evidence.
  - support configurable provider delivery such as email or enterprise chat gateways.
  - keep AI outputs advisory unless explicitly routed through policy and approval.
- Validation target:
  - AI summaries cite structured findings/evidence rather than hallucinated state.
  - operator-facing channels remain provider-contract based, not tied to Telegram/WhatsApp.
  - action execution still flows through the deterministic approval path.
- Progress:
  - local model probing now checks reachability, installed-model inventory, and air-gapped guidance.
  - Helm now supports optional in-cluster Ollama runtime with configurable storage mode (`baked`, `pvc`, `hostPath`, `emptyDir`).
  - baked-image tooling and release-path wiring now exist for `image-factory-ollama`.
  - backend policy defaults can now inherit `IF_SRE_AGENT_RUNTIME_BASE_URL` and `IF_SRE_AGENT_RUNTIME_MODEL`, so in-cluster Ollama deployments start with a sensible internal default.
  - bootstrap/reset can now optionally seed the persisted global `robot_sre_policy` on first run via `bootstrap.seedRobotSREPolicyDefaults=true` when `ollama.enabled=true`.
  - incidents workspace now includes a demo incident generator so AI/operator demos can create grounded scenarios on demand instead of manually breaking runtime config.
- Next step:
  - add richer MCP coverage and keep the agent runtime optional, bounded, and approval-safe.
  - teach the draft/interpretation layer to compare recent logs with recent golden-signal windows explicitly.
- Source: `docs/implementation/ROBOT_SRE_OPS_PERSONA_REQUIREMENTS_AND_DESIGN.md`

### AIOps Assistant v1 PR Track

1. `AIOPS-01` Deterministic Incident Triage Copilot
- Status: `done`
- Scope:
  - `GET /api/v1/admin/sre/incidents/{id}/agent/triage`
  - outputs probable cause, confidence, next 3 checks, and safe recommended action.
  - preserves deterministic/approval-safe action authority.

2. `AIOPS-02` Signal Correlation Severity Layer + UI cards
- Status: `done`
- Scope:
  - correlate `logs + http_signals + async_backlog + messaging_transport` into one severity score contract.
  - add operator-facing "Why This Is Severe" cards in incident summary/AI workspace.
- Validation:
  - deterministic score tests for stable, mixed-cause, and transport-driven pressure cases.
  - frontend card rendering tests for low/medium/high/critical severity explanations.
- Completion note:
  - added deterministic severity endpoint `GET /api/v1/admin/sre/incidents/{id}/agent/severity`.
  - AI workspace/operator control center now render correlated severity score + factor cards.

3. `AIOPS-03` Small-LLM usefulness pack (bounded)
- Status: `done`
- Scope:
  - local-model outputs only for timeline summary, 15-minute change detection, and operator handoff note.
  - cache by incident/evidence hash to reduce repeated inference and keep responses stable.
- Validation:
  - deterministic cache hit/miss tests.
  - interpretation endpoints return grounded fallback when model unavailable.
- Completion note:
  - `GET /api/v1/admin/sre/incidents/{id}/agent/interpretation` now returns bounded small-LLM fields: `timeline_summary`, `change_detection_15m`, and `operator_handoff_note`.
  - added in-memory cache keyed by `incident_id + evidence_hash + runtime provider/model` to stabilize repeated interpretation calls.
  - interpretation now returns deterministic grounded fallback summaries when local model runtime is unavailable instead of failing request flow.

4. `AIOPS-04` Runbook grounding + citation contract
- Status: `done`
- Scope:
  - build retrieval index over approved runbooks/docs.
  - require agent outputs to cite runbook sections and current incident evidence references.
- Validation:
  - responses without citations fail validation gate.
  - retrieval contract tests enforce runbook-only source allowlist.
- Completion note:
  - interpretation responses now include `citations[]` with mixed `runbook` and `evidence` citation kinds.
  - added allowlisted in-memory runbook grounding index with section-level keyword matching for incident draft context.
  - citation validation gate now rejects interpretation output lacking runbook/evidence citations or referencing non-allowlisted runbook sources.

5. `AIOPS-05` Approval-safe suggested action reasoning
- Status: `done`
- Scope:
  - AI suggestion envelope includes action, justification, and blast-radius category.
  - execution continues to require existing deterministic approval gates.
- Validation:
  - policy tests prove suggestions never bypass approval workflow.
  - UI clearly labels advisory suggestion vs executable action.
- Completion note:
  - added deterministic suggested action endpoint `GET /api/v1/admin/sre/incidents/{id}/agent/suggested-action`.
  - suggestion contract now includes `action_key`, `action_summary`, `justification`, `blast_radius`, and explicit guardrail fields (`advisory_only`, `execution_requires_approval`, `execution_guardrail`).
  - AI workspace and operator control center now render advisory-only suggested-action cards with clear non-executable labeling.

6. `AIOPS-06` Evaluation harness (replay + safety)
- Status: `done`
- Scope:
  - replay suite with fixed incidents and expected triage/summary outcomes.
  - evaluate correctness, hallucination guard, and policy compliance.
- Validation:
  - publish reproducible QA runner + artifact log under `docs/qa/artifacts`.
  - fail build/PR checks when hallucination/policy assertions regress.
- Completion note:
  - added deterministic replay harness test `TestAIOpsEvaluationHarness_ReplaySuite` to verify triage/severity/suggested-action correctness across fixed incident fixtures.
  - added interpretation safety gate to reject unsafe generated summary content (approval-bypass/auto-execute phrasing) and fallback to deterministic grounded output.
  - added reproducible QA runner `scripts/qa/sre_smartbot_aiops_eval_validate.sh` and Make target `qa-sre-smartbot-aiops-eval` with artifact logs under `docs/qa/artifacts`.

7. `AIOPS-07` Incident scorecard contract + expanded “Why Severe” cards
- Status: `done`
- Scope:
  - add consolidated incident scorecard endpoint for operators: probable cause, confidence, severity score, top “why severe” cards, and approval-safe action guidance.
  - render scorecard cards in incident AI surfaces for both desktop and mobile layouts.
- Validation:
  - backend tests for scorecard composition and why-severe card trimming.
  - frontend build and SRE regression/eval harness remain green.
- Completion note:
  - added `GET /api/v1/admin/sre/incidents/{id}/agent/scorecard`.
  - incident AI workspace now includes dedicated `Incident Scorecard` cards with severity + why-severe factor breakdown.

8. `AIOPS-08` One-call deterministic AI snapshot
- Status: `done`
- Scope:
  - add deterministic one-call snapshot endpoint returning triage, severity, scorecard, and suggested action in a single payload.
  - add operator UI control to generate all core advisory views in one action.
- Validation:
  - backend tests for snapshot composition + approval-bound guarantees.
  - REST contract tests/build/regression and AIOPS eval harness remain green.
- Completion note:
  - added `GET /api/v1/admin/sre/incidents/{id}/agent/snapshot`.
  - incidents AI page now includes `Generate AI Snapshot` and dedicated snapshot cards in both layout variants.

9. `AIOPS-09` Snapshot operator handoff + policy guardrail bundle
- Status: `done`
- Scope:
  - include deterministic operator handoff note and policy guardrails in snapshot contract.
  - surface guardrail language directly in snapshot UI cards.
- Validation:
  - backend snapshot tests enforce approval-bound handoff/guardrail language.
  - full SRE regression + AIOPS eval harness remain green.
- Completion note:
  - `AgentIncidentSnapshotResponse` now includes `operator_handoff_note` and `policy_guardrails[]`.
  - AI snapshot cards now render explicit handoff guidance and advisory/approval guardrails.

10. `AIOPS-10` Snapshot evidence health signals
- Status: `done`
- Scope:
  - add deterministic evidence-health fields to snapshot (`expected`, `observed`, `coverage_percent`, `health_note`).
  - surface evidence-health visibility in AI snapshot cards so operators can gauge confidence quickly.
- Validation:
  - backend tests cover coverage and note banding.
  - full SRE regression + AIOPS eval harness remain green.
- Completion note:
  - `AgentIncidentSnapshotResponse` now includes evidence coverage and observed-signal metadata.
  - AI snapshot cards now show evidence coverage percent + evidence health summary.

11. `AIOPS-11` Deterministic “Why Severe” weighting + operator rationale
- Status: `done`
- Scope:
  - extend severity/scorecard factors with deterministic weight percentage and explicit operator rationale text.
  - surface weight + rationale in incident AI cards for both desktop and mobile layouts.
- Validation:
  - backend tests enforce positive bounded factor weights and non-empty operator rationale fields.
  - full SRE regression + AIOPS eval harness remain green.
- Completion note:
  - `AgentSeverityFactor` now includes `weight_percent` and `operator_rationale` fields populated deterministically from contribution and incident severity level.
  - incident severity and scorecard cards now render contribution, weight percentage, and operator rationale for each why-severe factor.

12. `AIOPS-12` Runbook-grounded next-check mapping
- Status: `done`
- Scope:
  - add deterministic triage mapping for each next check with runbook source/section and supporting evidence signal.
  - surface mapping in triage and snapshot cards so operators can follow check-to-runbook-evidence flow.
- Validation:
  - backend tests enforce populated runbook/evidence mapping for triage and snapshot outputs.
  - full SRE regression + AIOPS eval harness remain green.
- Completion note:
  - `AgentTriageResponse` now includes `next_check_refs[]` with `check`, `runbook_source`, `runbook_section`, `evidence_signals[]`, and `evidence_note`.
  - incidents AI cards now render runbook-grounded check mappings in both desktop and mobile layouts.

0. Deployment guardrails: strict Helm configuration (no silent fallbacks) + Supabase pooler stability
- Status: `done`
- Priority: `P0`
- Owner: `Platform/Ops + Backend`
- Problem: Helm upgrades could silently fall back to in-cluster defaults (especially DB), causing runtime drift; Supabase session pooler hit max-client saturation under default service connection pool settings.
- Outcome target:
  - enforce fail-fast chart validation for ambiguous/implicit config (database mode + component image/storage config).
  - remove implicit chart fallback behavior for DB and service image wiring.
  - stabilize external DB runtime by explicitly configuring low DB pool limits for pooler mode.
  - confirm release runtime points to external Supabase and not in-cluster Postgres.
- Validation target:
  - `helm template` passes for valid `incluster` and `external` modes, fails for ambiguous configs with explicit error.
  - cluster release upgraded in `external` mode (`postgres.enabled=false`) with verified runtime env (`IF_DATABASE_HOST`, `IF_DATABASE_*` pool settings).
  - post-rollout logs clear for `MaxClientsInSessionMode` saturation errors.
- Source: `deploy/helm/image-factory/templates/_helpers.tpl`, `deploy/helm/image-factory/templates/configmap.yaml`, `deploy/helm/image-factory/values.yaml`, cluster release revisions `93-95`.

0. Quarantine Phase F intake hardening: EPR-gated Quarantine Intake Workflow stepper UX
- Status: `done`
- Priority: `P0`
- Owner: `Frontend + Backend`
- Problem: tenant intake still allowed dead-end clicks and unclear prereq progression (users discovered missing approval only after action attempts).
- Outcome target:
  - enforce deterministic 3-step intake progression (`EPR check -> approval status -> quarantine request details`).
  - lock Step 3 until EPR is confirmed/approved.
  - show explicit status banner for pending/approved/rejected prereq states.
- Validation target:
  - frontend: `npm run test -- --run src/pages/quarantine/__tests__/QuarantineRequestsPage.test.tsx` (includes lock/unlock stepper regression tests).
  - backend/frontend EPR API smoke: create/list admin+tenant EPR request flow validated on local.
- Source: `docs/implementation/IMAGE_QUARANTINE_PROCESS_IMPLEMENTATION_PLAN.md` (Phase G intake flow)

0. Quarantine Phase G admission guardrails: deploy/build intake enforcement expansion (G-02)
- Status: `done`
- Priority: `P0`
- Owner: `Backend + Frontend + QA`
- Problem: build preflight guardrails exist, but released-only enforcement is not yet fully consistent across remaining intake/deployment paths and UX remediation hints.
- Outcome target:
  - extend `quarantine_artifact_not_released` admission checks beyond current build create slice to remaining deployment/intake paths.
  - surface consistent user remediation messaging in UI for denied unreleased refs.
  - add integration/regression coverage across entitlement + EPR + release-state gates.
- Validation target:
  - backend: targeted REST tests for each guarded path returning deterministic `409 quarantine_artifact_not_released`.
  - frontend: actionable error rendering tests in affected create/edit flows.
- Completion note:
  - closed as part of Phases `I` through `K` hardening, with deterministic deny/remediation contracts carried through intake and execution readiness UX flows.
- Source: `docs/implementation/IMAGE_QUARANTINE_PROCESS_IMPLEMENTATION_PLAN.md` (Phase G guardrails)

0. Quarantine request dedicated detail page (tenant + admin/reviewer)
- Status: `done`
- Priority: `P0`
- Owner: `Frontend`
- Problem: quarantine execution diagnostics are currently mostly drawer-only, making deep-linking and operational triage difficult compared to dedicated build details.
- Outcome target:
  - add first-class request detail route for tenant queue.
  - add first-class request detail route for admin/reviewer queue.
  - expose `View Page` action from queue rows while retaining existing in-page drawer action.
- Validation target:
  - frontend route contract includes tenant/admin/reviewer detail routes.
  - details page fetches direct tenant request and admin/reviewer request lookup by id.
- Source: quarantine UX hardening follow-up (details parity with build detail workflows)

0. Quarantine detail log parity with build details (live stream + execution/lifecycle tabs)
- Status: `done`
- Priority: `P0`
- Owner: `Backend + Frontend`
- Problem: quarantine detail logs were fetch-only and sparse compared to build details, with no live stream and no clear execution-vs-lifecycle separation.
- Outcome target:
  - add tenant/admin quarantine log stream endpoints (`/logs/stream`) with same auth/RBAC model as existing quarantine detail routes.
  - expose live stream state and tabbed `Execution` / `Lifecycle` logs on quarantine detail page.
  - keep REST log endpoint as fallback/history source while stream appends incremental entries.
- Validation target:
  - backend route security contract includes both new stream endpoints.
  - frontend quarantine detail page receives stream entries and appends into tab-specific log panes.
- Source: quarantine UX parity request (build-details-style workflow progression + logs)

0. Quarantine Phase G admission guardrails: build intake enforcement (G-01 slice)
- Status: `done`
- Priority: `P0`
- Owner: `Backend`
- Problem: deployment/build intake allowed non-released quarantine image references, creating runtime risk despite release governance.
- Outcome target:
  - enforce released-only admission for quarantine artifact refs in build create preflight.
  - return deterministic deny contract for unreleased refs (`409`, `code=quarantine_artifact_not_released`).
  - wire checker from image-import persistence to avoid frontend-only gating.
- Validation target:
  - `go test ./internal/adapters/primary/rest -run "TestCreateBuild_QuarantineBaseImage(NotReleased|Released)_"`.
- Source: `docs/implementation/IMAGE_QUARANTINE_PROCESS_IMPLEMENTATION_PLAN.md` (Phase G kickoff)

1. Quarantine Phase A: admission skeleton + mock SOR validation adapter
- Status: `done`
- Priority: `P0`
- Owner: `Backend`
- Problem: quarantine request flow needs SOR prerequisite enforcement before approval side effects.
- Outcome target:
  - add minimal `external_image_imports` create/list/get with required `sor_record_id`
  - add SOR client interface + mock HTTP adapter using existing external service config pattern
  - deny request with deterministic `412 sor_registration_required` contract
- Validation target: `go test` for imageimport domain/repository + REST contract tests for SOR allow/deny cases.
- Source: `docs/implementation/IMAGE_QUARANTINE_PROCESS_IMPLEMENTATION_PLAN.md`

2. Quarantine Phase B: capability + approval workflow gate chain
- Status: `done`
- Priority: `P0`
- Owner: `Backend`
- Problem: capability checks and SOR checks must both gate request admission consistently for create/retry.
- Outcome target:
  - enforce chain: auth -> RBAC -> capability -> SOR -> approval create
  - ensure no approval records are created on denied requests
  - add audit entries for denial reasons
- Validation target: admission chain tests (create + retry) and approval side-effect absence assertions.
- Source: `docs/implementation/IMAGE_QUARANTINE_PROCESS_IMPLEMENTATION_PLAN.md`

3. Quarantine Phase C: Tekton quarantine MVP orchestration
- Status: `done`
- Priority: `P1`
- Owner: `Backend + Platform/Ops`
- Problem: approved imports need deterministic Tekton execution and result ingestion.
- Outcome target:
  - orchestrate approved imports into quarantine pipeline
  - persist scan/SBOM evidence and terminal import status
  - emit completion/quarantine/failure notifications
- Validation target: domain orchestration tests + local/staging end-to-end execution proof.
- Source: `docs/implementation/IMAGE_QUARANTINE_PROCESS_IMPLEMENTATION_PLAN.md`

4. Quarantine Phase D: tenant/admin UX + operability hardening
- Status: `done`
- Priority: `P1`
- Owner: `Frontend + Backend`
- Problem: users/admins need clear SOR prerequisite guidance and actionable failure reasons.
- Outcome target:
  - add SOR input/validation UX in import request flow
  - add status surfaces for denied vs runtime-failed states
  - add metrics + runbook for supportability
- Validation target: focused frontend tests for error states and backend metrics/event validation.
- Source: `docs/implementation/IMAGE_QUARANTINE_PROCESS_IMPLEMENTATION_PLAN.md`

5. Quarantine capability-first UX journey execution (D-01..D-09)
- Status: `done`
- Priority: `P1`
- Owner: `Frontend + Backend + QA`
- Problem: entitlement model exists but end-user/admin UX needs deterministic capability-scoped navigation, route-guarding, and denial guidance.
- Outcome target:
  - implement ticketized Phase D backlog from quarantine plan (`D-01` through `D-09`) (`D-01`, `D-02`, `D-03`, `D-04`, `D-05`, `D-06`, `D-07`, `D-08`, `D-09` complete; rollout evidence + signoff pending)
  - ensure tenant login surfaces only entitled capabilities (`build`, `quarantine_request`, `ondemand_image_scanning`)
  - ship deny-path consistency across UI + API (`tenant_capability_not_entitled`, `sor_registration_required`)
- Validation target: journey matrix coverage for entitled/denied permutations (login visibility, direct URL guard, create/retry API denials, telemetry/audit).
- Source: `docs/implementation/IMAGE_QUARANTINE_PROCESS_IMPLEMENTATION_PLAN.md`; `docs/user-journeys/QUARANTINE_CAPABILITY_JOURNEY.md`

6. Quarantine Phase C completion: approved import orchestration and evidence closure
- Status: `done`
- Priority: `P0`
- Owner: `Backend + Frontend + Platform/Ops + QA`
- Problem: approved quarantine imports still need production-grade Tekton execution closure with deterministic evidence persistence and operability.
- Outcome target:
  - harden approved import -> Tekton orchestration path with explicit retry/error contracts
  - persist and surface terminal evidence payloads (scan + SBOM + import status) for quarantine imports
  - expose clear runtime-failure diagnostics in tenant/admin UI (distinct from entitlement/SOR denials)
  - publish rollout runbook checks and alert/metric baselines for quarantine runtime health
- Validation target:
  - backend: orchestration/evidence/retry tests for success + failure paths
  - frontend: status/diagnostic rendering tests for denied vs runtime-failed vs completed states
  - staging: documented end-to-end proof run with evidence records and failure simulation
- Progress (2026-02-28):
  - Added manual approval gate scaffolding (removed auto-unblock of `approval.decision`; explicit decision now required in handler).
  - Added quarantine approval/reject APIs (`/api/v1/images/import-requests/{id}/approve|reject`) and workflow-step decision queuing.
  - Added RBAC seed entries for central approval model: `Security Reviewer` role + `quarantine:approve|reject|read` permissions.
  - Added bootstrap seed for central `security_reviewers` system group under `sysadmin` tenant.
  - Hardened dispatch/retry contract slice with tests: dispatch failures persist deterministic `dispatch_failed:` error prefix, API maps these to `sync_state=dispatch_failed` + `retryable=true`, dispatch step replay from terminal `failed` state is side-effect free (no duplicate dispatch/event/status writes), and `quarantined` terminal responses now correctly advertise `retryable=true` to match retry endpoint semantics.
  - Started C-02 monitor progression coverage: added integration tests for blocked `importing` monitor states (pipeline still running and missing pipeline refs) plus terminal mapping to `quarantined` and `failed`, while preserving persisted pipeline refs (`pipeline_run_name`, `pipeline_namespace`) across retries and terminalization.
  - Started C-03 evidence API stability hardening: terminal import responses now normalize evidence fields to deterministic non-empty fallbacks when pipeline outputs are missing (`policy_decision`, policy/scan/SBOM JSON, source digest), with mapper tests and handler-level list/get API tests asserting fallback presence and non-override of persisted non-empty values.
  - Added DSN-gated monitor integration for partial-evidence catalog projection fallback: SBOM projects from `sbom-summary` when `sbom-evidence` is absent, vulnerability scan rows are skipped when `scan-summary` is missing, and metadata freshness/digest fields remain deterministic (`sbom=fresh`, `vulnerability=unavailable`).
  - Added DSN-gated image-import handler integration coverage for terminal evidence fallback normalization on repository-backed GET/list endpoints when persisted evidence columns are NULL.
  - Added non-terminal guard coverage to ensure fallback evidence values are applied only after terminalization (`success|quarantined|failed`) and not while imports are still `pending|approved|importing`.
  - Started C-04/C-05 frontend diagnostics parity pass: shared import-diagnostics utility now drives sync-state labels (`dispatch_failed` support), diagnostic tone/summary messaging, and consistent detail/list diagnostics rendering across quarantine and on-demand request pages with explicit dark-mode-safe surfaces; evidence panels now suppress placeholder-only fallback JSON blobs.
  - Added quarantine workspace regression test coverage for `dispatch_failed` diagnostics visibility in status chips and detail drawer diagnostics panel.
  - Continued C-05 admin parity by adding admin routes + navigation for request workspaces (`/admin/quarantine/requests`, `/admin/images/scans`) and admin-aware image-catalog shortcut pathing, so tenant/admin contexts share the same lifecycle/diagnostics rendering contract.
  - Added frontend route-contract test coverage to pin admin import workspace route registration (`/admin/quarantine/requests`, `/admin/images/scans`) in `App.tsx`.
  - Started C-06 with committed staging runbook + validation template artifacts under `docs/qa/`:
    - `QUARANTINE_PHASE_C_STAGING_RUNBOOK.md`
    - `QUARANTINE_PHASE_C_VALIDATION_LOG.md`
  - Executed C-06 automated validation command set and recorded results in validation log: backend + frontend contract suites passed; DSN-gated staging-style integration targets are pending due missing `IF_TEST_POSTGRES_DSN` in current shell.
  - Started C-07 validation automation by adding executable runner `scripts/qa/quarantine_phase_c_validate.sh` and artifact output path `docs/qa/artifacts/`; hardened baseline check scope to Phase C contracts for sandbox stability and generated stable `status: OK` artifact `quarantine_phase_c_validation_20260228T224506Z.log`.
- Source: `docs/implementation/IMAGE_QUARANTINE_PROCESS_IMPLEMENTATION_PLAN.md`

7. Quarantine Phase E: release governance + controlled promotion
- Status: `done`
- Priority: `P0`
- Owner: `Backend + Frontend + Platform/Ops + QA`
- Problem: addressed; controlled promotion from quarantined artifacts to tenant-consumable references is now governed end-to-end.
- Outcome target:
  - introduce explicit release lifecycle (`ready_for_release`, `release_approved`, `released`, `release_blocked`) for quarantine imports/versions.
  - enforce `quarantine_release` capability + reviewer/policy preconditions before release.
  - ship release execution APIs/UI with idempotent retry and deterministic failure contracts.
  - expose only released artifacts to downstream tenant build/deploy selection paths.
  - emit release audit/notification events for full traceability.
- Validation target:
  - backend transition + idempotency + deny-side-effect tests.
  - frontend release panel/action tests for approve/release/failure diagnostics in light/dark.
  - staging runbook evidence for one successful release and one blocked release scenario.
- Completion note:
  - closed with runner artifact `docs/qa/artifacts/quarantine_phase_e_validation_20260302T223820Z.log` (`status: OK`, `pass=4`, `fail=0`, `skip=0`) and updated Phase E validation log signoff.
- Source: `docs/implementation/IMAGE_QUARANTINE_PROCESS_IMPLEMENTATION_PLAN.md`

7. Quarantine reviewer workbench (next functional slice, C-08)
- Status: `done`
- Priority: `P0`
- Owner: `Backend + Frontend`
- Problem: central security reviewers currently need fragmented context/action points; approval flow lacks a dedicated reviewer queue with in-page diagnostics/evidence preview.
- Outcome target:
  - add reviewer queue page for pending `external_image_import` approvals
  - include request diagnostics + normalized evidence preview in reviewer detail panel
  - wire approve/reject actions using existing endpoints with clear state refresh and decision feedback
- Validation target:
  - frontend tests for queue rendering, detail panel, approve/reject flows, and permission-denied state
  - backend route/permission contract tests for reviewer actions remain enforced (`quarantine:approve|reject|read`)
- Progress (2026-03-01):
  - Added frontend reviewer workbench page at `frontend/src/pages/admin/QuarantineReviewWorkbenchPage.tsx` with pending queue, detail drawer diagnostics, evidence preview, and approve/reject actions.
  - Added admin route wiring `/admin/quarantine/review` in `frontend/src/App.tsx` and admin navigation entry `Security Review Queue` in `frontend/src/components/layout/AdminLayout.tsx`.
  - Extended `imageImportService` with `approveImportRequest` and `rejectImportRequest` methods plus approval error-code mappings.
  - Added regression tests `frontend/src/pages/admin/__tests__/QuarantineReviewWorkbenchPage.test.tsx` and route-contract coverage update in `frontend/src/test/App.adminRoutes.contract.test.ts`.
  - Validation run passed: `npm run test -- --run src/pages/admin/__tests__/QuarantineReviewWorkbenchPage.test.tsx src/test/App.adminRoutes.contract.test.ts`.
  - Added reviewer-specific `403` permission-denied UX mapping in workbench load/action paths, including inline guidance with Capability Matrix link and normalized forbidden-action toast messaging.
  - Aligned backend import-request read routes to quarantine reviewer permission model by requiring `quarantine:read` on `GET /api/v1/images/import-requests` and `GET /api/v1/images/import-requests/{id}`; updated route security contract test to prevent regression.
  - Backend permission contract validation passed: `go test ./internal/adapters/primary/rest -run "TestRouter_AdminRoutePermissionContracts|TestImageImportHandlerApproveImportRequest_QueuesApprovalDecisionStep|TestImageImportHandlerRejectImportRequest_QueuesRejectedDecision"`.
  - Added DSN-gated essential seeder integration test (`cmd/essential-config-seeder`) verifying `ensureSystemBootstrap` idempotently seeds central `security_reviewers` system group in `sysadmin` tenant and preserves expected bootstrap group/membership counts.
  - Seeder + route-contract validation passed: `go test ./cmd/essential-config-seeder ./internal/adapters/primary/rest -run "TestEnsureSystemBootstrap_SeedsSecurityReviewersGroupIdempotently|TestRouter_AdminRoutePermissionContracts"`.
- Source: `docs/implementation/IMAGE_QUARANTINE_PROCESS_IMPLEMENTATION_PLAN.md`

8. Quarantine reviewer observability hardening (C-09)
- Status: `done`
- Priority: `P0`
- Owner: `Backend + Frontend + QA`
- Problem: reviewer queue needs stronger decision observability and faster triage surfaces (filters/search/timeline/reconciliation) beyond baseline approve/reject actions.
- Outcome target:
  - add reviewer queue filter/search controls for high-volume pending/terminal request review
  - expose decision-oriented timeline metadata and evidence checkpoints in reviewer details
  - define reconciliation checks (reviewer action vs notification/event completion)
- Validation target:
  - frontend tests for filter/search behavior and detail timeline rendering
  - backend contract tests for reviewer-facing event/timeline payloads where applicable
  - staged checklist updates for reconciliation verification
- Progress (2026-03-01):
  - Deployed C-08 closure to cluster (`helm` revision `54`) with backend/frontend tag `dbe2d1f`.
  - Started C-09 first slice by adding reviewer queue observability controls (search, status filter, filtered row count) in `QuarantineReviewWorkbenchPage`.
  - Added test coverage for queue search/status filter behavior in `QuarantineReviewWorkbenchPage.test.tsx`.
  - Added backend reviewer timeline contract by exposing `decision_timeline` from `approval.decision` workflow step payload/status in import request list/get responses.
  - Added reviewer drawer "Decision Timeline" UI rendering (decision, step state, reviewer user id, decided timestamp) with fallback messaging when no decision exists.
  - Validation run passed: `go test ./internal/adapters/primary/rest -run "TestImageImportHandlerLoadDecisionTimeline_FromApprovalStepPayload|TestMapImportResponse_IncludesDecisionTimeline|TestRouter_AdminRoutePermissionContracts|TestImageImportHandlerApproveImportRequest_QueuesApprovalDecisionStep|TestImageImportHandlerRejectImportRequest_QueuesRejectedDecision"` and `npm run test -- --run src/pages/admin/__tests__/QuarantineReviewWorkbenchPage.test.tsx src/test/App.adminRoutes.contract.test.ts`.
  - Added `notification_reconciliation` contract in image import list/get responses (decision event type, idempotency key, expected recipients, receipt count, in-app count, delivery state) for approved/rejected decisions.
  - Added reviewer drawer reconciliation panel and test assertions for fallback and delivered-state rendering.
  - Published QA runbook checklist `docs/qa/QUARANTINE_REVIEWER_RECONCILIATION_CHECKLIST.md` for API/DB/UI consistency validation.
  - Added executable runner `scripts/qa/quarantine_reviewer_reconciliation_validate.sh` to capture approve/reject reconciliation evidence into timestamped logs under `docs/qa/artifacts/`.
  - Executed local approve/reject reconciliation verification on restarted backend with artifact `docs/qa/artifacts/quarantine_reviewer_reconciliation_20260301T203421Z.log` (`status: OK`, API and DB parity checks passed for approved/rejected imports).
  - Closed C-09 functional scope (filter/search + decision timeline + notification reconciliation contract, UI, checklist, and validation runner).
- Source: `docs/implementation/IMAGE_QUARANTINE_PROCESS_IMPLEMENTATION_PLAN.md`

## Recently Completed

1. Admin viewer role routing + read-only UX hardening (`D-08`) and essential seed updates
- Status: `done`
- Priority: `P1`
- Owner: `Frontend + Backend`
- Outcome: system administrator viewer users now land on admin dashboard and see read-only admin UX (mutating actions hidden); essential seed/config updates merged with this stream.
- Release marker: merged to `main` and tagged `v2026.02.28-admin-viewer-ux`.
- Validation: targeted frontend route/access/admin-page tests passed for permission management, system configuration, operational capabilities, and capability route guard paths.

1. Provider tenant namespace deprovision epic
- Status: `done`
- Priority: `P1`
- Owner: `Backend + Frontend`
- Outcome: delivered managed-provider tenant namespace teardown capability (`POST /api/v1/admin/infrastructure/providers/{id}/tenants/{tenant_id}/deprovision-namespace`) with safety checks, namespace deletion wait semantics, persisted result summary/status, and admin-provider-detail `Deprovision` UX action.
- Validation: `go test ./internal/domain/infrastructure/...` (backend) and frontend compile/type-check path via existing provider detail code integration.
- Source: `docs/implementation/PROVIDER_DEPROVISION_NAMESPACE_PLAN.md`

1. Notification Center websocket-driven unread updates
- Status: `done`
- Priority: `P0`
- Owner: `Frontend`
- Outcome: removed interval-based unread-count polling loop in notification center and refreshed unread/list state on websocket build events.
- Follow-up: consider a dedicated notification websocket channel for non-build notification types.

2. Dedicated notification websocket channel (backend + UI wiring)
- Status: `done`
- Priority: `P0`
- Owner: `Backend + Frontend`
- Outcome: added `/api/notifications/events` websocket channel, user/tenant-scoped notification broadcasts (`notification.created`, `notification.read`, `notification.read_all`) and later expanded delete-oriented events (`notification.deleted`, `notification.deleted_read`, `notification.deleted_bulk`) used by notification management flows.
- Validation: added targeted backend tests for auth guard, tenant/user scoping, and notification payload contract.

3. Tenant asset drift policy epic (Phases 1-4)
- Status: `done`
- Priority: `P1`
- Owner: `Backend + Frontend`
- Outcome: completed runtime policy controls, drift persistence/API/UI surfacing, watcher + targeted reconcile actions, and operability hardening (metrics, alerts, runbook).
- Validation: added and ran staging-validation coverage for watcher enabled/disabled behavior, long-running stale-set drift scans, and targeted reconcile under load.

4. Tenant-scoped tool enforcement hardening
- Status: `done`
- Priority: `P0`
- Owner: `Backend`
- Outcome: enforcement now resolves tenant-scoped tool availability before global fallback, including explicit support for tenant-scoped `container` and `nix` build methods.
- Validation: added domain and API coverage for tenant-scoped method enforcement (`service_tool_availability_scope_test.go`, `build_handler_test.go`).

5. Tenant-scoped tool management UI (Admin)
- Status: `done`
- Priority: `P0`
- Owner: `Frontend`
- Outcome: admin Tool Management now reliably supports tenant override vs global default scope switching with deterministic load/save behavior.
- Validation: added focused frontend test coverage for tenant/global load + save parameter behavior in `frontend/src/components/admin/__tests__/ToolAvailabilityManager.test.tsx`.

6. Build notifications epic completion
- Status: `done`
- Priority: `P0`
- Owner: `Backend + Frontend`
- Outcome: core notifications epic is closed with persisted trigger matrix, in-app durable feed, websocket updates, and channel routing in place.
- Validation: Playwright coverage in `frontend/e2e/12-build-notifications.spec.ts`, matrix validation tests in `frontend/src/components/projects/__tests__/ProjectNotificationTriggerMatrix.test.tsx`, and closure doc `docs/implementation/BUILD_NOTIFICATIONS_E2E_VALIDATION_RESULTS.md`.

7. Webhook-triggered build execution (project-level) with project sources pivot
- Status: `done`
- Priority: `P1`
- Owner: `Backend + Frontend`
- Outcome: delivered project sources model + CRUD/UI, build source/ref binding, source-aware inbound webhook dispatch, and receipt/dedupe diagnostics.
- Validation: backend regression passes (`go test ./...`) and implementation tracker closure in `docs/implementation/BUILD_WEBHOOK_TRIGGER_PLAN.md`.

8. Build execution evidence capture completeness (layers/SBOM/vulnerabilities/lineage)
- Status: `done`
- Priority: `P0`
- Owner: `Backend`
- Outcome: implemented end-to-end evidence capture/persistence by preserving method artifacts in build results, persisting completion artifact payloads in `build_executions`, and ingesting normalized evidence on successful completion into `build_artifacts`, `build_metrics`, `catalog_image_metadata`, `catalog_image_layers`, `catalog_image_sbom`, `sbom_packages`, and `catalog_image_vulnerability_scans` while keeping existing catalog image/version auto-ingest flow.
- Validation: `go test ./internal/domain/build ./internal/application/imagecatalog ./internal/adapters/secondary/postgres` and full backend regression `go test ./...`.
- Source: `docs/implementation/BUILD_EXECUTION_EVIDENCE_CAPTURE_PLAN.md`

9. Tool availability strict-key behavior for build methods
- Status: `done`
- Priority: `P1`
- Owner: `Backend + Frontend`
- Outcome: enforced strict tenant semantics for tool availability (`missing => false`), added tenant config build-method backfill migration, and normalized admin UI handling for partial payloads so save contracts always emit explicit key sets.
- Validation: `go test ./internal/domain/systemconfig`; `npm run test -- --run src/components/admin/__tests__/ToolAvailabilityManager.test.tsx`.
- Source: `docs/implementation/TOOL_AVAILABILITY_STRICT_KEY_PLAN.md`

## Next High-Value Functional Epic (Proposed)

1. Quarantine Phase L: deployment trust enforcement + runtime compliance
- Status: `planned`
- Priority: `P0`
- Owner: `Backend + Frontend + QA + Platform/Ops`
- Problem: phases `A-K` closed intake, approval, release, and evidence-readiness controls, but deployment/runtime paths still need end-to-end trust enforcement to prevent post-release drift or bypass.
- Outcome target:
  - enforce released-only + immutable digest admission across deployment/update entry points.
  - block withdrawn/superseded/unreleased refs with deterministic deny contracts and remediation UX.
  - add runtime compliance watcher for deployed-image vs approved-digest drift detection.
  - add compliance telemetry/alerts and actionable audit timelines for tenant/admin workflows.
- Validation target:
  - backend integration tests for allow/deny + no-side-effect deployment admission checks.
  - frontend tests for blocked-path diagnostics and remediation actions (light/dark).
  - runbook + validation artifact coverage for one allowed deployment and one blocked/drift scenario.
- Source: `docs/implementation/IMAGE_QUARANTINE_PROCESS_IMPLEMENTATION_PLAN.md` (post-Phase K next epic proposal)

2. Build policy rules productization (BP-1)
- Status: `planned`
- Priority: `P1`
- Owner: `Backend + Frontend + QA + Platform/Ops`
- Problem: build policies/rules currently exist as admin CRUD + demo-seeded placeholders, but policy contracts are not fully enforced in build admission/dispatch/runtime paths.
- Outcome target:
  - define policy schema/versioning and strict validation contracts (bounds, units, enums, cross-field rules).
  - enforce active policies in build create/retry/dispatch paths with deterministic deny/error codes.
  - add policy simulation/dry-run endpoint to evaluate manifests before submit.
  - replace static demo defaults with environment-aware seed/bootstrap policy profiles.
  - complete admin policy UX for edit/validate/audit history and dark-mode-safe error/focus/disabled states.
- Validation target:
  - backend contract tests for policy enforcement and no-side-effect denials.
  - frontend tests for policy edit/preview/error surfaces in light/dark.
  - QA runbook artifacts covering allow/deny/simulate flows and rollout safety checks.
- Source: `backend/migrations/037_build_policies.up.sql`, `backend/bootstrap/seed-demo-data.sql`, `frontend/src/pages/admin/AdminBuildPoliciesPage.tsx`

Sequencing decision:
- Complete Phase `L` first (`P0`) to close deployment/runtime trust risk, then execute build policy productization (`BP-1`, `P1`) immediately after.

10. Tenant-scoped build capabilities and entitlements
- Status: `done`
- Priority: `P1`
- Owner: `Backend + Frontend`
- Outcome: shipped tenant/global `build_capabilities` entitlement model, admin settings APIs/UI, build-create admission enforcement, and dispatcher/retry revalidation to prevent capability-denied queue/retry execution.
- Validation: `go test ./internal/domain/systemconfig`; `go test ./internal/domain/build`; `go test ./internal/application/dispatcher`; `npm run test -- --run src/components/admin/__tests__/ToolAvailabilityManager.test.tsx`.
- Source: `docs/implementation/TENANT_BUILD_CAPABILITIES_ENTITLEMENTS_PLAN.md`

11. Tenant dashboard realtime + widget redesign
- Status: `done`
- Priority: `P1`
- Owner: `Backend + Frontend`
- Outcome: completed dashboard migration from mock data to tenant-scoped live cards/widgets with websocket refresh, then closed aggregation hardening with dedicated summary/activity backend endpoints (`/api/v1/dashboard/tenant/summary`, `/api/v1/dashboard/tenant/activity`) and a two-call dashboard load path.
- Validation: `go test ./internal/adapters/primary/rest -run 'TestTenantDashboardHandler|TestFormatDashboardDuration'`; `npm run test -- --run src/pages/__tests__/DashboardPage.test.tsx`.
- Source: `docs/implementation/TENANT_DASHBOARD_REALTIME_PLAN.md`

12. Provider onboarding consolidation and gated orchestration
- Status: `done`
- Priority: `P1`
- Owner: `Backend + Frontend`
- Outcome: closed onboarding gating with structured build-preflight `409` payloads, persisted provider `blocked_by` gate keys through DB/API/UI, and added a single guided onboarding flow in provider details that drives prepare/readiness/tenant-namespace actions from one place.
- Validation: `go test ./internal/adapters/primary/rest -run 'TestListProviders_IncludesBlockedByGates|TestTenantHandler_CreateTenant_SystemAdminAutoTriggersTenantPrepare'`; `go test ./internal/domain/infrastructure -run 'TestUpdateProviderReadiness'`.
- Source: `docs/implementation/INFRA_PROVIDER_E2E_READINESS_PLAN.md`

13. Build orchestration Phase 5 hardening rollout
- Status: `done`
- Priority: `P2`
- Owner: `Backend + Frontend + Platform/Ops`
- Outcome: completed hardening UX/ops closure with admin dashboard websocket-driven pipeline refresh, build-details direct recovery actions for blocked control-plane steps, and rollout script preflight route-checklist guard.
- Validation: `npm --prefix frontend run test -- --run src/pages/builds/__tests__/BuildDetailPage.test.tsx`; `AUTH_TOKEN=<backend-issued-token> TENANT_ID=<tenant-id> scripts/build-orchestration-phase5-rollout-check.sh` (passed: dispatcher/orchestrator running, blocked_step_count=0).
- Source: `docs/implementation/BUILD_ORCHESTRATION_SOLID_REFACTOR_PLAN.md`

14. Tekton enhancements plan completion
- Status: `done`
- Priority: `P2`
- Owner: `Backend`
- Outcome: closed planned Tekton enhancement slices across readiness/installer traceability, profile-version-aware asset resolution, load/capability-aware provider fallback routing, and optional security stages (scan/SBOM/sign) with preflight-aware task checks.
- Validation: `go test ./internal/domain/build ./internal/domain/infrastructure ./internal/infrastructure/kubernetes -count=1`.
- Source: `docs/implementation/TEKTON_ENHANCEMENTS_IMPLEMENTATION_PLAN.md`

15. Build notifications hardening and replay tooling
- Status: `done`
- Priority: `P1`
- Owner: `Backend + Platform/Ops`
- Outcome: closed replay/hardening follow-up with tenant-admin replay endpoints for failed build-notification emails (`/api/v1/admin/tenants/{tenant_id}/notification-replay*`), execution-pipeline metrics exposure for build-notification subscriber counters, and runbook/alert guidance plus executable validation contract.
- Validation: `go test ./internal/adapters/primary/rest ./internal/application/buildnotifications ./internal/application/runtimehealth -count=1`; staging validation script `scripts/build-notification-replay-check.sh`.
- Source: `docs/implementation/BUILD_NOTIFICATIONS_DESIGN_IMPLEMENTATION_PLAN.md`

16. Build monitor event-driven Phase 7 production closure
- Status: `done`
- Priority: `P1`
- Owner: `Backend + Platform/Ops`
- Outcome: closed production-readiness gate with shared-DB multi-instance idempotency integration coverage (success + failure paths), outage/recovery replay-drain coverage, execution-pipeline metrics export operationalization, and executable rollout validation script for runtime health + metrics + optional outage/DB checks.
- Validation: `go test ./internal/application/build/steps ./internal/infrastructure/messaging ./internal/adapters/primary/rest -count=1`; `AUTH_TOKEN=<backend-issued-token> TENANT_ID=<tenant-id> scripts/build-monitor-phase7-rollout-check.sh`.
- Source: `docs/implementation/BUILD_MONITOR_EVENT_DRIVEN_NATS_PLAN.md`

17. Layer-level security evidence and Image Details UX completion
- Status: `done`
- Priority: `P0`
- Owner: `Backend + Frontend`
- Issue: image detail pages currently do not provide actionable per-layer drill-down for vulnerabilities/SBOM, and users cannot identify which layer introduced vulnerable content.
- Resolution: expand build/catalog evidence model and UI to support layer-aware security for each image:
  - persist richer layer metadata (history/config linkage, package/file attribution, source command where available)
  - persist vulnerability-to-layer mappings and SBOM package-to-layer mappings
  - expose layer evidence APIs for per-layer detail panels
  - update Image Details UI with expandable layer rows showing content summary, mapped vulnerabilities, and SBOM package slices
  - ensure Catalog Metadata digest/scan/SBOM sections consistently reflect latest stored evidence
- Outcome: persisted layer evidence (`history/source command/diff`), layer-package mappings, derived layer-vulnerability mappings, expanded image details API contracts, and Image Details expandable layer drill-down UX with dark-mode-safe evidence panels.
- Validation: `go test ./internal/application/imagecatalog ./internal/adapters/secondary/postgres ./internal/adapters/primary/rest -count=1`; `npm run build` (currently fails due unrelated pre-existing frontend TypeScript issues outside this backlog scope).
- Next step: track and burn down unrelated frontend compile debt separately from this completed backlog item.
- Source: `docs/implementation/LAYER_SECURITY_EVIDENCE_IMAGE_DETAILS_PLAN.md`

18. Runtime services dependency watcher + admin alerting
- Status: `done`
- Priority: `P1`
- Owner: `Backend + Platform/Ops + Frontend`
- Issue: control-plane/runtime dependencies can be down while only low-level logs indicate impact; admins need explicit degraded-state alerting for required services.
- Resolution: added a runtime dependency watcher that continuously evaluates required dependencies (database connectivity, dispatcher/orchestrator state, messaging outbox relay, provider/runtime watchers), publishes structured health state via `runtime_dependency_watcher`, and emits deduplicated admin in-app `system_alert` notifications on degradation/recovery transitions.
- Outcome: execution pipeline health/metrics now include dependency watcher status + counters (`runtime_dependency_*`), admin dashboard surfaces dependency alert banners, and tenant admins receive websocket-backed in-app alerts when dependency state degrades or recovers.
- Validation: `go test ./backend/cmd/server ./backend/internal/adapters/primary/rest -count=1` and frontend admin dashboard regression tests (where available) with dark-mode-safe alert panel rendering.
- Source: `docs/implementation/BUILD_MONITOR_EVENT_DRIVEN_NATS_PLAN.md`

19. Project source duplicate guard (`provider + repository + branch`) 
- Status: `done`
- Priority: `P1`
- Owner: `Backend + Frontend`
- Issue: users could create duplicate project sources for the same project with identical provider/repository/default-branch tuples, causing ambiguous source selection and webhook/source matching.
- Outcome: added backend duplicate detection in project source repository for create/update with REST `409 Conflict` responses, plus frontend pre-submit validation messaging in source drawer for immediate feedback.
- Validation: `go test ./internal/adapters/secondary/postgres ./internal/adapters/primary/rest -count=1`.
- Source: `docs/implementation/BUILD_WEBHOOK_TRIGGER_PLAN.md`

20. Project sources UX follow-through (remove legacy single-repo project cues)
- Status: `done`
- Priority: `P1`
- Owner: `Frontend`
- Issue: project sources pivot was functionally complete, but key project/build surfaces still implied a legacy one-project/one-repo model.
- Outcome: shipped source-centric UX pass across project/build flows: project detail/build pages now surface source context, project source CRUD moved to drawer-first flow with auth selection and branch probing (including public-repo branch fetch), create flow uses initial source setup semantics, and edit-from-detail is narrowed to basic project metadata.
- Validation: targeted UI verification in project detail, project builds list, global builds list, build detail, and source drawer create/edit paths.
- Source: `docs/implementation/BUILD_WEBHOOK_TRIGGER_PLAN.md`

21. Notification center management expansion (dedicated page + selective/bulk delete)
- Status: `done`
- Priority: `P1`
- Owner: `Backend + Frontend`
- Issue: notification dropdown-only UX did not scale for high-volume users and lacked selective cleanup controls.
- Outcome:
  - added dedicated notifications management page (`/notifications`) with pagination/filtering and per-item actions
  - limited header dropdown to latest 10 notifications with explicit “View all notifications” navigation
  - added selective delete, delete-read, and checkbox-based bulk delete flows
  - added backend bulk-delete endpoint (`POST /api/v1/notifications/delete-bulk`) with user/tenant/channel scoping and websocket event emission
- Validation: `go test ./internal/adapters/primary/rest -count=1`.

22. Profile dropdown dismissal hardening
- Status: `done`
- Priority: `P2`
- Owner: `Frontend`
- Issue: profile/avatar dropdown could remain open unless user explicitly clicked elsewhere.
- Outcome: dropdown now closes on outside click, focus change, `Escape`, and pointer leave from profile menu area (with short debounce to avoid hover flicker).
- Validation: targeted UI verification across profile menu open/close paths in light and dark modes.

23. Project webhook trigger management UX (project settings)
- Status: `done`
- Priority: `P1`
- Owner: `Frontend + Backend`
- Issue: project webhooks tab initially supported create/delete only and lacked receipts diagnostics + edit/toggle operations.
- Outcome:
  - added project-level trigger management APIs:
    - `GET /api/v1/projects/{projectID}/triggers`
    - `PATCH /api/v1/projects/{projectID}/triggers/{triggerID}`
  - expanded `ProjectWebhooksPanel` with:
    - trigger edit drawer
    - active/inactive toggle
    - provider-specific setup hints (GitHub/GitLab)
    - webhook receipts diagnostics table (`/api/v1/projects/{id}/webhook-receipts`)
  - added focused frontend tests for trigger drawer validation and list refresh flow.
- Validation: `go test ./internal/adapters/primary/http/handlers ./internal/domain/build ./internal/adapters/secondary/postgres ./internal/adapters/primary/rest -count=1`; `npm run test -- --run src/components/projects/__tests__/ProjectWebhooksPanel.test.tsx`.

24. Build/runtime startup version verification and commit identity logging
- Status: `done`
- Priority: `P1`
- Owner: `Backend + Platform/Ops`
- Issue: after backend restarts, operators could not reliably confirm whether runtime code matched latest expected commit.
- Outcome:
  - added startup build verification log path with explicit match/mismatch/unknown status using `IF_EXPECTED_BUILD_COMMIT`
  - improved startup metadata observability (`build_commit`, `build_source`, `build_fingerprint`) and mismatch warning contract
  - injected build metadata (`version/commit/time/dirty`) in local build commands and Docker build pipeline via `ldflags` + Docker build args
- Validation: `go test ./cmd/server ./internal/adapters/primary/rest ./internal/domain/build -count=1`.

25. Tenant-owner first-login onboarding tour with persisted profile preferences
- Status: `done`
- Priority: `P1`
- Owner: `Frontend + Backend`
- Issue: first-time tenant owners needed a guided, visual orientation for project setup and build flow.
- Outcome:
  - added guided onboarding tour modal with capability visuals + setup CTAs for tenant owners
  - moved tour state persistence from browser-local storage to user profile-backed preferences for cross-device consistency
  - introduced dedicated `user_profile_preferences` persistence model and profile API support for preference read/write
  - replaced patch-style user-column change with dedicated table migration strategy and compatibility migration
- Validation: `go test ./internal/adapters/primary/rest -count=1`; `npm run test -- --run src/pages/__tests__/DashboardPage.test.tsx`.

26. Professional guidance empty-states for projects/builds
- Status: `done`
- Priority: `P2`
- Owner: `Frontend`
- Issue: baseline empty-state messages were too terse and did not explain setup/build execution flow.
- Outcome:
  - upgraded Projects page zero-state with structured setup guidance (source/auth/config mode/build flow)
  - upgraded global Builds and Project Builds zero-states with execution/evidence guidance cards and direct next-step actions
  - preserved concise no-result messaging for filtered/search states
- Validation: targeted frontend regression pass on routed dashboard/layout path (`npm run test -- --run src/pages/__tests__/DashboardPage.test.tsx`).

27. Build-as-code (`image-factory.yaml`) diagnostics and operability closure
- Status: `done`
- Priority: `P1`
- Owner: `Backend + Frontend + Platform/Ops`
- Issue: repo-config parse/policy failures were visible in start/retry API errors but not clearly surfaced in trace UI, and image-catalog ingest counters were not exposed in execution-pipeline health/metrics.
- Outcome:
  - persisted repo-config failure diagnostics into build manifest metadata during start/retry failures (`repo_config_error*`) and included structured `diagnostics.repo_config` in build trace API
  - surfaced repo-config diagnostics in Build Details trace timeline so failures are visible without relying on manual API inspection
  - exposed image-catalog subscriber component health and counters via execution-pipeline health/metrics (`image_catalog_event_subscriber`, `image_catalog_*` metrics)
  - expanded build-as-code guide with a concrete troubleshooting flow covering trace diagnostics, project build settings, runtime metrics, onboarding tour, and empty-state guidance cards
- Validation: `go test ./internal/adapters/primary/rest ./internal/domain/build -count=1`; frontend trace flow validated through existing build details page behavior.

28. REST router modularization (`router.go` decomposition)
- Status: `done`
- Priority: `P1`
- Owner: `Backend`
- Outcome: completed route registration decomposition, setup wiring extraction, and module-boundary contract coverage; `router.go` reduced to orchestration-focused shape.
- Validation: `go test ./internal/adapters/primary/rest -count=1`; `go test ./...` (pass, 2026-02-20).

29. REST router modularization Phase 4 (composition + regression gate)
- Status: `done`
- Priority: `P1`
- Owner: `Backend`
- Outcome: finalized setup composition boundary and regression gating for modular router architecture without API contract changes.
- Validation: `go test ./internal/adapters/primary/rest -count=1`; `go test ./...` (pass, 2026-02-19).

30. Build domain/application modularization (`build.go` decomposition)
- Status: `done`
- Priority: `P1`
- Owner: `Backend`
- Outcome: completed core modularization for build handler/service/aggregate boundaries; moved remaining deep `method_tekton_executor.go` split to a separate follow-on item.
- Validation: `go test ./internal/domain/build -count=1`; `go test ./internal/adapters/primary/rest -count=1`; `go test ./...` (pass, 2026-02-20).

31. Build Tekton executor deep decomposition follow-on
- Status: `done`
- Priority: `P2`
- Owner: `Backend`
- Outcome: decomposed `method_tekton_executor.go` into focused modules for preflight/secret reconciliation, monitoring/finalization/artifact collection, and template definitions while preserving behavior parity.
- Validation: `go test ./internal/domain/build -count=1`; `go test ./internal/adapters/primary/rest -count=1`; `go test ./internal/application/build -count=1`; `go test ./...` (pass, 2026-02-19).

32. Image quarantine implementation Phase 1 delivery (runtime + policy APIs)
- Status: `done`
- Priority: `P1`
- Owner: `Backend + Frontend`
- Plan doc: `docs/implementation/IMAGE_QUARANTINE_PROCESS_IMPLEMENTATION_PLAN.md`
- Scope:
  - implement `quarantine_policy` config model + admin GET/PUT/validate/simulate endpoints.
  - implement import-request domain/repo/REST skeleton with entitlement gate enforcement.
  - add dispatcher/orchestrator claim/dispatch path for approved import subjects.
  - add watcher/subscriber terminal-state persistence + policy snapshot/reason persistence.
  - add initial admin UI policy editor and simulation panel.
- Acceptance gate:
  - policy APIs and import APIs pass contract tests. ✅
  - approved import creates PipelineRun and reaches terminal persisted state with policy decision metadata. ✅
  - terminal events/notifications include policy snapshot + reason fields per contract. ✅
- Next step: move to deferred QA/stabilization backlog execution and Phase 2+/canary hardening slices.

33. Image quarantine process implementation (epic closure)
- Status: `done`
- Priority: `P1`
- Owner: `Backend + Frontend + Platform/Ops`
- Plan doc: `docs/implementation/IMAGE_QUARANTINE_PROCESS_IMPLEMENTATION_PLAN.md`
- Outcome:
  - completed capability/SOR-gated admission chain with deterministic deny contracts (`403 tenant_capability_not_entitled`, `412 sor_registration_required`) and no-side-effect guarantees.
  - completed quarantine policy + SOR policy admin APIs/UI, approval->dispatch->monitor lifecycle, evidence projection, runtime-services hardening, and tenant-facing SOR posture clarity.
  - completed stabilization backlog closeout and canary-readiness summary publication (`docs/qa/QUARANTINE_STABILIZATION_BACKLOG.md`, `docs/qa/QUARANTINE_CANARY_READINESS_SUMMARY.md`).
- Validation:
  - one-pass stabilization automation sweep passed (`8` files / `24` tests) across capability visibility, route guards, import activity, scan UX, and SOR posture messaging.
  - targeted functional API/domain/UI suites and Tekton/runtime validation listed in plan/backlog logs.
- Follow-up:
  - optional pre-prod light/dark smoke screenshots can be tracked as fit-and-finish without reopening this epic.

35. Access Management operational capabilities surface (dedicated admin UX)
- Status: `done`
- Priority: `P1`
- Owner: `Frontend + Backend`
- Plan doc: `docs/implementation/ACCESS_MANAGEMENT_OPERATIONAL_CAPABILITIES_PLAN.md`
- Outcome:
  - added dedicated `Access Management -> Operational Capabilities` navigation and admin route (`/admin/access/operational-capabilities`).
  - implemented tenant capability matrix management (tenant selection, toggle matrix, required change reason, reset/save feedback) using existing operation capability APIs.
  - added focused coverage for load/switch/validation/save paths in `OperationalCapabilitiesPage` tests.
  - updated admin/user quarantine documentation to reflect the new primary management surface while keeping tenant edit as shortcut.
- Validation:
  - `npm run test -- --run src/pages/admin/__tests__/OperationalCapabilitiesPage.test.tsx src/pages/admin/__tests__/PermissionManagementPage.test.ts`.
- Follow-up:
  - optional enhancement: add an audit-history panel (last changed by/when/reason) on the operational capabilities page.

36. Prototype cleanup: remove legacy `external_image_import` capability
- Status: `done`
- Priority: `P0`
- Owner: `Backend + Frontend + QA`
- Plan doc: `docs/implementation/IMAGE_QUARANTINE_PROCESS_IMPLEMENTATION_PLAN.md`
- Outcome:
  - completed backend-driven capability surface gating for tenant nav/route/action visibility.
  - completed canonical capability model cleanup (`build`, `quarantine_request`, `quarantine_release`, `ondemand_image_scanning`) across active backend/frontend contracts.
  - completed docs/QA realignment so active entitlement guidance is canonical; legacy `external_image_import*` references remain only for persisted entities/events where applicable.
- Validation:
  - backend: `go test ./internal/domain/systemconfig ./internal/adapters/secondary/policy ./internal/adapters/primary/rest -run "CapabilitySurfaces|OperationCapabilities|ImageImport|SystemConfigHandler|OperationCapabilityChecker"`
  - frontend: targeted capability gating suites for layout/routes/dashboard/images/admin capability UX.

## Active backlog

36. SRE Smart Bot async backlog and transport pressure signals
- Status: `done`
- Priority: `P1`
- Owner: `Backend + Frontend + Platform/Ops`
- Plan doc: `docs/implementation/SRE_SMART_BOT_ASYNC_BACKLOG_TRANSPORT_PRESSURE_EPIC.md`
- Problem:
  - async queue depth and messaging transport instability are now observable, but they are not yet productized as first-class incident findings, summary cards, and deterministic operator guidance.
- Outcome target:
  - turn async backlog pressure into normalized SRE findings and evidence.
  - surface backlog and transport pressure directly in the incident summary UX.
  - correlate backlog growth with transport instability in the deterministic draft.
  - prepare the same model for later true NATS lag and consumer-pressure coverage.
- Validation:
  - backend: `cd backend && go test ./internal/application/sresmartbot -run 'Test(BuildDraft|DemoService|ObserveAsyncBacklogSignals_)'`
  - frontend: `cd frontend && npm test -- --run src/pages/admin/__tests__/sreSmartBotAsyncSummary.test.ts`

40. SRE Smart Bot NATS consumer lag and pressure signals
- Status: `done`
- Priority: `P1`
- Owner: `Backend + Frontend + Platform/Ops`
- Plan doc: `docs/implementation/SRE_SMART_BOT_NATS_CONSUMER_LAG_PRESSURE_EPIC.md`
- Problem:
  - current messaging observability stops at transport reconnect and disconnect pressure, so operators still cannot tell which NATS consumer is lagging or whether lag explains async backlog growth.
- Outcome target:
  - turn consumer lag into normalized SRE findings and evidence.
  - surface consumer lag and stalled progress directly in the incident summary UX and AI workspace.
  - correlate consumer lag with transport instability and async backlog growth in the deterministic draft.
  - prepare the same model for deeper stream-level delivery telemetry later.
- Outcome:
  - consumer lag and stalled-progress signals now normalize into incident ledger findings/evidence with deterministic correlation keys.
  - workspace and deterministic draft outputs now include consumer-pressure context and tool summaries (`messaging_consumers.recent`).
  - consumer pressure is now correlated with transport instability and async backlog context in SRE incident interpretation flows.
- Validation:
  - backend: `cd backend && go test ./internal/application/sresmartbot -run 'TestObserveNATSConsumerLagSignals_|TestBuildIncidentWorkspace_IncludesMessagingConsumerSummaryAndBundle|TestBuildDraftSummary_ExplainsConsumerLagWithoutTransportInstability|TestBuildDraftHypotheses_DistinguishesConsumerPressureFromTransport'`
  - frontend: `cd frontend && npm test -- --run src/pages/admin/__tests__/SRESmartBotIncidentsPage.test.tsx src/pages/admin/__tests__/sreSmartBotAsyncSummary.test.ts`

### SRE Smart Bot Handoff Checkpoint

- Date: `2026-03-28`
- Branch: `feature/sre-smartbot-phase1-loki`
- Status:
  - policy foundation is complete.
  - incident ledger and core watcher/detector wiring are complete for current product scope.
  - startup/watcher modularization checkpoint is complete.
  - admin incidents, approvals, settings, detector-rule suggestions, and remediation-pack flows are available.
- Immediate next product step:
  - continue observability/intelligence plumbing (Loki ingestion parity, detector contract hardening, MCP seams).
  - improve operator-readable evidence/action summaries and add the next low-risk allowlisted executable action.

37. Quarantine telemetry expansion (runtime mode + scope analytics)
- Status: `done`
- Priority: `P1`
- Owner: `Backend + Platform/Ops`
- Plan doc: `docs/implementation/IMAGE_QUARANTINE_PROCESS_IMPLEMENTATION_PLAN.md`
- Outcome:
  - SOR denial metrics include explicit `sor_runtime_mode` and `sor_policy_scope` label dimensions.
  - admin stats/dashboard surface breakdowns for SOR denial telemetry to support policy tuning.
  - focused metric tests cover labeled emission grouping behavior.
- Validation:
  - `go test ./internal/infrastructure/denialtelemetry ./internal/adapters/primary/rest -run "Metrics_RecordDenied|ImageImportHandlerCreateImportRequest_SORDenied_RecordsSORPolicyLabels|SystemStats"`

38. Quarantine UX redesign (capability-first tenant experience)
- Status: `in_progress`
- Priority: `P1`
- Owner: `Frontend + Backend`
- Plan doc: `docs/implementation/IMAGE_QUARANTINE_PROCESS_IMPLEMENTATION_PLAN.md`
- Next step:
  - complete route-level deny-state parity checks for remaining capability-gated tenant routes.
  - align deny/empty-state copy so tenants with zero capabilities get consistent guidance across dashboard, images, and quarantine routes.

39. Air-gapped Tekton task image configuration (admin-managed)
- Status: `in_progress`
- Priority: `P1`
- Owner: `Backend + Frontend + Platform/Ops`
- Plan doc: `docs/implementation/AIR_GAPPED_TEKTON_TASK_IMAGE_CONFIGURATION_PLAN.md`
- Outcome target:
  - support global admin-managed Tekton task image overrides (`tekton_task_images`) for air-gapped deployments.
  - apply image overrides consistently across tenant namespace prepare, provider prepare, and installer apply flows.
  - expose/administer image overrides in System Configuration -> Tekton tab.
- Progress update:
  - backend model/services/APIs complete (`GET/PUT /api/v1/admin/settings/tekton-task-images`).
  - backend handler tests added for tekton task images GET/PUT + invalid payload behavior (`systemconfig_handler_test.go`).
  - infrastructure reconcile/install integration complete (manifest image rewrite + drift hash awareness).
  - frontend Tekton tab form + typed service wiring in place, with improved save-time validation error messaging.
  - frontend component-level tests added for Tekton task image load/edit/save and save-error messaging (`SystemConfigurationPage.test.tsx`).
  - staging failure investigation for build `e2e8a942-124b-4b66-8cee-abb8303c1e87` isolated buildx direct-push mismatch (`https` attempted against HTTP internal registry).
  - buildx pipeline retained direct push and buildx task hardened for local registry direct push (`registry.http=true` + `registry.insecure=true`), with registry-based scan/sbom/final push wiring.
- Next step:
  - persist `tekton_task_images` config in staging and mirror required Tekton runtime images to internal registry.
  - execute staging rollout validation across build methods (`docker`, `kaniko`, `buildx`) in internal-registry-only mode and capture evidence logs.
- Acceptance checklist:
  - [x] Admin can read/update `tekton_task_images` via `/api/v1/admin/settings/tekton-task-images`.
  - [x] System Configuration -> Tekton tab persists all task image fields and reloads saved values.
  - [ ] Provider prepare applies Tekton assets with configured image overrides (no public registry image refs remain in applied Task/Pipeline specs).
  - [ ] Tenant namespace prepare applies Tekton assets with configured image overrides.
  - [ ] Drift desired-version changes when image override values change.
  - [ ] Staging validation passed for `docker`, `kaniko`, and `buildx` build methods in internal-registry-only mode.
  - [ ] Build logs for staging run show no external image pull attempts for Tekton task runtime images.

40. Internal registry temp image garbage collection worker
- Status: `in_progress`
- Priority: `P1`
- Owner: `Backend + Frontend + Platform/Ops`
- Plan doc: `docs/implementation/INTERNAL_REGISTRY_TEMP_IMAGE_GC_PLAN.md`
- Outcome target:
  - add standalone `internal-registry-gc-worker` that periodically deletes expired temp scan images from internal registry.
  - make cleanup interval/retention/batch + worker endpoint settings configurable via admin runtime services config.
  - expose worker health in Admin Dashboard system component checks.
- Next step:
  - implement Phase 1 worker loop + health endpoint and wire runtime_services/admin dashboard fields.

41. On-demand external scan requests (standalone tenant workflow)
- Status: `in_progress`
- Priority: `P1`
- Owner: `Backend + Frontend`
- Plan doc: `docs/implementation/ON_DEMAND_EXTERNAL_SCAN_REQUESTS_PLAN.md`
- Outcome target:
  - provide dedicated tenant `On-Demand Scans` workflow for external image submission and async processing, separate from Image Catalog browsing.
  - persist and display scan request history/results with retry and notification visibility.
  - enforce `ondemand_image_scanning` capability explicitly for scan-request APIs and UI route/nav.
- Progress update:
  - added `request_type` model/migration (`quarantine` vs `scan`) and scan-specific API routes under `/api/v1/images/scan-requests`.
  - added dedicated tenant route/page `/images/scans` and moved nav target from `/images` to `/images/scans`.
  - reused existing async pipeline/notification flow with scan-specific messaging and request-type payload context.
- Next step:
  - add focused contract tests for scan-request endpoints (create/list/get/retry) and UI tests for `OnDemandScansPage` submit/history/error/denied states.

42. Build node provider provisioning parity (replace mock flow)
- Status: `planned`
- Priority: `P1`
- Owner: `Backend + Frontend + Platform/Ops`
- Plan doc: `docs/implementation/BUILD_NODE_PROVIDER_PREPARE_PLAN.md`
- Outcome target:
  - implement first-class `build_nodes` provider provisioning lifecycle with persisted prepare runs/checks and readiness/schedulable gating.
  - provide build-node specific preflight checks (`connectivity`, `auth`, `runtime`, `storage`, `capacity`, `queue_binding`) with deterministic remediation guidance.
  - reuse provider prepare status/stream UX to surface provisioning progress and terminal state for build-node providers.
- Next step:
  - implement backend prepare orchestration branch for `ProviderTypeBuildNodes` reusing `provider_prepare_runs` + checks repository contract.

43. Runtime asset storage profiles (admin-managed)
- Status: `in_progress`
- Priority: `P1`
- Owner: `Backend + Frontend + Platform/Ops`
- Plan doc: `docs/implementation/RUNTIME_ASSET_STORAGE_PROFILES_PLAN.md`
- Outcome target:
  - centralize storage profile configuration for runtime/bootstrap assets under `runtime_services.storage_profiles`.
  - allow admins to choose `hostPath`, `PVC`, or `emptyDir` for internal registry bootstrap assets from System Configuration UI.
  - ensure provider prepare/bootstrap apply paths mutate/skip storage manifests deterministically and avoid mixed volume-type errors.
- Next step:
  - complete phase 1 validation across provider prepare flows for `hostPath` and `PVC`, then extend profile enforcement to remaining storage-backed runtime assets.

50. Quarantine auto-dispatch on Tekton readiness (provider/runtime-aware)
- Status: `in_progress`
- Priority: `P0`
- Owner: `Backend + Platform/Ops`
- Plan doc: `docs/implementation/IMAGE_QUARANTINE_PROCESS_IMPLEMENTATION_PLAN.md`
- Outcome target:
  - keep approved quarantine requests in deterministic `awaiting_dispatch` state while Tekton dispatch path is unavailable.
  - automatically requeue blocked `import.dispatch` workflow steps when a Tekton-capable dispatcher becomes available.
  - avoid false terminal failures during temporary control-plane unavailability.
- Next step:
  - complete dynamic dispatcher attach + blocked-step requeue loop, then add regression coverage for blocked->pending replay semantics and publish validation evidence.

44. Quarantine release governance + controlled promotion (Phase E)
- Status: `done`
- Priority: `P0`
- Owner: `Backend + Frontend + Platform/Ops + QA`
- Plan doc: `docs/implementation/IMAGE_QUARANTINE_PROCESS_IMPLEMENTATION_PLAN.md`
- Outcome target:
  - implement governed promotion from quarantined artifacts to released, tenant-consumable image references.
  - enforce release preconditions (capability, policy, approval decision, immutable digest binding).
  - provide release API + reviewer/admin UI with deterministic retry and explicit diagnostics.
  - persist release timeline/audit data and emit release events for notification/reconciliation.
- Next step:
  - move to next major functional epic (Phase F) focused on post-release governance telemetry, alerting automation, and operational SLO instrumentation.
- Ticket breakdown:
  - `E-01` release-readiness API projection (done)
  - `E-02` release workflow transition model (done)
  - `E-03` release action APIs + capability enforcement (`quarantine_release`) (done)
  - `E-04` release event + audit timeline persistence (done)
  - `E-05` released-only artifact exposure enforcement (done)
  - `E-06` reviewer/admin release panel UX (done)
  - `E-07` staged runbook + validation artifact capture (done)
  - `E-08` hardening/alerts/reconciliation closure (done)

45. Post-release governance telemetry + operations (Phase F)
- Status: `in_progress`
- Priority: `P0`
- Owner: `Backend + Frontend + Platform/Ops + QA`
- Plan doc: `docs/implementation/PHASE_F_RELEASE_GOVERNANCE_TELEMETRY_PLAN.md`
- Outcome target:
  - operationalize quarantine release governance with first-class telemetry, alerting thresholds, and SLO-oriented visibility.
  - provide deterministic admin-facing release counters and failure alert pathways.
  - define executable QA validation artifacts for normal and degraded release-governance scenarios.
- Next step:
  - complete `F-06` hardening/closure notes and finalize merge readiness after full DSN-enabled Phase F runner pass (`pass=5 fail=0 skip=0`).
- Ticket breakdown:
  - `F-01` release telemetry counters + admin stats exposure (done)
  - `F-02` threshold configuration for release failure ratio/burst (done)
  - `F-03` release-failure alert emission + dedupe/recovery behavior (done)
  - `F-04` admin dashboard/reviewer telemetry + alert surfacing (in progress)
  - `F-05` Phase F validation runner + runbook artifacts (done)
  - `F-06` hardening + closure docs/signoff (in progress)

46. EPR-gated quarantine intake + reviewer approvals (Phase G)
- Status: `in_progress`
- Priority: `P0`
- Owner: `Backend + Frontend`
- Plan doc: `docs/implementation/IMAGE_QUARANTINE_PROCESS_IMPLEMENTATION_PLAN.md`
- Outcome target:
  - add stepper-driven tenant intake that enforces EPR registration decision before quarantine request submission.
  - implement EPR registration request lifecycle (`pending`, `approved`, `rejected`) with security reviewer actions.
  - emit transition notifications and allow approved EPR registrations to satisfy quarantine prerequisite checks.
- Next step:
  - complete admin reviewer UX polish (filters + reason capture), add REST handler contract tests for EPR registration endpoints, then run local E2E validation and phase signoff.
- Ticket breakdown:
  - `G-01` backend EPR registration request APIs + persistence + approval actions (in progress)
  - `G-02` EPR validator integration with approved registration override (in progress)
  - `G-03` tenant quarantine stepper UX + EPR registration form/status panel (in progress)
  - `G-04` reviewer queue integration for EPR approvals/rejections (in progress)
  - `G-05` transition notifications for EPR workflow events (in progress)

47. EPR lifecycle governance + revalidation (Phase H)
- Status: `in_progress`
- Priority: `P0`
- Owner: `Backend + Frontend + Platform/Ops`
- Plan doc: `docs/implementation/IMAGE_QUARANTINE_PROCESS_IMPLEMENTATION_PLAN.md`
- Outcome target:
  - make EPR approvals lifecycle-aware (`active`, `expiring`, `expired`, `suspended`) instead of effectively permanent.
  - block quarantine intake deterministically when EPR state is `expired` or `suspended`, with actionable denial guidance.
  - add reviewer/admin revalidation workflow with auditability, lifecycle notifications, and metrics.
- Why high value:
  - closes governance gap after Phase G by preventing stale EPR approvals from silently bypassing security policy.
  - directly reduces production-risk imports while preserving clear tenant/reviewer UX.
- Initial ticket slice:
  - `H-01` data model + policy fields (`approved_at`, `expires_at`, lifecycle status, suspension metadata). (`done`)
  - `H-02` admission enforcement for lifecycle states on quarantine create/retry. (`done`)
  - `H-03` reviewer/admin lifecycle actions (revalidate, suspend, reactivate) + reason capture. (`done`)
  - `H-04` scheduled lifecycle transitions + expiring/expired queues. (`done`)
  - `H-05` lifecycle notifications (expiring/expired/suspended/reactivated) + template wiring. (`done`)
  - `H-06` reviewer/admin UX for lifecycle queues, filters, and bulk actions. (`done`)
  - `H-07` telemetry + admin stats/dashboard slices for lifecycle posture and drift. (`done`)
  - `H-08` rollout validation runner, runbook closure, and regression signoff artifacts. (`done`)
- Next step:
  - Phase H closed. Propose next major functional epic after lifecycle governance closure (for example, tenant-facing release consumption self-service hardening).

48. Tenant release consumption self-service hardening (Phase I)
- Status: `done`
- Priority: `P0`
- Owner: `Backend + Frontend + QA + Platform/Ops`
- Plan doc: `docs/implementation/IMAGE_QUARANTINE_PROCESS_IMPLEMENTATION_PLAN.md`
- Outcome target:
  - provide a first-class tenant release-consumption journey from released quarantine artifacts into project build workflows.
  - emit deterministic consumption audit/telemetry events with project context.
  - enforce fail-closed behavior for non-consumable release states and preserve dark-mode UX parity.
- Initial ticket slice:
  - `I-01` released artifact API contract expansion (consumption-ready posture + pagination/search consistency). (`done`)
  - `I-02` tenant released-artifact workspace UX refresh (compact rows, provenance + posture cues). (`done`)
  - `I-03` use-in-project drawer flow (project picker + build wizard prefill). (`done`)
  - `I-04` build wizard intake support for released artifact prefill params. (`done`)
  - `I-05` release consumption event endpoint + audit payload (`quarantine.release_consumed`). (`done`)
  - `I-06` release consumption telemetry surfaced in admin stats. (`done`)
  - `I-07` Phase I validation runner + runbook + validation log artifacts. (`done`)
  - `I-08` closure/signoff and backlog handoff to next epic. (`done`)
- Next step:
  - propose and kick off next major functional epic (Phase J) focused on end-to-end quarantine workflow reliability and regression confidence.

49. End-to-end quarantine workflow reliability + regression confidence (Phase J)
- Status: `done`
- Priority: `P0`
- Owner: `Backend + Frontend + QA + Platform/Ops`
- Plan doc: `docs/implementation/IMAGE_QUARANTINE_PROCESS_IMPLEMENTATION_PLAN.md`
- Outcome target:
  - provide deterministic end-to-end path coverage for tenant intake -> reviewer decisions -> release consumption -> build handoff.
  - add regression guardrails for role-routing, permission filtering, notification delivery, and tenant/reviewer/admin dashboard routing.
  - eliminate recurring local-vs-cluster parity issues with executable validation gates and environment sanity checks.
- Initial ticket slice:
  - `J-01` quarantine happy-path E2E suite (tenant submit -> reviewer approve -> released artifact visible/consumable). (`done`)
  - `J-02` deny/withdraw/clone branch-path E2E suite with state assertions. (`done`)
  - `J-03` role-routing + dashboard landing contract hardening (backend-driven profile checks). (`done`)
  - `J-04` permission-scope contract tests for tenant/system roles and assignment APIs. (`done`)
  - `J-05` notification pipeline reliability checks (template seeded, recipient resolution, fallback escalation behavior). (`done`)
  - `J-06` local/cluster parity smoke runner (healthz, stats, setup-required, LDAP load-from-env sanity). (`done`)
  - `J-07` CI wiring for phased regression packs + artifact publication. (`done`)
  - `J-08` runbook/signoff + rollback/recovery playbook updates. (`done`)
- Next step:
  - Phase J closed on local parity baseline; next epic can focus on workflow productization or cluster hardening follow-through.

50. Quarantine dispatch resilience follow-up (J-09 closure)
- Status: `done`
- Priority: `P0`
- Owner: `Backend + Frontend + Platform/Ops`
- Plan doc: `docs/implementation/IMAGE_QUARANTINE_PROCESS_IMPLEMENTATION_PLAN.md`
- Outcome target:
  - keep approved quarantine requests in deterministic `awaiting_dispatch` posture when no eligible Tekton runtime exists.
  - auto-resume blocked dispatch without manual restarts once an eligible provider becomes available.
  - make provider eligibility explicit and operator-controlled (`tekton_enabled=true` + `quarantine_dispatch_enabled=true`).
- Completed in this slice:
  - provider-aware dispatcher detection wired to infrastructure providers with deterministic selection.
  - runtime contract updated so unavailable dispatch path blocks with `waiting_for_dispatch` instead of false terminal failure.
  - admin UI now exposes `quarantine_dispatch_enabled` alongside `tekton_enabled` in provider create/edit and detail views.
  - added dispatch replay regression test proving blocked `import.dispatch` can succeed on next execution once provider path becomes available.
  - improved `awaiting_dispatch` diagnostics copy with explicit provider-flag guidance for operators and tenant/reviewer users.
- Next step:
  - J-09 closed with parity artifact `docs/qa/artifacts/quarantine_phase_j_validation_20260304T203405Z.log`; start next functional epic (Phase K) focused on Tekton execution assurance + release-readiness from real pipeline evidence.

51. Quarantine pipeline execution assurance + release-readiness from evidence (Phase K)
- Status: `done`
- Priority: `P0`
- Owner: `Backend + Frontend + Platform/Ops + QA`
- Plan doc: `docs/implementation/IMAGE_QUARANTINE_PROCESS_IMPLEMENTATION_PLAN.md`
- Outcome target:
  - expose deterministic dispatch/runtime readiness contracts so operators can diagnose why quarantine jobs are blocked before tenant retries.
  - harden pipeline failure classification and retry semantics for quarantine imports.
  - gate release readiness on complete, current pipeline evidence (scan/SBOM/policy) with explicit triage UX.
- Initial ticket slice:
  - `K-01` provider dispatch-readiness contract API with explicit blocker reasons. (`done`)
  - `K-02` quarantine execution state/timestamp contract expansion (`awaiting_dispatch` -> `pipeline_running` -> `evidence_pending` -> `ready_for_release`). (`done`)
  - `K-03` normalized Tekton failure classification and deterministic error codes. (`done`)
  - `K-04` retry policy engine (bounded attempts + backoff by failure class). (`done`)
  - `K-05` reviewer/admin triage drawer with root-cause guidance and actions. (`done`)
  - `K-06` release-readiness gate from evidence completeness/freshness. (`done`)
  - `K-07` notification/escalation wiring for failure classes. (`done`)
  - `K-08` validation runner + runbook closure artifacts. (`done`)
- Next step:
  - Phase K closed with zero-skip validation artifact (`quarantine_phase_k_validation_20260304T213323Z.log`); prepare next functional epic proposal.

52. Deployment trust enforcement + runtime compliance (Phase L)
- Status: `in_progress`
- Priority: `P0`
- Owner: `Backend + Frontend + Platform/Ops + QA`
- Plan doc: `docs/implementation/IMAGE_QUARANTINE_PROCESS_IMPLEMENTATION_PLAN.md` (to be extended with Phase L workboard)
- Outcome target:
  - enforce released-only deployment/update admission with immutable digest trust checks.
  - detect and surface runtime drift between deployed artifacts and approved released digests.
  - provide deterministic remediation and auditability for tenant/admin operators.
- Initial ticket slice:
  - `L-01` deployment/update admission contract expansion for released-only + digest pinning. (`done` on available build intake/update surfaces)
  - `L-02` withdrawn/superseded deny-path hardening with deterministic API/UX remediation contracts. (`done`)
  - `L-03` runtime compliance watcher + drift event publication. (`done`)
  - `L-04` drift triage/remediation UX in tenant/admin views with light/dark parity.
  - `L-05` compliance telemetry + alert thresholds in admin stats/dashboard.
  - `L-06` Phase L validation runner + runbook + closure artifacts.
- Next step:
  - execute `L-04` drift triage/remediation UX and connect compliance metrics to admin dashboard slices.

53. Build policy rules productization (BP-1)
- Status: `planned`
- Priority: `P1`
- Owner: `Backend + Frontend + QA + Platform/Ops`
- Plan doc: `docs/implementation/BUILD_POLICY_RULES_PRODUCTIZATION_PLAN.md`
- Outcome target:
  - move build policy support from demo-seeded configuration to production-grade enforced policy controls.
  - apply policy checks deterministically at build intake/retry/dispatch with explicit remediation contracts.
  - provide admin policy authoring UX with validation, simulation, and auditability.
- Initial ticket slice:
  - `BP-01` policy schema + validator hardening (typed values, ranges, units, enums, cross-field invariants).
  - `BP-02` build admission/retry enforcement wiring with deterministic error codes and no-side-effect denial behavior.
  - `BP-03` dispatch/runtime guardrails for policies that depend on provider/runtime capabilities.
  - `BP-04` policy simulation endpoint (`/validate` or `/simulate`) for pre-submit manifest checks.
  - `BP-05` admin policy UX completion (create/edit/disable/history) with dark-mode parity and focused test coverage.
  - `BP-06` seed/bootstrap profile strategy (dev/demo/prod-safe defaults) and integrity checks.
  - `BP-07` validation runner + rollout runbook + closure artifacts.
- Next step:
  - start `BP-01` validator contract hardening once Phase L `L-04/L-05` slices are stable.
- Execution board:
  - [ ] `BP-01` schema + validator hardening
  - [ ] `BP-02` create/retry admission enforcement
  - [ ] `BP-03` dispatch/runtime guardrails
  - [ ] `BP-04` simulation endpoint + parity fixtures
  - [ ] `BP-05` admin policy UX (create/edit/disable/history/simulate)
  - [ ] `BP-06` profile strategy (`dev`, `demo`, `prod-safe`) + integrity checks
  - [ ] `BP-07` validation runner + rollout runbook + closure artifact log

## Intake rule

Add new items here when:
- work is agreed, and
- it is not committed in the same change set.

Each item should include:
- `Status` (`planned`, `in_progress`, `blocked`, `done`)
- `Priority` (`P0`, `P1`, `P2`)
- `Owner` (`Backend`, `Frontend`, `Platform/Ops`, or combo)
- concrete next step
- link to design/implementation plan doc.

Epic planning rule:
- for every new epic, create the design/implementation plan doc first, then execute work against that plan.

Prototype migration guardrail:
- avoid patch/alter migrations unless introducing a new table is required.

Fail-fast runtime guardrail:
- avoid silent fallback assumptions when explicit configuration exists but is invalid.
- if config files are present and parse/validation/policy checks fail, surface explicit errors and stop execution rather than degrading to implicit defaults.
- `In progress`: detector rule learning loop
  - store proposed detector rules from observed incidents
  - review/accept/reject through admin APIs
  - auto-activate into `robot_sre_policy.detector_rules` only on acceptance
  - optional `training_auto_create` mode for automatic learned-rule activation
