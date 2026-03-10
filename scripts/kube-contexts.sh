#!/usr/bin/env bash
set -euo pipefail

# Simple kube context helper:
# - list all contexts
# - show current context
# - switch by context name
# - interactive pick (fzf/select)

SCRIPT_NAME="$(basename "$0")"

usage() {
  cat <<USAGE
Usage:
  $SCRIPT_NAME list
  $SCRIPT_NAME current
  $SCRIPT_NAME switch <context-name>
  $SCRIPT_NAME pick
  $SCRIPT_NAME help

Notes:
  - Uses your default kubeconfig resolution (KUBECONFIG or ~/.kube/config).
  - If 'fzf' is installed, 'pick' uses fuzzy search; otherwise it uses a numbered menu.
USAGE
}

require_kubectl() {
  if ! command -v kubectl >/dev/null 2>&1; then
    echo "Error: kubectl is required but not found in PATH." >&2
    exit 1
  fi
}

list_contexts() {
  require_kubectl
  local current
  current="$(kubectl config current-context 2>/dev/null || true)"

  if [[ -z "$current" ]]; then
    echo "No current context is set."
  else
    echo "Current context: $current"
  fi
  echo

  local contexts
  contexts="$(kubectl config get-contexts -o name 2>/dev/null || true)"
  if [[ -z "$contexts" ]]; then
    echo "No contexts found."
    return 0
  fi

  echo "Available contexts:"
  while IFS= read -r ctx; do
    if [[ "$ctx" == "$current" ]]; then
      echo "* $ctx"
    else
      echo "  $ctx"
    fi
  done <<< "$contexts"
}

current_context() {
  require_kubectl
  kubectl config current-context
}

switch_context() {
  require_kubectl
  if [[ $# -lt 1 ]]; then
    echo "Error: missing context name." >&2
    usage
    exit 1
  fi

  local target="$1"
  local exists
  exists="$(kubectl config get-contexts -o name | grep -Fx "$target" || true)"
  if [[ -z "$exists" ]]; then
    echo "Error: context '$target' not found." >&2
    echo
    list_contexts
    exit 1
  fi

  kubectl config use-context "$target" >/dev/null
  echo "Switched to context: $target"
}

pick_context() {
  require_kubectl

  local contexts
  contexts="$(kubectl config get-contexts -o name 2>/dev/null || true)"
  if [[ -z "$contexts" ]]; then
    echo "No contexts found."
    exit 1
  fi

  local selected=""
  if command -v fzf >/dev/null 2>&1; then
    selected="$(printf '%s\n' "$contexts" | fzf --prompt='kube context> ' --height=40% --reverse || true)"
  else
    echo "fzf not found; using numbered menu:"
    local options=()
    while IFS= read -r line; do
      options+=("$line")
    done <<< "$contexts"
    select choice in "${options[@]}"; do
      selected="$choice"
      break
    done
  fi

  if [[ -z "$selected" ]]; then
    echo "No context selected."
    exit 1
  fi

  switch_context "$selected"
}

main() {
  local cmd="${1:-list}"
  case "$cmd" in
    list)
      list_contexts
      ;;
    current)
      current_context
      ;;
    switch)
      shift
      switch_context "$@"
      ;;
    pick)
      pick_context
      ;;
    help|-h|--help)
      usage
      ;;
    *)
      echo "Error: unknown command '$cmd'." >&2
      echo
      usage
      exit 1
      ;;
  esac
}

main "$@"
