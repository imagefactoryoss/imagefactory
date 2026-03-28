# Packer VM Lifecycle Provider-Native Matrix Validation Runbook

Last updated: 2026-03-27

## Purpose

Run provider-native lifecycle smoke validation across multiple providers in one operator command and capture a consolidated evidence log.

## Script

- `scripts/packer-lifecycle-provider-native-matrix.sh`

## Required Environment

Shared API access:

- `BASE_URL` (default: `http://localhost:8080`)
- `AUTH_TOKEN` (required)
- `TENANT_ID` (required)

Provider selection and execution IDs:

- `TARGET_PROVIDERS` (default: `aws,vmware,azure,gcp`)
- `AWS_EXECUTION_IDS` (required for `aws` target)
- `VMWARE_EXECUTION_IDS` (required for `vmware` target)
- `AZURE_EXECUTION_IDS` (required for `azure` target)
- `GCP_EXECUTION_IDS` (required for `gcp` target)
- `FAIL_ON_MISSING_PROVIDER` (default: `true`)

Behavior:

- `CONFIRM_DESTRUCTIVE=true` (required for delete path)
- `ACTION_SEQUENCE` (default: `promote,deprecate,delete`)
- `SMOKE_MODE` (default: `api`; set `mock_success` for no-cloud deterministic validation)
- `REQUIRE_PROVIDER_NATIVE` (default: `true`)
- `REQUEST_TIMEOUT_SECONDS` (default: `30`)
- `REASON_PREFIX` (default: `provider-native matrix`)
- `REPORT_FILE` (default: `/tmp/packer-provider-native-matrix-<timestamp>.log`)

Backend runtime should use provider-native execution and enabled provider toggles during validation:

- `IF_VM_LIFECYCLE_EXECUTION_MODE=require_provider_native`
- `IF_VM_LIFECYCLE_PROVIDER_AWS_ENABLED=true`
- `IF_VM_LIFECYCLE_PROVIDER_VMWARE_ENABLED=true`
- `IF_VM_LIFECYCLE_PROVIDER_AZURE_ENABLED=true`
- `IF_VM_LIFECYCLE_PROVIDER_GCP_ENABLED=true`

## Example

```bash
BASE_URL=http://localhost:8080 \
AUTH_TOKEN='<token>' \
TENANT_ID='<tenant-uuid>' \
AWS_EXECUTION_IDS='<aws-execution-uuid>' \
VMWARE_EXECUTION_IDS='<vmware-execution-uuid>' \
AZURE_EXECUTION_IDS='<azure-execution-uuid>' \
GCP_EXECUTION_IDS='<gcp-execution-uuid>' \
CONFIRM_DESTRUCTIVE=true \
REQUIRE_PROVIDER_NATIVE=true \
./scripts/packer-lifecycle-provider-native-matrix.sh
```

No-cloud mock example:

```bash
TARGET_PROVIDERS=aws,vmware,azure,gcp \
AWS_EXECUTION_IDS='mock-aws-1' \
VMWARE_EXECUTION_IDS='mock-vmware-1' \
AZURE_EXECUTION_IDS='mock-azure-1' \
GCP_EXECUTION_IDS='mock-gcp-1' \
CONFIRM_DESTRUCTIVE=true \
REQUIRE_PROVIDER_NATIVE=true \
SMOKE_MODE=mock_success \
./scripts/packer-lifecycle-provider-native-matrix.sh
```

## Evidence and Pass Criteria

The script writes a consolidated report file with per-provider sections and result lines.

Validation passes when:

- each targeted provider reports `status=pass`;
- no targeted provider reports `status=fail`;
- report includes final summary with `fail=0`.

## Safety

- Use disposable execution IDs only.
- `delete` actions are irreversible for many provider image paths.
- For emergency fallback, set `IF_VM_LIFECYCLE_EXECUTION_MODE=metadata_only`.
