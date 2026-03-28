#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
ARTIFACT_DIR="$REPO_ROOT/docs/qa/artifacts"
TIMESTAMP="$(date -u +%Y%m%dT%H%M%SZ)"
LOG_FILE="${1:-$ARTIFACT_DIR/packer_provider_native_matrix_validation_${TIMESTAMP}.log}"

MATRIX_SCRIPT="$REPO_ROOT/scripts/packer-lifecycle-provider-native-matrix.sh"

SMOKE_MODE="${SMOKE_MODE:-mock_success}"
TARGET_PROVIDERS="${TARGET_PROVIDERS:-aws,vmware,azure,gcp}"
ACTION_SEQUENCE="${ACTION_SEQUENCE:-promote,deprecate,delete}"
CONFIRM_DESTRUCTIVE="${CONFIRM_DESTRUCTIVE:-true}"
REQUIRE_PROVIDER_NATIVE="${REQUIRE_PROVIDER_NATIVE:-true}"
FAIL_ON_MISSING_PROVIDER="${FAIL_ON_MISSING_PROVIDER:-true}"

PASS_COUNT=0
FAIL_COUNT=0
SKIP_COUNT=0

log() {
  printf '%s\n' "$*" | tee -a "$LOG_FILE"
}

run_check() {
  local name="$1"
  shift
  log ""
  log "=== CHECK: ${name} ==="
  log "CMD: $*"
  if "$@" >>"$LOG_FILE" 2>&1; then
    log "RESULT: PASS"
    PASS_COUNT=$((PASS_COUNT + 1))
    return 0
  fi
  log "RESULT: FAIL"
  FAIL_COUNT=$((FAIL_COUNT + 1))
  return 1
}

main() {
  mkdir -p "$ARTIFACT_DIR"
  : >"$LOG_FILE"

  local overall_rc=0

  log "# Packer Provider-Native Matrix Validation Runner"
  log "timestamp_utc: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
  log "repo_root: $REPO_ROOT"
  log "log_file: $LOG_FILE"
  log "branch: $(cd "$REPO_ROOT" && git branch --show-current)"
  log "smoke_mode: $SMOKE_MODE"
  log "target_providers: $TARGET_PROVIDERS"
  log "action_sequence: $ACTION_SEQUENCE"
  log "require_provider_native: $REQUIRE_PROVIDER_NATIVE"

  if [[ ! -x "$MATRIX_SCRIPT" ]]; then
    log "ERROR: matrix script missing or not executable: $MATRIX_SCRIPT"
    exit 1
  fi

  run_check "matrix_runner" \
    bash -lc "cd '$REPO_ROOT' && \
      SMOKE_MODE='$SMOKE_MODE' \
      TARGET_PROVIDERS='$TARGET_PROVIDERS' \
      ACTION_SEQUENCE='$ACTION_SEQUENCE' \
      CONFIRM_DESTRUCTIVE='$CONFIRM_DESTRUCTIVE' \
      REQUIRE_PROVIDER_NATIVE='$REQUIRE_PROVIDER_NATIVE' \
      FAIL_ON_MISSING_PROVIDER='$FAIL_ON_MISSING_PROVIDER' \
      AWS_EXECUTION_IDS='${AWS_EXECUTION_IDS:-mock-aws-1}' \
      VMWARE_EXECUTION_IDS='${VMWARE_EXECUTION_IDS:-mock-vmware-1}' \
      AZURE_EXECUTION_IDS='${AZURE_EXECUTION_IDS:-mock-azure-1}' \
      GCP_EXECUTION_IDS='${GCP_EXECUTION_IDS:-mock-gcp-1}' \
      BASE_URL='${BASE_URL:-http://localhost:8080}' \
      AUTH_TOKEN='${AUTH_TOKEN:-}' \
      TENANT_ID='${TENANT_ID:-}' \
      ./scripts/packer-lifecycle-provider-native-matrix.sh" || overall_rc=1

  log ""
  log "=== SUMMARY ==="
  log "pass: $PASS_COUNT"
  log "fail: $FAIL_COUNT"
  log "skip: $SKIP_COUNT"

  if [[ "$overall_rc" -ne 0 ]]; then
    log "status: FAILED"
  else
    log "status: OK"
  fi

  return "$overall_rc"
}

main "$@"
