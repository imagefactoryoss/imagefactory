# Packer Multi-Cloud Build Handover

Last updated: 2026-03-27  
Branch: `feature/packer-builds`

## Delivered Through PR6

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

## Validation Run Notes

Backend:
- `go test ./internal/adapters/primary/rest -count=1`
- `go test ./internal/domain/build -count=1`

Frontend:
- `npm run build`

## Known Gap For Next PR (PR7)

- scheduled triggers for packer builds (schedule config + scheduler dispatch metadata).
- expand VM catalog semantics for lifecycle actions (promote/deprecate/delete) after schedule support lands.
- continue hardening provider-specific artifact extraction heuristics for edge-case parser coverage.
