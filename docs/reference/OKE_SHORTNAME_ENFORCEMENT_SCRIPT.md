# OKE Shortname Enforcement Disable Script

Script:
- `scripts/oke/disable-oke-shortname-enforcement.sh`

Purpose:
- Disables CRI-O shortname enforcement on OKE workers by setting:
  - `short_name_mode="disabled"`
  - in `/etc/crio/crio.conf.d/11-default.conf`
- Persists this through node pool `node_metadata.user_data` so replacement/new nodes get it automatically.

Oracle reference:
- https://docs.oracle.com/en-us/iaas/Content/ContEng/Concepts/contengaboutk8sversions.htm

## Requirements
- `oci` CLI authenticated and working
- `jq`
- `base64`
- IAM permissions to read/update OKE node pools

## What The Script Changes
- Reads active node pools in a target cluster.
- Merges cloud-init into existing node metadata (`user_data`) for selected pools.
- Attempts node cycling with:
  - `INSTANCE_REPLACE`
  - configurable `maxUnavailable` and `maxSurge`
- If cluster type is BASIC and cycling API is rejected, script falls back to metadata-only update and prints follow-up action.

## Interactive Usage
```bash
scripts/oke/disable-oke-shortname-enforcement.sh
```

Prompts include:
- OCI profile
- region (optional)
- compartment OCID
- cluster OCID
- node pool choice (single or all active)
- rollout settings (`maxUnavailable`, `maxSurge`, timeout)

## Non-Interactive Usage
All active node pools:
```bash
scripts/oke/disable-oke-shortname-enforcement.sh \
  --compartment-id ocid1.compartment... \
  --cluster-id ocid1.cluster... \
  --all-node-pools \
  --profile DEFAULT \
  --region us-phoenix-1 \
  --yes \
  --non-interactive
```

Specific node pools:
```bash
scripts/oke/disable-oke-shortname-enforcement.sh \
  --compartment-id ocid1.compartment... \
  --cluster-id ocid1.cluster... \
  --node-pool-id ocid1.nodepool... \
  --node-pool-id ocid1.nodepool... \
  --profile DEFAULT \
  --region us-phoenix-1 \
  --yes \
  --non-interactive
```

## Flags
- `--compartment-id <ocid>`: compartment containing the cluster
- `--cluster-id <ocid>`: OKE cluster OCID
- `--node-pool-id <ocid>`: select one/more pools (repeatable)
- `--all-node-pools`: target all ACTIVE pools
- `--profile <name>`: OCI CLI profile (default `DEFAULT`)
- `--region <region>`: OCI region override
- `--max-unavailable <value>`: cycling max unavailable (default `10%`)
- `--max-surge <value>`: cycling max surge (default `10%`)
- `--wait-seconds <seconds>`: wait timeout per update (default `7200`)
- `--non-interactive`: disable prompts
- `--yes`: skip final confirmation

## Verification
Check node metadata:
```bash
oci ce node-pool get --node-pool-id <node_pool_ocid> --output json \
| jq -r '.data."node-metadata".user_data' | base64 --decode
```

Expected block:
```bash
[crio]
  [crio.image]
    short_name_mode="disabled"
```

## Important Behavior By Cluster Type
- Enhanced clusters:
  - script can update metadata and trigger node cycling in one run.
- Basic clusters:
  - cycling option is not supported by OCI API.
  - script applies metadata only and prints:
    - `Metadata updated. Recreate nodes in this pool to apply to existing workers.`

## Operational Notes
- Run during a maintenance window for production.
- Existing `user_data` is replaced by merged value; review if you rely on custom bootstrap logic.
- If metadata is updated but nodes were not cycled, manually replace nodes in affected pools.
