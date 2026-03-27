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
  - lifecycle transitions currently report `metadata_only` to explicitly signal metadata-state transitions (provider-native lifecycle actions pending).
- PR8 lifecycle audit mode-depth completed:
  - lifecycle history metadata now includes per-transition `transition_mode`.
  - VM catalog lifecycle history UI now renders each transition's mode for clearer audit context.
- PR8 lifecycle action response contract hardening completed:
  - idempotent lifecycle transitions now return VM catalog `data` payload plus message (not message-only).
  - backend now uses a shared VM catalog item builder across list/detail/action responses for payload consistency.

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

- implement provider-side lifecycle actions (actual provider deprecate/delete integrations) behind policy gates; current lifecycle transitions remain metadata-only state management.
