# Packer Multi-Cloud Build Handover

Last updated: 2026-03-27  
Branch: `feature/packer-builds`

## Delivered Through PR7 + scheduled notification hooks

- PR1 completed:
  - Packer config contract hardening (`variables`, `build_vars`, `on_error`, `parallel`) across backend/frontend/runtime.
- PR2 completed:
  - Tekton parity for Packer invocation semantics, including deterministic repeated `-var` handling.
- PR3 completed:
  - admin-managed `packer_target_profiles` model, APIs, persistence, and deterministic validation.
  - admin frontend page for profile CRUD + validate.
- PR4 completed:
  - tenant Packer build config now requires `packer_target_profile_id`.
  - create/start/retry preflight enforces profile entitlement and `validation_status=valid`.
  - execution metadata persists selected profile/provider context and derived provider artifact identifiers.
- PR5 completed:
  - tenant Packer wizard now uses entitled target profile selector (no free-form UUID entry).
- PR6 completed:
  - tenant VM image catalog APIs and UI shipped (`/images/vm` list + details drawer).
  - catalog exposes provider artifact identifiers, source traceability, and lifecycle state.
- PR7 backend scheduler slice completed:
  - dispatcher now processes due schedule triggers and queues scheduled-origin packer builds.
  - cron-based `next_trigger_at` calculation added for schedule create + post-fire advancement.
  - scheduled-origin metadata + default `forbid` concurrency behavior added.
- PR7 follow-up notification hooks completed:
  - schedule runner now publishes scheduled outcome statuses (`scheduled_queued`, `scheduled_failed`, `scheduled_noop`).
  - build notification mapping includes trigger IDs `BN-011`, `BN-012`, `BN-013`.
  - scheduled failure trigger defaults include email channel alongside in-app.
- PR7 frontend notification catalog parity completed:
  - admin tenant defaults panel includes `BN-011`/`BN-012`/`BN-013`.
  - tenant details notification drawer includes `BN-011`/`BN-012`/`BN-013`.
  - project notification trigger matrix includes scheduled trigger rows and descriptions.
- Provider artifact extraction hardening completed:
  - extraction now scans nested/object artifact payloads in addition to array payloads.
  - GCP artifact capture supports both `/projects/...` and `projects/...` identifier forms.
  - VMware capture now prefers identifier-like values and avoids bare label false positives.
- PR8 backend lifecycle action foundation completed:
  - added lifecycle action routes for VM catalog executions:
    - `POST /api/v1/images/vm/{executionId}/promote`
    - `POST /api/v1/images/vm/{executionId}/deprecate`
    - `DELETE /api/v1/images/vm/{executionId}`
  - lifecycle transitions now persist `metadata.packer.lifecycle_state` (+ action metadata fields).
  - VM catalog list/detail lifecycle responses now honor metadata overrides (`released`, `deprecated`, `deleted`).
  - guardrails block lifecycle transitions for active/failed/cancelled executions.
- PR8 frontend lifecycle action controls completed:
  - VM catalog row actions now include `Promote`, `Deprecate`, and `Delete`.
  - actions use confirmation-dialog flow with success/error toast feedback.
  - lifecycle-aware action disabling prevents invalid UI transitions.
  - list and detail drawer are refreshed after action completion.
- PR8 lifecycle policy + audit hardening completed:
  - `deprecate` and `delete` require explicit reason payloads.
  - `delete` transitions are accepted only from `deprecated` lifecycle state.
  - metadata now persists bounded `lifecycle_history` entries (`state`, `reason`, `actor_id`, `at`).
  - VM catalog responses include lifecycle last-action fields + lifecycle history for traceability.
- PR8 lifecycle action contract normalization completed:
  - VM catalog list/detail API responses now include `action_permissions` booleans (`can_promote`, `can_deprecate`, `can_delete`).
  - frontend VM lifecycle action buttons now use backend-provided permissions instead of local transition inference.
- PR8 frontend lifecycle reason-entry UX completed:
  - VM catalog now uses an in-app reason modal for `deprecate` and `delete` before transition confirmation.
  - UI now forwards typed reasons to lifecycle APIs rather than using generated default reason strings.
- PR8 lifecycle reason payload hardening completed:
  - backend now enforces max lifecycle reason length at 500 characters.
  - VM catalog reason modal now applies the same 500-character limit with inline count feedback.
- PR8 lifecycle transition-mode contract clarity completed:
  - VM catalog responses now include `lifecycle_transition_mode`.
  - lifecycle transitions now report runtime mode depth (`metadata_only` / `provider_native` / `hybrid`) based on execution behavior.
- PR8 lifecycle audit mode-depth completed:
  - lifecycle history metadata now includes per-transition `transition_mode`.
  - VM catalog lifecycle history UI now renders each transition's mode for clearer audit context.
- PR8 lifecycle provider audit-depth expansion completed:
  - lifecycle metadata/history now records provider-native execution details (`provider_action`, `provider_identifier`, `provider_outcome`).
  - VM catalog payloads now include latest provider execution audit fields to reduce provider-side debugging latency.
- PR8 lifecycle action response contract hardening completed:
  - idempotent lifecycle transitions now return VM catalog `data` payload plus message (not message-only).
  - backend now uses a shared VM catalog item builder across list/detail/action responses for payload consistency.
- PR8 lifecycle action UX parity follow-up completed:
  - VM catalog action toasts now prioritize backend action messages.
  - idempotent lifecycle outcomes are now presented with backend-provided messaging in UI.
- PR8 lifecycle transition-mode normalization hardening completed:
  - backend now enforces allowlisted transition modes (`metadata_only`, `provider_native`, `hybrid`) when parsing metadata.
  - unknown transition-mode values now safely normalize to `metadata_only`.
- PR8 lifecycle transition-mode default contract hardening completed:
  - VM catalog response payloads now always emit `lifecycle_transition_mode`.
  - empty/malformed execution metadata now defaults transition mode to `metadata_only`.
- PR8 lifecycle transition-mode visibility follow-up completed:
  - VM catalog table lifecycle column now shows transition mode inline per artifact row.
- PR8 lifecycle transition-mode filterability follow-up completed:
  - VM catalog list API now supports `transition_mode` filtering.
  - VM catalog UI now includes transition-mode filter dropdown for list narrowing.
- PR8 provider-lifecycle execution seam foundation completed:
  - lifecycle transitions now pass through a backend lifecycle executor interface before metadata update.
  - executor now supports provider-native transitions across AWS, VMware, Azure, and GCP when execution mode allows.
- PR8 provider-lifecycle execution-mode policy gate completed:
  - added `IF_VM_LIFECYCLE_EXECUTION_MODE` with `metadata_only` (default), `prefer_provider_native`, and `require_provider_native`.
  - `require_provider_native` now returns `501` for unsupported provider/state paths and missing provider execution metadata.
  - added per-provider provider-native rollout toggles (default enabled): `IF_VM_LIFECYCLE_PROVIDER_AWS_ENABLED`, `IF_VM_LIFECYCLE_PROVIDER_VMWARE_ENABLED`, `IF_VM_LIFECYCLE_PROVIDER_AZURE_ENABLED`, `IF_VM_LIFECYCLE_PROVIDER_GCP_ENABLED`.
- PR8 provider-native lifecycle smoke tooling completed:
  - executable smoke runner: `scripts/packer-lifecycle-provider-native-smoke.sh`.
  - operational runbook: `docs/implementation/PACKER_VM_LIFECYCLE_PROVIDER_NATIVE_SMOKE_RUNBOOK.md`.
- PR8 provider-native lifecycle matrix validation tooling completed:
  - executable matrix runner: `scripts/packer-lifecycle-provider-native-matrix.sh`.
  - matrix validation runbook: `docs/implementation/PACKER_VM_LIFECYCLE_PROVIDER_NATIVE_MATRIX_RUNBOOK.md`.
  - smoke runbook now links single-provider and matrix workflows.
- PR8 provider-native lifecycle no-cloud mock validation mode completed:
  - smoke and matrix runners now support `SMOKE_MODE=mock_success` for deterministic validation without cloud/API credentials.
  - mock mode validates payload shape, lifecycle transition assertions, and report generation paths while skipping provider API calls.
- PR8 provider-native lifecycle initial execution completed:
  - AWS `delete` lifecycle action now supports provider-native execution via EC2 `DeregisterImage` when execution mode allows.
  - successful AWS native delete transitions now persist `provider_native`; invalid/missing AWS image metadata now returns `400`.
- PR8 provider-native lifecycle expansion completed:
  - AWS `deprecate` lifecycle action now supports provider-native EC2 image deprecation when execution mode allows.
- PR8 provider-native lifecycle release expansion completed:
  - AWS `released` lifecycle action now supports provider-native EC2 image undeprecation when execution mode allows.
- PR8 provider-native lifecycle metadata compatibility expansion completed:
  - AWS native lifecycle actions now use execution artifact values as fallback image lookup when provider identifiers are not present in execution metadata.
- PR8 provider-native lifecycle artifact-shape compatibility expansion completed:
  - execution artifact parsing now supports nested payload shapes for identifier extraction fallback, improving native lifecycle compatibility with older/non-standard artifact payloads.
- PR8 VMware provider-native lifecycle execution completed:
  - VMware lifecycle actions now support provider-native execution for `released` / `deprecated` / `deleted` transitions via vCenter-backed executor path.
  - VMware image/template identifiers are resolved from provider metadata and execution artifacts with strict validation (`400` on missing/invalid identifier input).
  - vCenter executor configuration uses:
    - `IF_VM_LIFECYCLE_VMWARE_VCENTER_URL`
    - `IF_VM_LIFECYCLE_VMWARE_USERNAME`
    - `IF_VM_LIFECYCLE_VMWARE_PASSWORD`
    - optional `IF_VM_LIFECYCLE_VMWARE_DATACENTER`
    - optional `IF_VM_LIFECYCLE_VMWARE_INSECURE`
- PR8 Azure provider-native lifecycle execution completed:
  - Azure lifecycle actions now support provider-native execution for `released` / `deprecated` / `deleted` transitions via ARM API-backed executor path.
  - Azure image identifiers are resolved from provider metadata and execution artifacts with strict validation (`400` on missing/invalid identifier input).
  - Azure executor configuration uses:
    - `IF_VM_LIFECYCLE_AZURE_BEARER_TOKEN`
    - optional `IF_VM_LIFECYCLE_AZURE_API_VERSION`
    - optional `IF_VM_LIFECYCLE_AZURE_DEPRECATION_HOURS`
- PR8 GCP provider-native lifecycle execution completed:
  - GCP lifecycle actions now support provider-native execution for `released` / `deprecated` / `deleted` transitions via Compute API-backed executor path.
  - GCP image identifiers are resolved from provider metadata and execution artifacts with strict validation (`400` on missing/invalid identifier input).
  - GCP executor configuration uses:
    - `IF_VM_LIFECYCLE_GCP_BEARER_TOKEN`
    - optional `IF_VM_LIFECYCLE_GCP_BASE_URL`

## PR3 Backend Summary

Migration:
- `backend/migrations/082_packer_target_profiles.up.sql`
- `backend/migrations/082_packer_target_profiles.down.sql`

Core backend modules:
- `backend/internal/domain/packertarget/*`
- `backend/internal/adapters/secondary/postgres/packer_target_profile_repository.go`
- `backend/internal/adapters/primary/rest/packer_target_profile_handler.go`

Admin API contract:
- `GET /api/v1/admin/packer-target-profiles`
- `POST /api/v1/admin/packer-target-profiles`
- `GET /api/v1/admin/packer-target-profiles/{id}`
- `PUT /api/v1/admin/packer-target-profiles/{id}`
- `DELETE /api/v1/admin/packer-target-profiles/{id}`
- `POST /api/v1/admin/packer-target-profiles/{id}/validate`

Validation behavior:
- deterministic/config-based checks in PR3 (no external connectivity probing yet).
- persisted fields:
  - `validation_status` (`untested|valid|invalid`)
  - `last_validated_at`
  - `last_validation_message`
  - `last_remediation_hints`

## PR3 Frontend Summary

New page:
- `frontend/src/pages/admin/AdminPackerTargetProfilesPage.tsx`

Routing/navigation:
- route: `/admin/infrastructure/packer-target-profiles`
- App routing wired in `frontend/src/App.tsx`
- Admin nav entry wired in `frontend/src/components/layout/AdminLayout.tsx`

Service/types:
- `frontend/src/services/packerTargetProfileService.ts`
- `frontend/src/types/index.ts` (Packer target profile + validation types)

## PR4 Backend Summary

Core backend modules touched:
- `backend/internal/domain/build/build_config_models.go`
- `backend/internal/domain/build/build_validation.go`
- `backend/internal/domain/build/service_create_preflight.go`
- `backend/internal/domain/build/service_commands.go`
- `backend/internal/domain/build/service_packer_target_profile_validation.go`
- `backend/internal/domain/build/service_execution_metadata.go`
- `backend/internal/domain/build/packer_execution_metadata.go`
- `backend/internal/domain/build/method_tekton_executor.go`
- `backend/internal/domain/build/method_tekton_executor_monitoring.go`
- `backend/internal/adapters/primary/rest/config_handler.go`
- `backend/internal/adapters/primary/rest/router.go`

PR4 behavior:
- `packer_target_profile_id` is validated as required UUID for `BuildTypePacker`.
- Build preflight fails fast when:
  - profile is not tenant-entitled.
  - profile validation status is not `valid`.
- Packer execution metadata now stores:
  - `packer.target_profile_id`
  - `packer.target_provider` (when available)
  - `packer.provider_artifact_identifiers` (derived from artifacts/logged results).

## PR4 Frontend Summary

Files touched:
- `frontend/src/services/buildService.ts`
- `frontend/src/components/builds/steps/PackerConfigForm.tsx`
- `frontend/src/components/builds/steps/ConfigurationStep.tsx`
- `frontend/src/types/buildConfig.ts`
- `frontend/src/types/index.ts`

PR4 behavior:
- build payload mapping includes `build_config.packer_target_profile_id`.
- wizard Packer form exposes required target profile ID input.

## PR6 Backend Summary

Core backend modules touched:
- `backend/internal/adapters/primary/rest/vm_image_handler.go`
- `backend/internal/adapters/primary/rest/router_route_registration.go`
- `backend/internal/adapters/primary/rest/router.go`
- `backend/internal/adapters/primary/rest/vm_image_handler_test.go`

PR6 API contract:
- `GET /api/v1/images/vm`
- `GET /api/v1/images/vm/{executionId}`

PR6 behavior:
- tenant reads packer execution-derived VM artifacts from `build_executions` + `builds` + `projects` + `build_configs`.
- supports filters: `provider`, `status`, `search`, plus pagination (`limit`, `offset`).
- response includes:
  - source traceability (`project_id`, `build_id`, `execution_id`, build number).
  - lifecycle fields (`build_status`, `execution_status`, derived `lifecycle_state`).
  - packer metadata (`target_provider`, `target_profile_id`, `provider_artifact_identifiers`).
  - execution artifact values for operator inspection.

## PR6 Frontend Summary

Files touched:
- `frontend/src/services/vmImageService.ts`
- `frontend/src/pages/images/VMImagesPage.tsx`
- `frontend/src/App.tsx`
- `frontend/src/components/layout/Layout.tsx`

PR6 behavior:
- new tenant route: `/images/vm`.
- filterable VM image catalog table.
- detail drawer showing provider artifact identifiers, source build link, and lifecycle state.
- dark-mode-ready styling for all new controls/tables/drawer sections.

## PR7 Backend Summary

Core backend modules touched:
- `backend/internal/domain/build/service_scheduled_triggers.go`
- `backend/internal/domain/build/cron_schedule.go`
- `backend/internal/domain/build/service_triggers.go`
- `backend/internal/adapters/secondary/postgres/trigger_repository.go`
- `backend/internal/application/dispatcher/queue_dispatcher.go`

Tests added/updated:
- `backend/internal/domain/build/cron_schedule_test.go`
- `backend/internal/application/dispatcher/queue_dispatcher_test.go`

PR7 backend behavior:
- dispatcher executes `ProcessScheduledTriggers(...)` each cycle before claiming queued builds.
- due active schedule triggers are loaded from trigger repository and processed in bounded batches.
- only packer source builds are auto-fired by schedule runner.
- fired builds are created as queued builds with schedule metadata:
  - `trigger_type=schedule`
  - `trigger_mode=scheduled`
  - `schedule_trigger_id`
  - `schedule_fire_timestamp`
  - `schedule_concurrency_policy=forbid`
- `forbid` policy: skip schedule fire when another queued/running packer build exists in the same project.
- schedule trigger update endpoint now supports schedule fields and pause/resume:
  - `PATCH /api/v1/projects/{projectID}/triggers/{triggerID}`
  - accepted fields for schedule triggers: `name`, `description`, `cron_expression`, `timezone`, `is_active`.
- trigger update and schedule pause/resume now emit build status audit events:
  - `trigger.updated`
  - `trigger.schedule.paused`
  - `trigger.schedule.resumed`
- scheduled trigger execution now emits outcome status updates for notification routing:
  - `scheduled_queued`
  - `scheduled_failed`
  - `scheduled_noop`

## Validation Run Notes

Backend:
- `go test ./internal/adapters/primary/rest -count=1`
- `go test ./internal/domain/build -count=1`
- `go test ./internal/application/dispatcher -count=1`
- `go test ./internal/adapters/secondary/postgres -run TriggerRepository -count=1`
- `go test ./internal/application/buildnotifications -count=1`

Frontend:
- `npm run build`
- `npm test -- --run src/components/projects/__tests__/ProjectNotificationTriggerMatrix.test.tsx`

## Known Gap For Next PR

- provider-native lifecycle actions are implemented across AWS, VMware, Azure, and GCP; next gap is executing the matrix runbook against staging/prod-like environments and attaching evidence artifacts for each provider.
