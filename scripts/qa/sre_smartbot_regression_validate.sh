#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
ARTIFACT_DIR="$REPO_ROOT/docs/qa/artifacts"
TIMESTAMP="$(date -u +%Y%m%dT%H%M%SZ)"
LOG_FILE="${1:-$ARTIFACT_DIR/sre_smartbot_regression_${TIMESTAMP}.log}"

PASS_COUNT=0
FAIL_COUNT=0

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

  log "# SRE Smart Bot Regression Runner"
  log "timestamp_utc: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
  log "repo_root: $REPO_ROOT"
  log "log_file: $LOG_FILE"
  log "branch: $(cd "$REPO_ROOT" && git branch --show-current)"

  run_check "backend_sresmartbot_suite" \
    bash -lc "cd '$REPO_ROOT/backend' && go test ./internal/application/sresmartbot -count=1" || overall_rc=1

  run_check "backend_sresmartbot_rest_contracts" \
    bash -lc "cd '$REPO_ROOT/backend' && go test ./internal/adapters/primary/rest -run SRESmartBot -count=1" || overall_rc=1

  run_check "backend_sresmartbot_focus_slice" \
    bash -lc "cd '$REPO_ROOT/backend' && go test ./internal/application/sresmartbot -run 'Test(BuildDraft|DemoService|ObserveAsyncBacklogSignals_|ObserveNATSConsumerLagSignals_|BuildIncidentWorkspace_IncludesMessagingConsumerSummaryAndBundle)' -count=1" || overall_rc=1

  local frontend_candidates=(
    "src/pages/admin/__tests__/SRESmartBotIncidentsPage.test.tsx"
    "src/pages/admin/__tests__/sreSmartBotAsyncSummary.test.ts"
  )
  local frontend_tests=()
  local candidate
  for candidate in "${frontend_candidates[@]}"; do
    if [[ -f "$REPO_ROOT/frontend/$candidate" ]]; then
      frontend_tests+=("$candidate")
    fi
  done

  if [[ "${#frontend_tests[@]}" -gt 0 ]]; then
    local frontend_args
    frontend_args="$(printf '%q ' "${frontend_tests[@]}")"
    run_check "frontend_sresmartbot_focus_slice" \
      bash -lc "cd '$REPO_ROOT/frontend' && npm test -- --run ${frontend_args}" || overall_rc=1
  else
    log ""
    log "=== CHECK: frontend_sresmartbot_focus_slice ==="
    log "RESULT: SKIP (no focused SRE frontend test files present)"
    PASS_COUNT=$((PASS_COUNT + 1))
  fi

  log ""
  log "=== SUMMARY ==="
  log "pass: $PASS_COUNT"
  log "fail: $FAIL_COUNT"
  if [[ "$overall_rc" -ne 0 ]]; then
    log "status: FAILED"
  else
    log "status: OK"
  fi

  return "$overall_rc"
}

main "$@"
