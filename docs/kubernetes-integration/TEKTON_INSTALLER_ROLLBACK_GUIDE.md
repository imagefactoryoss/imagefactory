# Tekton Installer Rollback Guide

## Purpose
This guide defines the manual rollback path for failed Tekton `upgrade` operations triggered from Image Factory.

## Scope
- Provider-scoped rollback (one infrastructure provider at a time).
- Covers both install modes:
  - `gitops`
  - `image_factory_installer`

## Preconditions
- Identify provider ID and intended rollback version.
- Confirm cluster access for the provider.
- Confirm no installer job is currently `pending` or `running` for that provider.

## 1) Assess Failure
1. Fetch installer status from Image Factory:
   - `GET /api/v1/admin/infrastructure/providers/{id}/tekton/status`
2. Record:
   - Failed job ID
   - Error message
   - Last successful asset version (from prior successful job, Git history, or release notes)
3. Validate cluster reachability and readiness:
   - `GET /api/v1/admin/infrastructure/providers/{id}/readiness`

## 2) Rollback for `gitops` Mode
1. Revert the GitOps source to the last known-good Tekton asset version.
2. Trigger your GitOps sync (ArgoCD/Flux) and wait for reconciliation.
3. Verify resources in target namespace:
   - Pipelines: `image-factory-build-v1-*`
   - Required tasks: `git-clone`, `docker-build`, `buildx`, `kaniko`, `packer`
4. Re-run readiness check endpoint.
5. Run a small validation build in Image Factory.

## 3) Rollback for `image_factory_installer` Mode
1. Checkout or prepare the previous known-good `backend/tekton` assets.
2. Apply the known-good manifests to the provider cluster namespace:
   - Use the same namespace configured by `tekton_target_namespace` (or tenant namespace fallback).
3. Re-run installer validation:
   - `POST /api/v1/admin/infrastructure/providers/{id}/tekton/validate`
4. Re-run readiness check endpoint.
5. Execute a smoke build to confirm successful scheduling and execution.

## 4) Post-Rollback Validation
1. Confirm no active installer job remains.
2. Confirm readiness reports no missing Tekton pipelines/tasks for the expected profile.
3. Confirm at least one build completes successfully.
4. Capture incident notes:
   - Failed version
   - Restored version
   - Root cause and follow-up action

## 5) Operational Notes
- Keep provider upgrades serialized; avoid concurrent install/upgrade operations.
- Prefer rollback to last known-good version before attempting forward-fix in production.
- If rollback fails repeatedly, disable provider selection for affected tenants until resolved.
