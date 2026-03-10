#!/bin/sh
set -eu

RUNTIME_CONFIG_PATH="/usr/share/nginx/html/config.js"
API_BASE_URL="${IF_FRONTEND_API_BASE_URL:-${API_BASE_URL:-}}"

cat > "${RUNTIME_CONFIG_PATH}" <<EOF
window.__APP_CONFIG__ = {
  API_BASE_URL: "${API_BASE_URL}"
};
EOF
