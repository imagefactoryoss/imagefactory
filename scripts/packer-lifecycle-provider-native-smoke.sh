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
SMOKE_MODE="${SMOKE_MODE:-api}"
MOCK_INITIAL_STATE="${MOCK_INITIAL_STATE:-released}"
MOCK_TRANSITION_MODE="${MOCK_TRANSITION_MODE:-provider_native}"
MOCK_PROVIDER_OUTCOME="${MOCK_PROVIDER_OUTCOME:-success}"
MOCK_PROVIDER_DEFAULT="${MOCK_PROVIDER_DEFAULT:-aws}"

API_BODY=""
API_STATUS=""
CURRENT_PROVIDER=""
CURRENT_STATE=""

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

normalize_smoke_mode() {
  local raw
  raw="$(printf '%s' "$1" | tr '[:upper:]' '[:lower:]' | xargs)"
  case "$raw" in
    api|mock_success) echo "$raw" ;;
    *) fail "unsupported SMOKE_MODE: $1 (expected api|mock_success)" ;;
  esac
}

normalize_transition_mode() {
  local raw
  raw="$(printf '%s' "$1" | tr '[:upper:]' '[:lower:]' | xargs)"
  case "$raw" in
    metadata_only|provider_native|hybrid) echo "$raw" ;;
    *) fail "unsupported MOCK_TRANSITION_MODE: $1" ;;
  esac
}

trim() {
  local value="$1"
  value="${value#${value%%[![:space:]]*}}"
  value="${value%${value##*[![:space:]]}}"
  printf '%s' "$value"
}

mock_set_payload() {
  local execution_id="$1"
  local provider="$2"
  local state="$3"
  local message="$4"

  local mode provider_action provider_identifier provider_outcome
  mode="$(normalize_transition_mode "$MOCK_TRANSITION_MODE")"
  provider_action=""
  provider_identifier=""
  provider_outcome=""

  if [[ "$mode" == "provider_native" || "$mode" == "hybrid" ]]; then
    provider_action="mock_${state}"
    provider_identifier="mock://${provider}/${execution_id}"
    provider_outcome="$MOCK_PROVIDER_OUTCOME"
  fi

  API_BODY="$(jq -cn \
    --arg execution_id "$execution_id" \
    --arg provider "$provider" \
    --arg state "$state" \
    --arg mode "$mode" \
    --arg provider_action "$provider_action" \
    --arg provider_identifier "$provider_identifier" \
    --arg provider_outcome "$provider_outcome" \
    --arg message "$message" \
    '{
      message: $message,
      data: {
        execution_id: $execution_id,
        target_provider: $provider,
        lifecycle_state: $state,
        lifecycle_transition_mode: $mode,
        lifecycle_last_provider_action: (if ($provider_action | length) > 0 then $provider_action else null end),
        lifecycle_last_provider_identifier: (if ($provider_identifier | length) > 0 then $provider_identifier else null end),
        lifecycle_last_provider_outcome: (if ($provider_outcome | length) > 0 then $provider_outcome else null end)
      }
    }')"
  API_STATUS="200"
}

mock_fetch_item() {
  local execution_id="$1"
  local provider

  provider="$(printf '%s' "${EXPECTED_PROVIDER:-$MOCK_PROVIDER_DEFAULT}" | tr '[:upper:]' '[:lower:]')"
  [[ -n "$provider" ]] || fail "mock mode could not resolve provider"
  CURRENT_PROVIDER="$provider"
  CURRENT_STATE="$(printf '%s' "$MOCK_INITIAL_STATE" | tr '[:upper:]' '[:lower:]')"
  mock_set_payload "$execution_id" "$CURRENT_PROVIDER" "$CURRENT_STATE" "mock fetch lifecycle state"
}

mock_apply_action() {
  local execution_id="$1"
  local action="$2"

  case "$action" in
    promote) CURRENT_STATE="released" ;;
    deprecate) CURRENT_STATE="deprecated" ;;
    delete) CURRENT_STATE="deleted" ;;
    *) fail "unsupported action in ACTION_SEQUENCE: $action" ;;
  esac

  mock_set_payload "$execution_id" "$CURRENT_PROVIDER" "$CURRENT_STATE" "mock ${action} applied"
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
  if [[ "$SMOKE_MODE" == "mock_success" ]]; then
    mock_fetch_item "$execution_id"
    validate_item_shape "$API_BODY"
    return
  fi

  api_call "GET" "/api/v1/images/vm/${execution_id}"
  expect_status "200"
  validate_item_shape "$API_BODY"
}

apply_action() {
  local execution_id="$1"
  local action="$2"
  local reason="$3"

  if [[ "$SMOKE_MODE" == "mock_success" ]]; then
    mock_apply_action "$execution_id" "$action"
    validate_item_shape "$API_BODY"
    return
  fi

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
  need_cmd jq
  need_env EXECUTION_IDS
  SMOKE_MODE="$(normalize_smoke_mode "$SMOKE_MODE")"

  if [[ "$SMOKE_MODE" == "api" ]]; then
    need_cmd curl
    need_env AUTH_TOKEN
    need_env TENANT_ID
  fi

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

  log "✅ provider-native lifecycle smoke finished (mode=${SMOKE_MODE})"
}

main "$@"
