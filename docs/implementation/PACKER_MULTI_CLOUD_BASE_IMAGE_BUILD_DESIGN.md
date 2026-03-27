# Packer Multi-Cloud Base Image Build Design

Status: Draft for refinement  
Last updated: 2026-03-27  
Owners: Backend + Platform/Ops + Frontend

## Implementation status (2026-03-27)

- PR1 (contract hardening + canonical path guardrails): completed on `feature/packer-builds`.
- PR2 (Tekton parity for Packer execution): completed on `feature/packer-builds`.
- PR3 (admin target profiles read/write/validate): completed on `feature/packer-builds`.
- PR4 (tenant profile binding + preflight + execution metadata): completed on `feature/packer-builds`.
- PR5 (tenant profile selector UX): completed on `feature/packer-builds`.
- PR6 (tenant VM image catalog read path): completed on `feature/packer-builds`.
- PR7 backend scheduler slice (scheduled trigger execution + metadata): completed on `feature/packer-builds`.
- PR7 follow-up (scheduled outcome notification hooks): completed on `feature/packer-builds`.
- PR7 UX/management follow-up + PR8 remain pending.

## 1) Objective

Define and implement a single, deterministic Packer build capability for base OS images targeting:
- VMware
- Azure
- AWS
- GCP

This document is for requirement refinement and design alignment before coding.

## 2) Scope

In scope:
- BuildType `packer` execution path (local + Tekton).
- Provider-specific template support and validation.
- Packer variable and option contract across UI, backend persistence, and runtime.
- Provider preflight checks for credentials and environment prerequisites.
- Reference build templates and smoke validation approach.

Out of scope for initial phase:
- Full golden-image lifecycle management (promotion, deprecation, regional replication policy).
- Marketplace publication automation beyond image artifact creation.
- Cost optimization and spot/preemptible tuning.

## 3) Current State Summary

### 3.1 Execution paths

- Modern path: `BuildTypePacker` runs through method executor and invokes `packer build` with template + variables.
- Legacy path: VM-oriented executor maps provider to builder type (`amazon-ebs`, `azure-arm`, `vmware-iso`), with no explicit GCP case currently present.
- Tekton path exists for Packer via dedicated task/pipeline assets.

### 3.2 Observed gaps

- Tekton `vars` handling requires explicit `-var key=value` normalization to match local behavior.
- Advanced Packer config fields exposed in UI are not fully persisted/executed end-to-end.
- Provider-specific preflight checks are not unified for all targets.
- Legacy VM mapper parity is incomplete for GCP and may diverge from modern path.

## 4) Target Architecture

## 4.1 Single source of truth

Use `BuildTypePacker` (method-based executor) as the canonical execution path for all four cloud/virtualization targets.

Decision:
- Keep legacy VM path only for backward compatibility in short term.
- Add explicit deprecation path after parity is validated.

## 4.2 Provider model

Provider selection for Packer builds is driven by template builder type plus provider credentials/readiness:
- VMware -> `vmware-iso` (or `vsphere-iso` as follow-up if required)
- Azure -> `azure-arm`
- AWS -> `amazon-ebs`
- GCP -> `googlecompute`

We should support both:
- direct template submission by advanced users
- curated provider-specific reference templates for guided onboarding

## 4.3 Contract alignment

Canonical Packer config contract:
- `template` (required)
- `packer_target_profile_id` (required, UUID)
- `variables` (map string->string, rendered as repeated `-var key=value`)
- `build_vars` (optional, additional runtime vars)
- `on_error` (optional: `cleanup`/`abort`/etc.)
- `parallel` (optional integer)

All accepted fields must be:
1. accepted in API
2. persisted in DB
3. rehydrated by repository
4. applied by local executor and Tekton executor

No silent field drops.

## 4.4 Tekton execution contract

Tekton Packer task should invoke:
- `packer init` (if required by template plugin blocks)
- `packer build` with normalized args:
  - template path
  - repeated `-var key=value`
  - optional `-parallel-builds`
  - optional `-on-error`

All arguments should be logged in sanitized form (never secrets).

## 5) Provider-specific readiness and credentials

Minimum preflight checks before scheduling:

1. VMware
- required credentials present (vCenter/ESXi auth)
- reachable endpoint
- datastore/network references resolvable (where applicable)

2. Azure
- service principal/workload identity fields present
- subscription/resource-group context present
- permissions validated for image create/update path

3. AWS
- credentials/role present
- region and VPC/subnet context valid
- AMI create/register permissions validated

4. GCP
- service account/project present
- required APIs enabled (or surfaced as actionable failure)
- image create permissions validated

Preflight must fail fast with provider-specific remediation.

## 6) Reference template strategy

Ship one baseline template per target under versioned docs/examples:
- `aws-amazon-ebs.pkr.hcl`
- `azure-arm.pkr.hcl`
- `gcp-googlecompute.pkr.hcl`
- `vmware-iso.pkr.hcl`

Each template includes:
- minimal base OS build
- required variable list
- example variable values (non-secret)
- expected output artifact identifiers

## 6.1 Tenant repo contract (when build method is Packer)

When a tenant selects build method `packer`, the project repository should act as the source of truth for build intent.

Minimum required in repo:
- one Packer template file committed to source control (for example `packer/base-os.pkr.hcl`).

Recommended repo layout:

```text
packer/
  templates/
    aws-base.pkr.hcl
    azure-base.pkr.hcl
    gcp-base.pkr.hcl
    vmware-base.pkr.hcl
  vars/
    common.pkrvars.hcl
    aws.example.pkrvars.hcl
    azure.example.pkrvars.hcl
    gcp.example.pkrvars.hcl
    vmware.example.pkrvars.hcl
  scripts/
    bootstrap.sh
  cloud-init/
    base-user-data.yaml
  README.md
```

Build configuration expectations:
- Build stores a `template path` (preferred) or inline template payload (allowed for quick tests only).
- Build stores runtime variables separately from repo for environment-specific values.
- Secrets are injected at runtime from managed secret sources and must not be committed to repository.

Tenant-facing validation rules:
- template file exists at declared path in selected revision.
- required variables declared by template are provided (or have defaults).
- template syntax is valid before scheduling (`packer validate` preflight).
- provider/builder compatibility is valid for target platform (`amazon-ebs`, `azure-arm`, `googlecompute`, `vmware-iso`).
- repository must not contain plaintext secrets in committed var files used by production builds.

Artifact expectations per run:
- execution captures provider-specific artifact identifiers and records them in build metadata:
  - AWS: AMI ID
  - Azure: image resource ID
  - GCP: image name/self link
  - VMware: template/image artifact identifier/path

## 7) Phased delivery plan

Phase 0: Contract hardening
- finalize Packer config schema and persistence mapping.
- enforce no-silent-drop validation behavior.

Phase 1: Execution parity
- align local and Tekton variable/option rendering.
- add execution tests for key arg assembly.

Phase 2: Provider readiness
- implement/extend provider-specific preflight checks.
- add actionable error catalog for each provider.

Phase 3: Template and UX
- add reference templates and docs.
- align frontend fields with actual backend support and validation messages.

Phase 4: Legacy path cleanup
- either add temporary GCP parity to legacy VM mapper or route all VM use cases to canonical packer path.
- deprecate legacy mapper once no longer needed.

## 8) Test and validation plan

Automated:
- unit tests for config mapping and executor arg rendering.
- integration tests for local/tekton packer scheduling contract.
- provider preflight tests for expected pass/fail scenarios.

Manual smoke matrix:
- one successful baseline image build per provider (VMware/Azure/AWS/GCP).
- one intentional misconfiguration per provider to verify actionable failure.

Definition of done:
- all four providers build base OS image via canonical path.
- logs and API responses are deterministic and diagnosable.
- docs include quick-start templates and troubleshooting.

## 9) Open questions for refinement

1. Do we need `vsphere-iso` in initial scope or can we start with `vmware-iso` only?
2. Should GCP support be added to legacy VM mapper immediately, or should we gate legacy mapper and route to canonical Packer path?
3. Do we require `packer init` on every run, or only when plugin blocks are detected?
4. Which credentials source is mandatory per provider (static secrets vs workload identity)?
5. What is the minimum artifact metadata we must persist per provider (AMI ID, Azure image resource ID, GCP image name, VMware artifact path)?

## 10) Initial implementation candidates

Likely touchpoints:
- `backend/internal/domain/build/method_packer_executor.go`
- `backend/internal/domain/build/service_build_config_persistence.go`
- `backend/internal/adapters/secondary/postgres/build_method_config_repository.go`
- `backend/tekton/tasks/v1/packer-task.yaml`
- `backend/internal/domain/build/method_tekton_executor_templates.go`
- `frontend/src/components/builds/steps/PackerConfigForm.tsx`
- `frontend/src/components/builds/steps/ConfigurationStep.tsx`

## 11) Artifact storage and lifecycle

Packer-produced base OS images are provider-native artifacts, not container images.  
For the initial design, we should treat provider image systems as the system of artifact storage and use Image Factory as the system of record for metadata and lifecycle state.

### 11.1 Where artifacts are stored

- AWS: AMI in target account/region.
- Azure: Managed Image or Shared Image Gallery (SIG) image version in target subscription/region.
- GCP: Compute Image in target project.
- VMware: template/OVA artifact in vSphere datastore or content library.

Non-goal for this phase:
- pushing Packer outputs to OCI container registry as the primary artifact store.

### 11.2 What we persist in Image Factory

On successful build completion, persist normalized artifact metadata:
- provider type (`aws`, `azure`, `gcp`, `vmware`)
- provider/account/project/subscription identifiers
- location (`region`, `zone`, `datacenter`, or equivalent)
- artifact identifier (AMI ID, Azure resource ID, GCP image self link/name, VMware template/path)
- artifact name/version
- source traceability (`git revision`, template path/hash, build execution id)
- status (`created`, `validated`, `released`, `deprecated`, `deleted`, `failed`)
- timestamps and actor context

Current implementation note (PR4):
- execution metadata now stores `packer.target_profile_id`, optional `packer.target_provider`, and derived `packer.provider_artifact_identifiers`.

### 11.3 When metadata is persisted

1. Build start:
- create provisional artifact record linked to execution with status `created_pending`.

2. Build success:
- parse provider output and update artifact record to `created` with final identifiers.

3. Post-build validation (optional but recommended):
- run provider-specific smoke checks and transition to `validated`.

4. Release/promotion:
- promote immutable artifact reference to environment/channel and transition to `released`.

### 11.4 Lifecycle controls

Required controls:
- naming/version convention including app version + commit short SHA.
- retention policy (max historical versions, age-based cleanup, protected releases).
- deprecation semantics (can no longer be used for new deployments but retained for rollback).
- delete guardrails (approval or policy-gated for destructive operations).

### 11.5 Open decisions (must finalize before implementation)

1. Azure default target: Managed Image vs SIG image version.
2. VMware default target: vSphere template vs OVA artifact.
3. Minimum validation gate before `released` status.
4. Multi-region replication strategy per provider.
5. Role permissions for release/deprecate/delete artifact actions.

## 12) Tenant frontend UX and VM image catalog

This section defines the intended tenant-facing experience when build method is `packer`, and the VM image catalog model used to consume produced artifacts.

### 12.1 Tenant Packer build experience

Entry point:
- existing tenant flow remains `Project -> Build`.
- when method is `packer`, wizard switches to a VM-image-focused configuration path.

Proposed wizard steps:

1. Template source
- select `repository path` (recommended) or `inline template` (advanced/testing).
- required inputs:
  - repository revision/branch
  - template path (for repo mode)
- preflight:
  - template path exists in selected revision
  - template parses/validates (`packer validate`)

2. Target platform
- selectable platform cards:
  - VMware
  - Azure
  - AWS
  - GCP
- selected platform drives builder compatibility checks and provider-specific field hints.

3. Destination
- show destination options allowed for selected platform and tenant entitlements.
- VMware initial options:
  - vSphere template
  - OVA artifact
  - Artifactory destination profile (admin-managed)

4. Variables and secrets
- plain variables and secret references are configured separately.
- show sanitized preview of effective variables (no secret value disclosure).

5. Review and run
- show final naming/version convention (`<version>-<commit-short-sha>`).
- show expected artifact destination and metadata that will be recorded.
- explicit warnings for destructive flags (if any).

### 12.2 Tenant VM image catalog

Add tenant workspace page:
- `Images -> VM Images`

Catalog list columns:
- image name
- platform
- provider/account/project/subscription
- location (region/datacenter)
- version/tag
- lifecycle status (`created`, `validated`, `released`, `deprecated`, `deleted`, `failed`)
- source project/build/execution
- created/updated timestamps

Catalog filters:
- platform
- status
- project
- provider
- environment/channel
- date range

Catalog row details drawer:
- artifact identifiers/URIs (provider-native IDs and optional external destination URI)
- template path + source revision
- build execution/log links
- validation and promotion history
- policy/audit trail for state changes

### 12.3 VMware + Artifactory admin configuration

Add admin configuration area:
- `Admin -> Infrastructure -> Artifact Destinations`

Destination profile types:
- VMware vSphere destination
- Artifactory generic destination

Artifactory profile fields:
- profile name
- base URL
- repository key and optional path prefix
- authentication mode:
  - token secret reference
  - username/password secret reference
- optional naming template (artifact file/object naming)
- TLS settings:
  - verify TLS toggle
  - optional custom CA secret reference
- connectivity test action and last-validated status

Usage model:
- destination profiles are created by admin and scoped/entitled to tenants.
- tenant Packer builds select from allowed destination profiles only.
- build execution writes resulting external URI/path into artifact metadata for catalog visibility.

### 12.4 UX guardrails and failure behavior

- fail fast in wizard when required provider/destination config is missing.
- render provider-specific remediation hints inline (no generic opaque error text).
- prevent release-state transitions for artifacts missing required metadata.
- keep permission-sensitive actions (promote/deprecate/delete) gated by role and audit logged.

## 13) Build trigger model (on-demand and scheduled)

Packer builds should support two trigger modes:
- on-demand execution
- scheduled execution

### 13.1 On-demand

Use case:
- tenant operator wants immediate image creation after template or variable changes.

UX:
- `Run now` action from build details and from build wizard review step.
- optional override inputs at run time (non-persisted one-off variables, if policy allows).

Behavior:
- create execution immediately and run standard preflight + validation pipeline.
- record trigger metadata:
  - `trigger_mode=on_demand`
  - `triggered_by` actor
  - request timestamp

### 13.2 Scheduled

Use case:
- routine base-image refresh (for example monthly patch baseline rebuilds).

UX:
- `Schedule` section on build configuration:
  - enabled/disabled toggle
  - cron expression
  - timezone
  - optional blackout window
  - optional max runtime/timeout override

Behavior:
- scheduler enqueues execution at next eligible window.
- record trigger metadata:
  - `trigger_mode=scheduled`
  - schedule id/version
  - schedule fire timestamp
- missed execution policy should be explicit:
  - `skip` missed windows by default
  - optional `catch_up_once` mode if enabled

### 13.3 Safety and operability for schedules

- concurrency policy required per build:
  - `forbid` (default)
  - `replace`
  - `allow`
- retries/backoff policy for transient provider failures.
- schedule-level notification routing for success/failure/no-op outcomes.
- guardrails to prevent duplicate overlapping runs during clock drift/restarts.
- explicit audit trail for schedule create/update/pause/resume/delete.

### 13.4 Suggested defaults

- default trigger mode: on-demand
- schedule disabled by default
- default concurrency policy: `forbid`
- default timezone: tenant-configured timezone (fallback UTC)
- default failure notifications: build owner + project maintainers

## 14) Future enhancements backlog (prioritized)

This section captures likely next features after core multi-cloud Packer delivery.

### P0 (high-value early follow-on)

1. Promotion pipeline support
- first-class `dev -> stage -> prod` promotion model using immutable provider artifact references.

2. Deprecation and EOL policy
- lifecycle controls for deprecate, grace period, and enforced retirement dates.

3. Quota and schedule guardrails
- tenant/project limits for concurrent Packer runs and schedule frequency.

4. CVE/SBOM linkage to VM artifacts
- attach scan/SBOM evidence to artifact records and expose in catalog details.

### P1 (operability and governance depth)

1. Drift detection
- detect when runtime/reference image state diverges from template, vars, or policy baseline.

2. Compliance profile packs
- selectable hardening/compliance profile gates (for example CIS/STIG/custom baseline checks).

3. Rollback recommendations
- suggest and enable revert to last-known-good release artifact by environment.

4. Secret rotation resilience
- non-disruptive credential rotation model for provider and destination profiles.

### P2 (advanced insight and optimization)

1. Cost estimation preflight
- estimate provider compute/storage/egress before execution.

2. Golden image lineage graph
- visualize artifact ancestry and release propagation path.

3. Intelligent schedule optimization
- optional recommendations for optimal rebuild cadence based on patch/advisory signals.

## 15) Execution topology and target-access profiles

Packer execution occurs on the Image Factory execution plane (local executor process or Tekton worker pod).  
Target platform APIs are called from that execution plane during build runtime.

### 15.1 Where builds run

- Local mode:
  - backend host/process executes `packer build`.
- Tekton mode:
  - Tekton task container executes `packer build` in cluster.

In both modes, the runtime needs outbound access to provider endpoints and required credentials.

### 15.2 Connectivity and credential requirements by provider

1. VMware
- network/API access from execution plane to vCenter or ESXi endpoint.
- credentials with required inventory/datastore/network/template permissions.
- optional content library or artifact destination connectivity when publishing outputs.

2. AWS
- outbound access to AWS APIs.
- IAM credentials/role with image create/register/read permissions.

3. Azure
- outbound access to Azure Resource Manager endpoints.
- service principal/workload identity permissions for image create/update/read.

4. GCP
- outbound access to GCP Compute APIs.
- service account/workload identity permissions for image create/read.

### 15.3 Admin-managed target-access profiles (modeled like infrastructure providers)

To mirror existing infrastructure-provider onboarding, introduce a first-class admin object for Packer targets.

Proposed object:
- `packer_target_profile`

Core fields:
- profile id/name
- provider type (`vmware`, `aws`, `azure`, `gcp`)
- endpoint/context fields per provider (for example vCenter URL, AWS account+region, Azure subscription, GCP project)
- credential reference (secret reference, not raw secret)
- network access hints (required egress endpoints, optional proxy config)
- validation status (`untested`, `valid`, `invalid`)
- last validation timestamp/message
- tenant visibility/entitlement scope
- policy flags (allowed builders, allowed destinations, schedule constraints)

### 15.4 Onboarding flow for target profiles

Admin flow (similar to infra provider flow):
1. create target profile
2. configure endpoint and credential references
3. run connectivity/permission validation checks
4. fix remediation items until status is `valid`
5. assign profile to allowed tenants/projects

Tenant behavior:
- tenant selects only entitled `packer_target_profile` entries in build wizard.
- tenant cannot override protected endpoint/credential fields.

### 15.5 Build-time enforcement

Before scheduling execution:
- resolve selected `packer_target_profile`
- enforce profile is `valid` and entitled to tenant
- verify required runtime capabilities (packer/plugin availability + network prerequisites)
- fail fast with targeted remediation if checks fail

### 15.6 Suggested implementation touchpoints

- backend domain: new `packer_target_profile` model + service/repository
- admin APIs/UI: create/edit/validate/list target profiles + entitlement controls
- build config: reference `packer_target_profile_id`
- preflight path: provider-specific target validation hooks

## 16) Ordered implementation rollout (PR slices)

This section converts the design into an implementation sequence with thin vertical slices.

### 16.1 Guiding constraints

- canonical execution path is `BuildTypePacker` method-based executor.
- avoid broad refactors in early slices; prioritize end-to-end deliverability.
- every slice must be releasable behind capability/feature flags where needed.

### 16.2 PR1 - Contract hardening and canonical path guardrails

Scope:
- ensure Packer config contract is explicit and no fields are silently dropped.
- enforce canonical path usage for Packer builds in orchestration and validation.
- add/extend tests for local executor argument rendering consistency.

Primary changes:
- backend build config validation + persistence alignment for:
  - template
  - variables
  - build vars
  - on_error
  - parallel
- add clear errors for unsupported/incomplete Packer config.

Acceptance criteria:
- build create/update round-trips all supported Packer fields.
- local executor emits deterministic `packer build` arg structure.
- existing non-Packer build methods remain unaffected.

### 16.3 PR2 - Tekton parity for Packer execution

Scope:
- normalize Tekton Packer task/pipeline parameter handling to match local executor behavior.
- pass variables as explicit repeated `-var key=value`.

Primary changes:
- `backend/tekton/tasks/v1/packer-task.yaml`
- Tekton render context/template wiring
- tests for rendered PipelineRun arguments

Acceptance criteria:
- same template + vars produce equivalent local and Tekton invocation semantics.
- failed Tekton preflight surfaces actionable missing-prereq messages.

### 16.4 PR3 - Admin Packer target profiles (read/write + validate)

Scope:
- introduce admin-managed `packer_target_profile` object.
- implement create/edit/list/detail/validate APIs.
- add admin UI for profile management and validation status.

Primary changes:
- backend domain/repository + DB migration(s)
- admin REST handlers
- frontend admin pages/components

Acceptance criteria:
- admin can create provider profiles (VMware/AWS/Azure/GCP) with secret references.
- validation endpoint returns deterministic pass/fail + remediation hints.
- profile status persisted (`untested|valid|invalid`) with timestamps.

Progress update (2026-03-27):
- backend schema added via `082_packer_target_profiles` migration.
- backend domain/service/repository added for `packer_target_profiles`.
- admin REST routes added:
  - `GET /api/v1/admin/packer-target-profiles`
  - `POST /api/v1/admin/packer-target-profiles`
  - `GET /api/v1/admin/packer-target-profiles/{id}`
  - `PUT /api/v1/admin/packer-target-profiles/{id}`
  - `DELETE /api/v1/admin/packer-target-profiles/{id}`
  - `POST /api/v1/admin/packer-target-profiles/{id}/validate`
- deterministic validation is currently configuration-driven (no live cloud/vCenter connectivity in PR3 backend slice).
- frontend admin page/forms added for target profile list/create/edit/delete/validate:
  - route: `/admin/infrastructure/packer-target-profiles`
  - navigation: `Admin > Build Management > Packer Target Profiles`

### 16.5 PR4 - Tenant build wiring to target profiles (on-demand only)

Scope:
- tenant Packer wizard consumes entitled target profiles.
- build config references `packer_target_profile_id`.
- enforce entitlement + valid-status preflight at schedule time.

Primary changes:
- tenant build API + domain enforcement
- frontend build wizard Packer steps
- audit metadata on trigger

Acceptance criteria:
- tenant can run on-demand Packer build using only entitled valid profiles.
- invalid/unentitled profiles fail fast with clear errors.
- successful run records target profile and provider artifact metadata.

### 16.6 PR5 - VMware destination options including Artifactory profile selection

Scope:
- add admin artifact destination profiles (starting with Artifactory).
- wire VMware Packer destination selection in tenant flow.
- persist destination URI/path in artifact metadata.

Primary changes:
- destination profile model + admin API/UI
- tenant destination selector for VMware builds
- execution metadata persistence updates

Acceptance criteria:
- admin can define Artifactory destination profile and test connectivity.
- tenant can select entitled VMware destination profile.
- successful run records resulting destination URI in artifact metadata.

### 16.7 PR6 - VM image catalog (tenant read path)

Scope:
- deliver tenant-facing `Images -> VM Images` list and detail drawer.
- expose artifact metadata + source traceability + status in API.

Primary changes:
- artifact read APIs
- frontend catalog table, filters, details drawer

Acceptance criteria:
- tenant can filter and inspect VM artifacts across supported providers.
- details include provider-native artifact IDs, source build links, and lifecycle state.

Progress update (2026-03-27):
- backend tenant VM image catalog APIs added:
  - `GET /api/v1/images/vm`
  - `GET /api/v1/images/vm/{executionId}`
- API exposes packer execution traceability and lifecycle context:
  - build/execution/project IDs, build number/status, execution status/lifecycle state.
  - `packer.target_provider`, `packer.target_profile_id`.
  - `packer.provider_artifact_identifiers`.
  - captured artifact values from execution payloads.
- frontend tenant page added at `/images/vm`:
  - filterable VM image catalog table (`provider`, `status`, `search`).
  - details drawer with source build link + provider identifier sections.

### 16.8 PR7 - Scheduled triggers for Packer builds

Scope:
- enable schedule config on Packer builds with concurrency policy.
- scheduler fires eligible executions with trigger metadata.

Primary changes:
- build schedule schema + API fields
- scheduler integration and idempotency safeguards
- notification hooks for scheduled outcomes

Acceptance criteria:
- schedule create/update/pause/resume works with audit trail.
- scheduled runs honor concurrency policy (`forbid` default).
- trigger metadata clearly identifies scheduled origin.

Progress update (2026-03-27):
- dispatcher now processes due active `schedule` triggers before queue dispatch.
- schedule triggers now compute and persist `next_trigger_at` from cron expression on create and after each fire.
- scheduled trigger firing currently scoped to `BuildTypePacker` (PR7 scope alignment).
- scheduled-origin metadata is now stamped into queued build manifests:
  - `trigger_type=schedule`
  - `trigger_mode=scheduled`
  - `schedule_trigger_id`
  - `schedule_fire_timestamp`
  - `schedule_concurrency_policy=forbid`
- default `forbid` policy is enforced by skipping schedule fire when another queued/running packer build exists in the project.
- schedule management parity now supports update/pause/resume via existing project trigger update endpoint:
  - `PATCH /api/v1/projects/{projectID}/triggers/{triggerID}`
  - schedule fields supported: `name`, `description`, `cron_expression`, `timezone`, `is_active`.
  - `is_active=false` pauses and `is_active=true` resumes schedule triggers.
- trigger update actions now publish build status audit events:
  - `trigger.schedule.paused`
  - `trigger.schedule.resumed`
  - `trigger.updated`
- scheduled fire outcomes now emit build status updates for notification routing:
  - `scheduled_queued` -> `BN-011` (`build_scheduled_queued`)
  - `scheduled_failed` -> `BN-012` (`build_scheduled_failed`)
  - `scheduled_noop` -> `BN-013` (`build_scheduled_noop`)
- notification defaults now include scheduled failure email delivery by default.
- frontend notification catalogs now include `BN-011`/`BN-012`/`BN-013` in:
  - admin tenant notification defaults panel
  - tenant details notification defaults drawer
  - project notification trigger matrix
- provider artifact extraction hardening now covers edge-case payload shapes:
  - nested/object artifact payload scanning (not only array payloads).
  - GCP identifiers in both `/projects/...` and `projects/...` formats.
  - VMware identifier filtering to reduce false positives from non-identifier labels.

### 16.9 PR8 - Lifecycle actions (promote/deprecate/delete) and policy guardrails

Scope:
- implement controlled artifact lifecycle transitions.
- enforce role/policy checks and audit logging for destructive transitions.

Primary changes:
- lifecycle state transition APIs
- policy/permission checks
- frontend actions in catalog/details

Acceptance criteria:
- authorized users can promote/deprecate with full audit trail.
- delete operations are policy-gated and blocked when protections apply.

Progress update (2026-03-27):
- backend lifecycle action foundation added for tenant VM catalog executions:
  - `POST /api/v1/images/vm/{executionId}/promote`
  - `POST /api/v1/images/vm/{executionId}/deprecate`
  - `DELETE /api/v1/images/vm/{executionId}`
- lifecycle state transitions persist under execution metadata:
  - `metadata.packer.lifecycle_state`
  - `metadata.packer.lifecycle_last_action_at`
  - `metadata.packer.lifecycle_last_action_by`
  - `metadata.packer.lifecycle_last_reason`
- catalog list/detail now prefer lifecycle override when present (`released`, `deprecated`, `deleted`).
- guardrails currently block transitions for active (`pending`/`running`) and non-releasable (`failed`/`cancelled`) executions.
- frontend VM catalog now includes lifecycle action controls:
  - `Promote`, `Deprecate`, `Delete` actions on each VM image row.
  - confirmation dialog + toast feedback + post-action list/detail refresh behavior.
  - lifecycle-aware button disabling to avoid invalid transitions from UI.
- lifecycle policy hardening now enforces:
  - explicit reason required for `deprecate` and `delete` transitions.
  - `delete` is allowed only from `deprecated` lifecycle state.
  - bounded lifecycle history persisted in execution metadata and surfaced in VM catalog details.
- VM catalog responses now include server-derived lifecycle action permissions:
  - `action_permissions.can_promote`
  - `action_permissions.can_deprecate`
  - `action_permissions.can_delete`
  - frontend action buttons consume these flags directly for policy-consistent UX.
- frontend lifecycle reason-entry UX now enforces operator-provided reasons:
  - `deprecate` and `delete` actions now use an in-app reason modal and require explicit reason text before confirmation.
  - typed reason values are sent to lifecycle transition APIs for audit-quality context.
- lifecycle reason payload hardening now enforces bounded reason size:
  - backend rejects lifecycle reason payloads longer than 500 characters.
  - frontend reason modal constrains input to 500 characters and shows inline character count.
- lifecycle transition mode is now explicit in VM catalog responses:
  - API exposes `lifecycle_transition_mode` for VM catalog list/detail/action responses.
  - current implementation emits `metadata_only`, clarifying lifecycle transitions are metadata state management until provider-native actions are integrated.
- lifecycle history audit depth now includes transition mode:
  - lifecycle history entries include `transition_mode` for each transition event.
  - VM catalog lifecycle history view surfaces each entry's transition mode for operator traceability.
- lifecycle action responses now use a consistent payload shape:
  - idempotent transitions (`already in target state`) return `data` + `message` instead of message-only payload.
  - list/detail/action responses share the same VM catalog item builder to reduce contract drift.
- VM catalog lifecycle UX now surfaces backend lifecycle action messages:
  - action success toasts prefer API-provided `message` values.
  - idempotent transition outcomes are shown with backend-authored text for operator clarity.
- lifecycle transition mode parsing now uses strict normalization:
  - accepted modes are `metadata_only`, `provider_native`, and `hybrid`.
  - unknown/invalid mode values are normalized to `metadata_only` for stable API/UI semantics.
- lifecycle transition mode now has a non-empty response contract:
  - VM catalog payloads default `lifecycle_transition_mode` to `metadata_only` when execution metadata is empty or malformed.
- VM catalog list UX now surfaces lifecycle transition mode inline:
  - lifecycle column in VM image table displays `lifecycle_transition_mode` per row for faster operator triage.
- VM catalog filtering now includes transition mode:
  - list endpoint accepts `transition_mode` filter with normalized mode matching.
  - frontend filter bar includes transition mode selector for rapid metadata-only/provider-native segmentation.
- lifecycle transition execution now has an extensible backend seam:
  - lifecycle handler invokes a lifecycle executor interface before persisting metadata transition fields.
  - default executor currently returns `metadata_only` (provider-native operations still pending), enabling incremental provider implementation without handler contract churn.
- lifecycle execution mode policy gate now exists for rollout safety:
  - `IF_VM_LIFECYCLE_EXECUTION_MODE` supports `metadata_only` (default), `prefer_provider_native`, and `require_provider_native`.
  - `require_provider_native` enforces fail-closed behavior (`501`) until provider-native transition executors are implemented.
- provider-native lifecycle execution now has initial AWS implementation:
  - AWS `delete` transitions invoke EC2 `DeregisterImage` when provider-native mode is enabled.
  - AWS `deprecate` transitions invoke EC2 `EnableImageDeprecation` with configurable deprecation window.
  - AWS `released` transitions invoke EC2 `DisableImageDeprecation` when provider-native mode is enabled.
  - AWS native transition image lookup now falls back to execution artifact payload values when provider identifier metadata is missing.
  - artifact extraction now scans nested artifact payload shapes (object/array string leaves), improving native fallback identifier discovery for legacy/non-standard payload formats.
  - successful native execution records `lifecycle_transition_mode=provider_native`; malformed AWS artifact metadata fails request validation.
- provider-native lifecycle execution now includes VMware implementation:
  - VMware `delete` transitions execute provider-native vCenter object destroy for the resolved VM/template identifier.
  - VMware `deprecate` / `released` transitions execute provider-native vCenter annotation updates to mark and clear deprecation state.
  - VMware identifier resolution supports provider metadata plus execution artifact fallback (`vsphere://`, inventory-path, and VM managed object reference forms).
  - vCenter connection is configured via `IF_VM_LIFECYCLE_VMWARE_VCENTER_URL`, `IF_VM_LIFECYCLE_VMWARE_USERNAME`, `IF_VM_LIFECYCLE_VMWARE_PASSWORD`, optional `IF_VM_LIFECYCLE_VMWARE_DATACENTER`, and `IF_VM_LIFECYCLE_VMWARE_INSECURE`.
- provider-native lifecycle execution now includes Azure implementation:
  - Azure `delete` transitions execute provider-native ARM `DELETE` for resolved image resource identifiers.
  - Azure `deprecate` / `released` transitions execute provider-native ARM `PATCH` updates for image lifecycle publication state.
  - Azure identifier resolution supports provider metadata plus execution artifact fallback (Azure resource ID forms).
  - Azure lifecycle execution is configured via `IF_VM_LIFECYCLE_AZURE_BEARER_TOKEN`, optional `IF_VM_LIFECYCLE_AZURE_API_VERSION`, and optional `IF_VM_LIFECYCLE_AZURE_DEPRECATION_HOURS`.

### 16.10 Cross-cutting quality gates for every PR

- unit + integration tests for changed modules.
- migration rollback safety (where schema changes apply).
- docs updated in same PR for user-facing behavior changes.
- no plaintext secrets in logs, API responses, or persisted config payloads.
- dark-mode-ready UI for all newly added frontend surfaces.
