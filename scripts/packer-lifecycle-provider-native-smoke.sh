#!/usr/bin/env bash

set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
AUTH_TOKEN="${AUTH_TOKEN:-}"
TENANT_ID="${TENANT_ID:-}"
EXECUTION_IDS="${EXECUTION_IDS:-}"
EXPECTED_PROVIDER="${EXPECTED_PROVIDER:-}"
ACTION_SEQUENCE="${ACTION_SEQUENCE:-promote,deprecate,delete}"
REQUIRE_PROVIDER_NATIVE="${REQUIRE_PROVIDER_NATIVE:-true}"
CONFIRM_DESTRUCTIVE="${CONFIRM_DESTRUCTIVE:-false}"
REQUEST_TIMEOUT_SECONDS="${REQUEST_TIMEOUT_SECONDS:-30}"
REASON_PREFIX="${REASON_PREFIX:-provider-native smoke}"

API_BODY=""
API_STATUS=""

log() {
  printf '[vm-lifecycle-smoke] %s\n' "$*"
}

fail() {
  printf '[vm-lifecycle-smoke] ERROR: %s\n' "$*" >&2
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

normalize_bool() {
  local raw
  raw="$(printf '%s' "$1" | tr '[:upper:]' '[:lower:]' | xargs)"
  case "$raw" in
    true|1|yes|y|on) echo "true" ;;
    false|0|no|n|off) echo "false" ;;
    *) fail "invalid boolean value: $1" ;;
  esac
}

trim() {
  local value="$1"
  value="${value#${value%%[![:space:]]*}}"
  value="${value%${value##*[![:space:]]}}"
  printf '%s' "$value"
}

api_call() {
  local method="$1"
  local path="$2"
  local body="${3:-}"
  local response

  if [[ -n "$body" ]]; then
    response="$(curl -sS -X "$method" "${BASE_URL}${path}" \
      --max-time "${REQUEST_TIMEOUT_SECONDS}" \
      -H "Authorization: Bearer ${AUTH_TOKEN}" \
      -H "X-Tenant-ID: ${TENANT_ID}" \
      -H "Content-Type: application/json" \
      -d "$body" \
      -w $'\n%{http_code}')"
  else
    response="$(curl -sS -X "$method" "${BASE_URL}${path}" \
      --max-time "${REQUEST_TIMEOUT_SECONDS}" \
      -H "Authorization: Bearer ${AUTH_TOKEN}" \
      -H "X-Tenant-ID: ${TENANT_ID}" \
      -H "Content-Type: application/json" \
      -w $'\n%{http_code}')"
  fi

  API_BODY="$(printf '%s\n' "$response" | sed '$d')"
  API_STATUS="$(printf '%s\n' "$response" | tail -n 1)"
}

expect_status() {
  local expected="$1"
  if [[ "$API_STATUS" != "$expected" ]]; then
    printf '%s\n' "$API_BODY" >&2
    fail "unexpected HTTP status: got $API_STATUS expected $expected"
  fi
}

json_get() {
  local payload="$1"
  local query="$2"
  printf '%s' "$payload" | jq -r "$query"
}

validate_item_shape() {
  local payload="$1"
  printf '%s' "$payload" | jq -e '
    (.data.execution_id | type == "string") and
    (.data.target_provider | type == "string") and
    (.data.lifecycle_state | type == "string") and
    (.data.lifecycle_transition_mode | type == "string")
  ' >/dev/null || fail "unexpected vm image payload shape"
}

fetch_item() {
  local execution_id="$1"
  api_call "GET" "/api/v1/images/vm/${execution_id}"
  expect_status "200"
  validate_item_shape "$API_BODY"
}

apply_action() {
  local execution_id="$1"
  local action="$2"
  local reason="$3"

  case "$action" in
    promote)
      api_call "POST" "/api/v1/images/vm/${execution_id}/promote" "{}"
      ;;
    deprecate)
      api_call "POST" "/api/v1/images/vm/${execution_id}/deprecate" "{\"reason\":\"${reason}\"}"
      ;;
    delete)
      api_call "DELETE" "/api/v1/images/vm/${execution_id}" "{\"reason\":\"${reason}\"}"
      ;;
    *)
      fail "unsupported action in ACTION_SEQUENCE: $action"
      ;;
  esac

  expect_status "200"
  validate_item_shape "$API_BODY"
}

assert_provider_native_fields() {
  local payload="$1"
  local require_native="$2"

  local mode provider_action provider_id provider_outcome
  mode="$(json_get "$payload" '.data.lifecycle_transition_mode')"
  provider_action="$(json_get "$payload" '.data.lifecycle_last_provider_action // ""')"
  provider_id="$(json_get "$payload" '.data.lifecycle_last_provider_identifier // ""')"
  provider_outcome="$(json_get "$payload" '.data.lifecycle_last_provider_outcome // ""')"

  if [[ "$require_native" == "true" ]]; then
    if [[ "$mode" != "provider_native" && "$mode" != "hybrid" ]]; then
      fail "expected provider-native/hybrid transition mode, got: $mode"
    fi
    [[ -n "$provider_action" ]] || fail "expected lifecycle_last_provider_action to be set"
    [[ -n "$provider_id" ]] || fail "expected lifecycle_last_provider_identifier to be set"
    [[ "$provider_outcome" == "success" ]] || fail "expected lifecycle_last_provider_outcome=success, got: $provider_outcome"
  fi
}

run_for_execution() {
  local execution_id="$1"

  fetch_item "$execution_id"
  local provider state
  provider="$(json_get "$API_BODY" '.data.target_provider | ascii_downcase')"
  state="$(json_get "$API_BODY" '.data.lifecycle_state | ascii_downcase')"

  if [[ -n "$EXPECTED_PROVIDER" ]]; then
    local expected
    expected="$(printf '%s' "$EXPECTED_PROVIDER" | tr '[:upper:]' '[:lower:]')"
    if [[ "$provider" != "$expected" ]]; then
      fail "execution $execution_id provider mismatch: expected $expected got $provider"
    fi
  fi

  if [[ "$state" == "deleted" ]]; then
    fail "execution $execution_id is already deleted; use a disposable non-deleted artifact"
  fi

  log "execution=$execution_id provider=$provider initial_state=$state"

  local require_native
  require_native="$(normalize_bool "$REQUIRE_PROVIDER_NATIVE")"

  IFS=',' read -r -a actions <<< "$ACTION_SEQUENCE"
  local action reason now
  for action in "${actions[@]}"; do
    action="$(trim "$action")"
    [[ -n "$action" ]] || continue

    now="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
    reason="${REASON_PREFIX} ${action} ${provider} ${execution_id} ${now}"

    log "applying action=$action execution=$execution_id"
    apply_action "$execution_id" "$action" "$reason"

    local message new_state
    message="$(json_get "$API_BODY" '.message // ""')"
    new_state="$(json_get "$API_BODY" '.data.lifecycle_state | ascii_downcase')"
    log "action=$action state=$new_state message=${message}"

    assert_provider_native_fields "$API_BODY" "$require_native"
  done

  log "✅ execution $execution_id lifecycle smoke passed"
}

main() {
  need_cmd curl
  need_cmd jq
  need_env AUTH_TOKEN
  need_env TENANT_ID
  need_env EXECUTION_IDS

  local confirm
  confirm="$(normalize_bool "$CONFIRM_DESTRUCTIVE")"
  if [[ "$confirm" != "true" ]]; then
    fail "CONFIRM_DESTRUCTIVE must be true because ACTION_SEQUENCE may include delete"
  fi

  IFS=',' read -r -a exec_ids <<< "$EXECUTION_IDS"
  local execution_id
  for execution_id in "${exec_ids[@]}"; do
    execution_id="$(trim "$execution_id")"
    [[ -n "$execution_id" ]] || continue
    run_for_execution "$execution_id"
  done

  log "✅ provider-native lifecycle smoke finished"
}

main "$@"
