# Packer VM Lifecycle Provider-Native Smoke Runbook

Last updated: 2026-03-27

## Purpose

Validate provider-native VM lifecycle transitions (`promote`, `deprecate`, `delete`) against real provider APIs for disposable VM image executions.

## Script

- `scripts/packer-lifecycle-provider-native-smoke.sh`

## Required Environment

Core API access:

- `BASE_URL` (default: `http://localhost:8080`)
- `AUTH_TOKEN` (required)
- `TENANT_ID` (required)
- `EXECUTION_IDS` (required, comma-separated VM execution IDs)

Safety + behavior:

- `CONFIRM_DESTRUCTIVE=true` (required; delete is destructive)
- `ACTION_SEQUENCE` (default: `promote,deprecate,delete`)
- `REQUIRE_PROVIDER_NATIVE` (default: `true`)
- `EXPECTED_PROVIDER` (optional; one of `aws|vmware|azure|gcp`)
- `REQUEST_TIMEOUT_SECONDS` (default: `30`)
- `REASON_PREFIX` (default: `provider-native smoke`)

Lifecycle execution mode + provider toggles:

- `IF_VM_LIFECYCLE_EXECUTION_MODE=require_provider_native` (recommended for smoke)
- `IF_VM_LIFECYCLE_PROVIDER_AWS_ENABLED=true`
- `IF_VM_LIFECYCLE_PROVIDER_VMWARE_ENABLED=true`
- `IF_VM_LIFECYCLE_PROVIDER_AZURE_ENABLED=true`
- `IF_VM_LIFECYCLE_PROVIDER_GCP_ENABLED=true`

Provider credentials/config must already be configured in backend runtime:

- AWS: `IF_VM_LIFECYCLE_AWS_REGION` (when needed for bare AMI identifiers)
- VMware: `IF_VM_LIFECYCLE_VMWARE_VCENTER_URL`, `IF_VM_LIFECYCLE_VMWARE_USERNAME`, `IF_VM_LIFECYCLE_VMWARE_PASSWORD` (plus optional datacenter/insecure)
- Azure: `IF_VM_LIFECYCLE_AZURE_BEARER_TOKEN` (plus optional API version/deprecation hours)
- GCP: `IF_VM_LIFECYCLE_GCP_BEARER_TOKEN` (plus optional base URL)

## Execution Order

1. Choose disposable VM executions per provider (never production-critical images).
2. Run per provider for clear evidence separation:

```bash
BASE_URL=http://localhost:8080 \
AUTH_TOKEN='<token>' \
TENANT_ID='<tenant-uuid>' \
EXECUTION_IDS='<execution-uuid>' \
EXPECTED_PROVIDER=aws \
CONFIRM_DESTRUCTIVE=true \
REQUIRE_PROVIDER_NATIVE=true \
ACTION_SEQUENCE=promote,deprecate,delete \
./scripts/packer-lifecycle-provider-native-smoke.sh
```

3. Repeat for `vmware`, `azure`, `gcp` with provider-specific disposable execution IDs.

## Pass Criteria

For each action:

- API returns `200`.
- Response includes non-empty lifecycle payload fields.
- `lifecycle_transition_mode` is `provider_native` or `hybrid`.
- Provider audit fields are populated:
  - `lifecycle_last_provider_action`
  - `lifecycle_last_provider_identifier`
  - `lifecycle_last_provider_outcome=success`

## Rollback / Safety

- `delete` is irreversible for many provider image paths.
- Use only disposable images for smoke.
- If provider-native behavior must be paused quickly, set provider toggle(s) to `false`:
  - `IF_VM_LIFECYCLE_PROVIDER_<PROVIDER>_ENABLED=false`
- If global rollback is required, set:
  - `IF_VM_LIFECYCLE_EXECUTION_MODE=metadata_only`

## Notes

- Script intentionally exits fast on missing env vars and non-`200` responses.
- Script does not create new images; it validates lifecycle transitions for existing execution IDs.
