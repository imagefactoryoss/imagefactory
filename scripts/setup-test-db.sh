#!/bin/bash

set -e

echo "🧪 SETTING UP TEST DATABASE - Terminating existing connections..."
psql -h localhost -U postgres --no-psqlrc -v ON_ERROR_STOP=1 << 'EOF_SQL'
SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = 'image_factory_test' AND pid <> pg_backend_pid();
EOF_SQL

sleep 2

echo "📉 Dropping test database if exists..."
psql -h localhost -U postgres --no-psqlrc -c "DROP DATABASE IF EXISTS image_factory_test;"

echo "🆕 Creating fresh test database..."
psql -h localhost -U postgres --no-psqlrc -c "CREATE DATABASE image_factory_test OWNER postgres;"

echo "📦 Running migrations on test database..."
cd backend
IF_AUTH_JWT_SECRET=test-jwt-secret \
IF_DATABASE_HOST=localhost \
IF_DATABASE_PORT=5432 \
IF_DATABASE_NAME=image_factory_test \
IF_DATABASE_USER=postgres \
IF_DATABASE_PASSWORD=postgres \
IF_DATABASE_SSL_MODE=disable \
go run cmd/server/main.go --migrate-only

echo "✅ Test database setup and migrations complete!"
