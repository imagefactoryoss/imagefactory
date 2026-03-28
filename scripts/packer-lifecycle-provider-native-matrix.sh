#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SMOKE_SCRIPT="${SCRIPT_DIR}/packer-lifecycle-provider-native-smoke.sh"

BASE_URL="${BASE_URL:-http://localhost:8080}"
AUTH_TOKEN="${AUTH_TOKEN:-}"
TENANT_ID="${TENANT_ID:-}"

TARGET_PROVIDERS="${TARGET_PROVIDERS:-aws,vmware,azure,gcp}"
ACTION_SEQUENCE="${ACTION_SEQUENCE:-promote,deprecate,delete}"
REQUIRE_PROVIDER_NATIVE="${REQUIRE_PROVIDER_NATIVE:-true}"
CONFIRM_DESTRUCTIVE="${CONFIRM_DESTRUCTIVE:-false}"
REQUEST_TIMEOUT_SECONDS="${REQUEST_TIMEOUT_SECONDS:-30}"
REASON_PREFIX="${REASON_PREFIX:-provider-native matrix}"
FAIL_ON_MISSING_PROVIDER="${FAIL_ON_MISSING_PROVIDER:-true}"

AWS_EXECUTION_IDS="${AWS_EXECUTION_IDS:-}"
VMWARE_EXECUTION_IDS="${VMWARE_EXECUTION_IDS:-}"
AZURE_EXECUTION_IDS="${AZURE_EXECUTION_IDS:-}"
GCP_EXECUTION_IDS="${GCP_EXECUTION_IDS:-}"

REPORT_FILE="${REPORT_FILE:-/tmp/packer-provider-native-matrix-$(date -u +%Y%m%dT%H%M%SZ).log}"

PASS_COUNT=0
SKIP_COUNT=0
FAIL_COUNT=0
OVERALL_FAILED="false"

log() {
  printf '[vm-lifecycle-matrix] %s\n' "$*"
}

fail() {
  printf '[vm-lifecycle-matrix] ERROR: %s\n' "$*" >&2
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "missing required command: $1"
}

need_env() {
  local key="$1"
  local value="${!key:-}"
  [[ -n "$value" ]] || fail "missing required env var: $key"
}

trim() {
  local value="$1"
  value="${value#${value%%[![:space:]]*}}"
  value="${value%${value##*[![:space:]]}}"
  printf '%s' "$value"
}

normalize_bool() {
  local raw
  raw="$(printf '%s' "$1" | tr '[:upper:]' '[:lower:]' | xargs)"
  case "$raw" in
    true|1|yes|y|on) echo "true" ;;
    false|0|no|n|off) echo "false" ;;
    *) fail "invalid boolean value: $1" ;;
  esac
}

execution_ids_for_provider() {
  local provider="$1"
  case "$provider" in
    aws) printf '%s' "$AWS_EXECUTION_IDS" ;;
    vmware) printf '%s' "$VMWARE_EXECUTION_IDS" ;;
    azure) printf '%s' "$AZURE_EXECUTION_IDS" ;;
    gcp) printf '%s' "$GCP_EXECUTION_IDS" ;;
    *) fail "unsupported provider in TARGET_PROVIDERS: $provider" ;;
  esac
}

append_report() {
  printf '%s\n' "$1" >> "$REPORT_FILE"
}

run_provider_smoke() {
  local provider="$1"
  local ids="$2"

  local marker_start marker_end
  marker_start="----- provider=${provider} start $(date -u +%Y-%m-%dT%H:%M:%SZ) -----"
  marker_end="----- provider=${provider} end $(date -u +%Y-%m-%dT%H:%M:%SZ) -----"

  append_report "$marker_start"

  set +e
  BASE_URL="$BASE_URL" \
  AUTH_TOKEN="$AUTH_TOKEN" \
  TENANT_ID="$TENANT_ID" \
  EXECUTION_IDS="$ids" \
  EXPECTED_PROVIDER="$provider" \
  ACTION_SEQUENCE="$ACTION_SEQUENCE" \
  REQUIRE_PROVIDER_NATIVE="$REQUIRE_PROVIDER_NATIVE" \
  CONFIRM_DESTRUCTIVE="$CONFIRM_DESTRUCTIVE" \
  REQUEST_TIMEOUT_SECONDS="$REQUEST_TIMEOUT_SECONDS" \
  REASON_PREFIX="$REASON_PREFIX [$provider]" \
  "$SMOKE_SCRIPT" >> "$REPORT_FILE" 2>&1
  local exit_code=$?
  set -e

  append_report "$marker_end"

  if [[ $exit_code -eq 0 ]]; then
    PASS_COUNT=$((PASS_COUNT + 1))
    log "provider=${provider} status=pass"
    append_report "RESULT provider=${provider} status=pass"
    return 0
  fi

  FAIL_COUNT=$((FAIL_COUNT + 1))
  OVERALL_FAILED="true"
  log "provider=${provider} status=fail"
  append_report "RESULT provider=${provider} status=fail"
  return 1
}

main() {
  need_cmd bash
  need_cmd date
  need_cmd tee
  need_env AUTH_TOKEN
  need_env TENANT_ID

  [[ -x "$SMOKE_SCRIPT" ]] || fail "smoke script not found or not executable: $SMOKE_SCRIPT"
  : > "$REPORT_FILE"

  local fail_on_missing
  fail_on_missing="$(normalize_bool "$FAIL_ON_MISSING_PROVIDER")"

  log "target_providers=$TARGET_PROVIDERS report_file=$REPORT_FILE"
  append_report "vm lifecycle provider-native matrix run"
  append_report "started_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  append_report "target_providers=$TARGET_PROVIDERS"
  append_report "action_sequence=$ACTION_SEQUENCE"
  append_report "require_provider_native=$REQUIRE_PROVIDER_NATIVE"

  local provider raw_ids ids
  IFS=',' read -r -a providers <<< "$TARGET_PROVIDERS"
  for provider in "${providers[@]}"; do
    provider="$(trim "$provider")"
    [[ -n "$provider" ]] || continue

    raw_ids="$(execution_ids_for_provider "$provider")"
    ids="$(trim "$raw_ids")"
    if [[ -z "$ids" ]]; then
      if [[ "$fail_on_missing" == "true" ]]; then
        fail "missing execution IDs for provider=${provider}; set $(printf '%s' "$provider" | tr '[:lower:]' '[:upper:]')_EXECUTION_IDS"
      fi
      SKIP_COUNT=$((SKIP_COUNT + 1))
      log "provider=${provider} status=skipped reason=no execution IDs"
      append_report "RESULT provider=${provider} status=skipped reason=no_execution_ids"
      continue
    fi

    run_provider_smoke "$provider" "$ids" || true
  done

  append_report "completed_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  append_report "summary pass=${PASS_COUNT} fail=${FAIL_COUNT} skipped=${SKIP_COUNT}"

  log "summary pass=${PASS_COUNT} fail=${FAIL_COUNT} skipped=${SKIP_COUNT}"
  log "report_file=$REPORT_FILE"

  if [[ "$OVERALL_FAILED" == "true" ]]; then
    fail "one or more providers failed smoke validation"
  fi

  log "✅ provider-native matrix validation finished"
}

main "$@"
