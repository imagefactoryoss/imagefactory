# Packer Provider-Native Matrix Staging Runbook

## Purpose

Execute final staging/prod-like validation for provider-native VM lifecycle transitions across AWS, VMware, Azure, and GCP using one consistent matrix flow and attach evidence.

## Commands

Default no-cloud dry run (sanity check):

```bash
make qa-packer-provider-native-matrix
```

Staging/provider API run:

```bash
SMOKE_MODE=api \
BASE_URL='https://<staging-api-host>' \
AUTH_TOKEN='<staging-token>' \
TENANT_ID='<tenant-uuid>' \
AWS_EXECUTION_IDS='<aws-exec-id>' \
VMWARE_EXECUTION_IDS='<vmware-exec-id>' \
AZURE_EXECUTION_IDS='<azure-exec-id>' \
GCP_EXECUTION_IDS='<gcp-exec-id>' \
make qa-packer-provider-native-matrix
```

## Preconditions

- Disposable execution IDs exist for all target providers.
- Backend runtime is configured with:
  - `IF_VM_LIFECYCLE_EXECUTION_MODE=require_provider_native`
  - provider toggles enabled (`IF_VM_LIFECYCLE_PROVIDER_<PROVIDER>_ENABLED=true`)
- Provider credentials are configured in backend runtime for AWS/VMware/Azure/GCP.

## Evidence To Attach

- Runner log path from `docs/qa/artifacts/packer_provider_native_matrix_validation_<timestamp>.log`.
- Matrix report path printed by the runner (`/tmp/packer-provider-native-matrix-<timestamp>.log`).
- API response snippets per provider showing:
  - `lifecycle_transition_mode` (`provider_native` or `hybrid`)
  - `lifecycle_last_provider_action`
  - `lifecycle_last_provider_identifier`
  - `lifecycle_last_provider_outcome=success`
- Optional UI screenshots from VM catalog row/detail for each provider transition.

## Exit Criteria

- Matrix run completed in staging/prod-like mode (`SMOKE_MODE=api`) with:
  - `pass=4 fail=0` (or `pass=<targeted provider count> fail=0` if subset intentionally scoped)
  - no provider API failures in runner log.
- Validation template updated:
  - `docs/qa/PACKER_PROVIDER_NATIVE_MATRIX_VALIDATION_LOG.md`
- Packer backlog item can be marked `done` once evidence is attached and signed off by Platform/Ops + QA.
