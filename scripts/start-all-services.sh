#!/bin/bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOG_DIR="$ROOT_DIR/logs"
ENV_FILE="$ROOT_DIR/.env.development"

mkdir -p "$LOG_DIR"

log_header() {
  local logfile="$1"
  local title="$2"
  : >"$logfile"
  echo "🚀 ================= ${title} STARTING $(date '+%Y-%m-%d %H:%M:%S') ==================" >>"$logfile"
}

kill_pattern() {
  local pattern="$1"
  pkill -f "$pattern" >/dev/null 2>&1 || true
}

kill_port() {
  local port="$1"
  local pids
  pids="$(lsof -ti:"$port" 2>/dev/null || true)"
  if [[ -n "$pids" ]]; then
    kill $pids >/dev/null 2>&1 || true
  fi
}

start_bg() {
  local name="$1"
  local cmd="$2"
  local logfile="$3"
  local startup_note="${4:-}"

  echo "▶️  Starting $name..."
  log_header "$logfile" "$name"
  if [[ -n "$startup_note" ]]; then
    echo "$startup_note" >>"$logfile"
  fi
  /bin/bash -lc "$cmd" >>"$logfile" 2>&1 &
  echo "✅ $name started"
}

echo "🚀 Starting all development services..."

echo "🧹 Cleaning existing processes..."
kill_pattern "docs-server"
kill_port 8000
kill_pattern "mailpit"
kill_pattern "nats-server"
kill_pattern "go run.*cmd/email-worker"
kill_pattern "email-worker"
kill_pattern "go run.*cmd/notification-worker"
kill_pattern "notification-worker"
kill_pattern "go run.*cmd/internal-registry-gc-worker"
kill_pattern "internal-registry-gc-worker"
kill_pattern "redis-server"
kill_pattern "go run.*cmd/external-tenant-service"
kill_pattern "external-tenant-service"
kill_pattern "glauth"
kill_pattern "go run.*cmd/server"
kill_pattern "if-backend"
kill_port 8080
kill_pattern "go run.*cmd/dispatcher"
kill_pattern "dispatcher"
kill_pattern "npm run dev"
kill_pattern "vite"
kill_pattern "node.*dev"
sleep 2

start_bg \
  "Docs Server" \
  "cd \"$ROOT_DIR/backend\" && env GOCACHE=/tmp/go-build go run ./cmd/docs-server --root ../docs --port 8000" \
  "$LOG_DIR/docs-server.log" \
  "📚 Serving docs at http://localhost:8000"

start_bg \
  "Mailpit" \
  "mailpit --smtp 127.0.0.1:1025 --listen 127.0.0.1:8025" \
  "$LOG_DIR/mailpit.log" \
  "📧 Web UI: http://localhost:8025"

start_bg \
  "NATS Server" \
  "nats-server -c \"$ROOT_DIR/config/nats/nats-server.conf\"" \
  "$LOG_DIR/nats-server.log"

start_bg \
  "Email Worker" \
  "cd \"$ROOT_DIR/backend\" && go run -a cmd/email-worker/main.go --env \"$ENV_FILE\"" \
  "$LOG_DIR/email-worker.log" \
  "📊 Health: http://localhost:8081/health"

start_bg \
  "Notification Worker" \
  "cd \"$ROOT_DIR/backend\" && go run -a cmd/notification-worker/main.go --env \"$ENV_FILE\" --health-port 8083" \
  "$LOG_DIR/notification-worker.log" \
  "📊 Health: http://localhost:8083/health"

start_bg \
  "Internal Registry GC Worker" \
  "cd \"$ROOT_DIR/backend\" && go run -a cmd/internal-registry-gc-worker/main.go --env \"$ENV_FILE\"" \
  "$LOG_DIR/internal-registry-gc-worker.log" \
  "📊 Health: http://localhost:8085/health"

start_bg \
  "Redis" \
  "redis-server /usr/local/etc/redis.conf" \
  "$LOG_DIR/redis.log"

start_bg \
  "External Tenant Service" \
  "cd \"$ROOT_DIR/backend\" && go run cmd/external-tenant-service/main.go" \
  "$LOG_DIR/external-tenant-service.log" \
  "🏢 API: http://localhost:8082/api/tenants"

start_bg \
  "GLAuth LDAP" \
  "$HOME/glauth -c \"$ROOT_DIR/ldap-config.cfg\"" \
  "$LOG_DIR/glauth.log"

start_bg \
  "Backend Server" \
  "cd \"$ROOT_DIR/backend\" && go run cmd/server/main.go --env \"$ENV_FILE\"" \
  "$LOG_DIR/backend.log" \
  "📊 API: http://localhost:8080"

start_bg \
  "Dispatcher" \
  "cd \"$ROOT_DIR/backend\" && go run -a cmd/dispatcher/main.go --env \"$ENV_FILE\"" \
  "$LOG_DIR/dispatcher.log"

start_bg \
  "Frontend Dev Server" \
  "cd \"$ROOT_DIR/frontend\" && npm run dev" \
  "$LOG_DIR/frontend.log" \
  "🌐 UI: http://localhost:3000"

echo "🎉 All development services started."
echo "📝 Logs are under $LOG_DIR"
