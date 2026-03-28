#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
ARTIFACT_DIR="$REPO_ROOT/docs/qa/artifacts"
TIMESTAMP="$(date -u +%Y%m%dT%H%M%SZ)"
LOG_FILE="${1:-$ARTIFACT_DIR/sre_smartbot_aiops_eval_${TIMESTAMP}.log}"

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

  log "# SRE Smart Bot AIOPS Evaluation Harness"
  log "timestamp_utc: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
  log "repo_root: $REPO_ROOT"
  log "log_file: $LOG_FILE"
  log "branch: $(cd "$REPO_ROOT" && git branch --show-current)"

  run_check "backend_aiops_replay_suite" \
    bash -lc "cd '$REPO_ROOT/backend' && go test ./internal/application/sresmartbot -run 'TestAIOpsEvaluationHarness_ReplaySuite' -count=1 -v" || overall_rc=1

  run_check "backend_interpretation_safety_guard" \
    bash -lc "cd '$REPO_ROOT/backend' && go test ./internal/application/sresmartbot -run 'TestBuildBoundedSummaries_RejectsUnsafeGeneratedOutput' -count=1 -v" || overall_rc=1

  run_check "backend_sresmartbot_regression_smoke" \
    bash -lc "cd '$REPO_ROOT/backend' && go test ./internal/application/sresmartbot -run 'Test(BuildTriageFromDraft_|BuildSeverityFromDraft_|BuildSuggestedActionFromDraft_)' -count=1" || overall_rc=1

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
