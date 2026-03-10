#!/usr/bin/env bash
set -euo pipefail

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "Missing dependency: $1" >&2
    exit 1
  }
}

need_cmd oci
need_cmd jq
need_cmd base64

usage() {
  cat <<USAGE
Usage:
  scripts/oke/disable-oke-shortname-enforcement.sh [options]

Options:
  --compartment-id <ocid>       OCI compartment OCID
  --cluster-id <ocid>           OKE cluster OCID
  --node-pool-id <ocid>         Optional specific node pool OCID (repeatable)
  --all-node-pools              Select all ACTIVE node pools in cluster
  --profile <name>              OCI CLI profile (default: DEFAULT)
  --region <region>             OCI region (optional)
  --max-unavailable <value>     Node cycling max unavailable (default: 10%)
  --max-surge <value>           Node cycling max surge (default: 10%)
  --wait-seconds <seconds>      Wait timeout per node pool (default: 7200)
  --non-interactive             Fail if required inputs are missing; no prompts
  --yes                         Skip final confirmation
  -h, --help                    Show this help

Examples:
  scripts/oke/disable-oke-shortname-enforcement.sh
  scripts/oke/disable-oke-shortname-enforcement.sh \
    --compartment-id ocid1.compartment... \
    --cluster-id ocid1.cluster... \
    --all-node-pools \
    --profile DEFAULT \
    --region us-phoenix-1
USAGE
}

read_input() {
  local prompt="$1"
  local default="${2:-}"
  local value=""
  read -r -p "$prompt${default:+ [$default]}: " value
  echo "${value:-$default}"
}

contains() {
  local needle="$1"
  shift
  local item
  for item in "$@"; do
    if [[ "$item" == "$needle" ]]; then
      return 0
    fi
  done
  return 1
}

PROFILE="DEFAULT"
REGION=""
COMPARTMENT_ID=""
CLUSTER_ID=""
MAX_UNAVAILABLE="10%"
MAX_SURGE="10%"
WAIT_SECONDS="7200"
NON_INTERACTIVE="false"
ASSUME_YES="false"
SELECT_ALL_NODE_POOLS="false"

SELECTED_NODE_POOL_IDS=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --compartment-id)
      COMPARTMENT_ID="${2:-}"
      shift 2
      ;;
    --cluster-id)
      CLUSTER_ID="${2:-}"
      shift 2
      ;;
    --node-pool-id)
      SELECTED_NODE_POOL_IDS+=("${2:-}")
      shift 2
      ;;
    --all-node-pools)
      SELECT_ALL_NODE_POOLS="true"
      shift
      ;;
    --profile)
      PROFILE="${2:-}"
      shift 2
      ;;
    --region)
      REGION="${2:-}"
      shift 2
      ;;
    --max-unavailable)
      MAX_UNAVAILABLE="${2:-}"
      shift 2
      ;;
    --max-surge)
      MAX_SURGE="${2:-}"
      shift 2
      ;;
    --wait-seconds)
      WAIT_SECONDS="${2:-}"
      shift 2
      ;;
    --non-interactive)
      NON_INTERACTIVE="true"
      shift
      ;;
    --yes)
      ASSUME_YES="true"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage
      exit 1
      ;;
  esac
done

if [[ "$NON_INTERACTIVE" != "true" ]]; then
  PROFILE="$(read_input "OCI CLI profile" "$PROFILE")"
  REGION="$(read_input "OCI region (optional, e.g. us-phoenix-1)" "$REGION")"
  if [[ -z "$COMPARTMENT_ID" ]]; then
    COMPARTMENT_ID="$(read_input "Compartment OCID")"
  fi
  if [[ -z "$CLUSTER_ID" ]]; then
    CLUSTER_ID="$(read_input "Cluster OCID")"
  fi
fi

if [[ -z "$COMPARTMENT_ID" || -z "$CLUSTER_ID" ]]; then
  echo "Both --compartment-id and --cluster-id are required." >&2
  exit 1
fi

OCI_ARGS=(--profile "$PROFILE")
if [[ -n "$REGION" ]]; then
  OCI_ARGS+=(--region "$REGION")
fi

echo "Listing ACTIVE node pools for cluster: $CLUSTER_ID"
NODE_POOLS_JSON="$({ oci "${OCI_ARGS[@]}" ce node-pool list --compartment-id "$COMPARTMENT_ID" --cluster-id "$CLUSTER_ID" --all; } )"

IFS=$'\n' ACTIVE_IDS=($(jq -r '.data[] | select(."lifecycle-state"=="ACTIVE") | .id' <<<"$NODE_POOLS_JSON"))
IFS=$'\n' ACTIVE_NAMES=($(jq -r '.data[] | select(."lifecycle-state"=="ACTIVE") | .name' <<<"$NODE_POOLS_JSON"))
unset IFS

if [[ "${#ACTIVE_IDS[@]}" -eq 0 ]]; then
  echo "No ACTIVE node pools found in cluster." >&2
  exit 1
fi

TARGET_NODE_POOLS=()

if [[ "$SELECT_ALL_NODE_POOLS" == "true" ]]; then
  TARGET_NODE_POOLS=("${ACTIVE_IDS[@]}")
elif [[ "${#SELECTED_NODE_POOL_IDS[@]}" -gt 0 ]]; then
  for np in "${SELECTED_NODE_POOL_IDS[@]}"; do
    if contains "$np" "${ACTIVE_IDS[@]}"; then
      TARGET_NODE_POOLS+=("$np")
    else
      echo "Skipping non-active or unknown node pool: $np" >&2
    fi
  done
else
  if [[ "$NON_INTERACTIVE" == "true" ]]; then
    echo "In --non-interactive mode, specify --node-pool-id or --all-node-pools." >&2
    exit 1
  fi

  echo ""
  echo "Available ACTIVE node pools:"
  for i in "${!ACTIVE_IDS[@]}"; do
    echo "  $((i + 1))) ${ACTIVE_NAMES[$i]} (${ACTIVE_IDS[$i]})"
  done
  echo "  a) All ACTIVE node pools"

  CHOICE="$(read_input "Choose node pool number or 'a'" "a")"
  if [[ "$CHOICE" == "a" || "$CHOICE" == "A" ]]; then
    TARGET_NODE_POOLS=("${ACTIVE_IDS[@]}")
  else
    if ! [[ "$CHOICE" =~ ^[0-9]+$ ]]; then
      echo "Invalid choice: $CHOICE" >&2
      exit 1
    fi
    idx=$((CHOICE - 1))
    if [[ $idx -lt 0 || $idx -ge ${#ACTIVE_IDS[@]} ]]; then
      echo "Invalid choice: $CHOICE" >&2
      exit 1
    fi
    TARGET_NODE_POOLS=("${ACTIVE_IDS[$idx]}")
  fi
fi

if [[ "${#TARGET_NODE_POOLS[@]}" -eq 0 ]]; then
  echo "No target node pools selected." >&2
  exit 1
fi

if [[ "$NON_INTERACTIVE" != "true" ]]; then
  MAX_UNAVAILABLE="$(read_input "Max unavailable for cycling" "$MAX_UNAVAILABLE")"
  MAX_SURGE="$(read_input "Max surge for cycling" "$MAX_SURGE")"
  WAIT_SECONDS="$(read_input "Wait timeout seconds" "$WAIT_SECONDS")"
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

CLOUD_INIT_FILE="$tmpdir/shortname-disable-cloudinit.sh"
cat > "$CLOUD_INIT_FILE" <<'SCRIPT'
#!/bin/bash
set -euxo pipefail

mkdir -p /etc/crio/crio.conf.d
cat >/etc/crio/crio.conf.d/11-default.conf <<'CFG'
[crio]
  [crio.image]
    short_name_mode="disabled"
CFG

curl --fail -H "Authorization: Bearer Oracle" -L0 \
  http://169.254.169.254/opc/v2/instance/metadata/oke_init_script \
  | base64 --decode >/var/run/oke-init.sh

bash /var/run/oke-init.sh
SCRIPT

USER_DATA_B64="$(base64 < "$CLOUD_INIT_FILE" | tr -d '\n')"

echo ""
echo "Target node pools (${#TARGET_NODE_POOLS[@]}):"
for np in "${TARGET_NODE_POOLS[@]}"; do
  echo "  - $np"
done
echo "Cycling config: maxUnavailable=$MAX_UNAVAILABLE, maxSurge=$MAX_SURGE"
echo ""

if [[ "$ASSUME_YES" != "true" ]]; then
  CONFIRM="$(read_input "Proceed with update and node cycling? (yes/no)" "no")"
  if [[ "$CONFIRM" != "yes" ]]; then
    echo "Aborted."
    exit 0
  fi
fi

for np in "${TARGET_NODE_POOLS[@]}"; do
  echo "Updating node pool: $np"

  NP_GET_JSON="$(oci "${OCI_ARGS[@]}" ce node-pool get --node-pool-id "$np")"
  MERGED_METADATA="$({ jq -c --arg ud "$USER_DATA_B64" '(.data."node-metadata" // {}) + {"user_data": $ud}' <<<"$NP_GET_JSON"; } )"

  if ! oci "${OCI_ARGS[@]}" ce node-pool update \
    --node-pool-id "$np" \
    --node-metadata "$MERGED_METADATA" \
    --node-pool-cycling-details "{\"isNodeCyclingEnabled\":true,\"cycleModes\":[\"INSTANCE_REPLACE\"],\"maximumUnavailable\":\"$MAX_UNAVAILABLE\",\"maximumSurge\":\"$MAX_SURGE\"}" \
    --force \
    --wait-for-state SUCCEEDED \
    --max-wait-seconds "$WAIT_SECONDS" \
    >/dev/null 2>"$tmpdir/update_error.log"; then
    if grep -q "restricted to Enhanced clusters" "$tmpdir/update_error.log"; then
      echo "Node pool cycling is not supported on this BASIC cluster. Applying metadata update without automatic cycling."
      oci "${OCI_ARGS[@]}" ce node-pool update \
        --node-pool-id "$np" \
        --node-metadata "$MERGED_METADATA" \
        --force \
        --wait-for-state SUCCEEDED \
        --max-wait-seconds "$WAIT_SECONDS" \
        >/dev/null
      echo "Metadata updated. Recreate nodes in this pool to apply to existing workers."
    else
      cat "$tmpdir/update_error.log" >&2
      exit 1
    fi
  fi

  echo "Done: $np"
done

echo "Completed."
