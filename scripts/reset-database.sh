#!/bin/bash

set -e

ENV_FILE="$(cd "$(dirname "$0")/.." && pwd)/.env.development"

read_env_value() {
    local key="$1"
    if [[ -f "$ENV_FILE" ]]; then
        local value
        value=$(grep -E "^${key}=" "$ENV_FILE" | tail -n 1 | cut -d'=' -f2- || true)
        value="${value%\"}"
        value="${value#\"}"
        echo "$value"
    fi
}

echo "🗑️  RESETTING DATABASE - Clean Workflow"
echo "Step 1: Drop database"
echo "Step 2: Recreate and apply schema migrations"
echo "Step 3: Seed essential data"
echo "Step 4: Seed essential config + system bootstrap"
echo "Step 5: Seed demo data"
echo "Step 6: Seed email templates"
echo "Step 7: Seed external services"
echo "Step 8: Validate seed integrity"
echo "Step 9: Validate demo integrity"
echo ""

DB_HOST="${IF_DATABASE_HOST:-$(read_env_value IF_DATABASE_HOST)}"
DB_USER="${IF_DATABASE_USER:-$(read_env_value IF_DATABASE_USER)}"
DB_NAME="${IF_DATABASE_NAME:-$(read_env_value IF_DATABASE_NAME)}"
DB_PORT="${IF_DATABASE_PORT:-$(read_env_value IF_DATABASE_PORT)}"
DB_HOST="${DB_HOST:-localhost}"
DB_USER="${DB_USER:-postgres}"
DB_NAME="${DB_NAME:-image_factory_dev}"
DB_PORT="${DB_PORT:-5432}"

clear_dirty_migration() {
    local has_table
    has_table=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -Atqc "SELECT to_regclass('public.schema_migrations') IS NOT NULL;" 2>/dev/null || echo "f")
    if [[ "$has_table" != "t" ]]; then
        return 1
    fi

    local dirty_version
    dirty_version=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -Atqc "SELECT version FROM schema_migrations WHERE dirty = true ORDER BY version DESC LIMIT 1;" 2>/dev/null || true)
    if [[ -z "$dirty_version" ]]; then
        return 1
    fi

    local forced_version
    forced_version=$((dirty_version - 1))
    if [[ $forced_version -lt 0 ]]; then
        forced_version=0
    fi

    echo "🩹 Dirty migration detected at version ${dirty_version}; forcing schema_migrations to version ${forced_version} (clean) on ${DB_NAME}..."
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1 -c "UPDATE schema_migrations SET version = ${forced_version}, dirty = false;" >/dev/null
    return 0
}

run_migrations_with_dirty_mitigation() {
    local output
    local status

    set +e
    output=$(go run cmd/server/main.go --env ../.env.development --migrate-only 2>&1)
    status=$?
    set -e

    echo "$output" | grep -E "Database migrations completed|No new migrations|FATAL|dirty|migration failed" || true

    if [[ $status -eq 0 ]]; then
        return 0
    fi

    if clear_dirty_migration; then
        echo "🔁 Retrying migrations after dirty-flag cleanup..."
        set +e
        output=$(go run cmd/server/main.go --env ../.env.development --migrate-only 2>&1)
        status=$?
        set -e

        echo "$output" | grep -E "Database migrations completed|No new migrations|FATAL|dirty|migration failed" || true
        if [[ $status -eq 0 ]]; then
            return 0
        fi
    fi

    echo "❌ Migration failed after mitigation attempt."
    return 1
}

# ============================================================================
# STEP 1: TERMINATE CONNECTIONS AND DROP DATABASE
# ============================================================================
echo "⏳ Step 1: Terminating database connections..."
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" << 'EOF' 2>/dev/null || true
-- Block new connections before terminating existing ones.
ALTER DATABASE image_factory_dev CONNECTION LIMIT 0;
REVOKE CONNECT ON DATABASE image_factory_dev FROM PUBLIC;

-- Force terminate all other sessions (repeat a few times to catch reconnects).
DO $$
DECLARE
    killed_count int;
BEGIN
    FOR i IN 1..5 LOOP
        SELECT COUNT(*) INTO killed_count
        FROM pg_stat_activity
        WHERE datname = 'image_factory_dev'
          AND pid <> pg_backend_pid();

        EXIT WHEN killed_count = 0;

        PERFORM pg_terminate_backend(pid)
        FROM pg_stat_activity
        WHERE datname = 'image_factory_dev'
          AND pid <> pg_backend_pid();

        PERFORM pg_sleep(1);
    END LOOP;
END $$;
EOF

sleep 1

echo "📉 Dropping database..."
# Try FORCE drop first (Postgres 13+), then fallback to normal drop.
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -c "DROP DATABASE IF EXISTS ${DB_NAME} WITH (FORCE);" 2>/dev/null || \
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -c "DROP DATABASE IF EXISTS ${DB_NAME};"

# ============================================================================
# STEP 2: RECREATE DATABASE AND RUN SCHEMA MIGRATIONS
# ============================================================================
echo "🆕 Creating fresh database..."
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -c "CREATE DATABASE ${DB_NAME} OWNER ${DB_USER};"

echo "📦 Running schema migrations..."
cd "$(dirname "$0")/../backend"
run_migrations_with_dirty_mitigation

# ============================================================================
# STEP 3: SEED ESSENTIAL DATA
# ============================================================================
echo "🌱 Seeding essential data..."
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -v ON_ERROR_STOP=1 "$DB_NAME" < "./bootstrap/seed-essential-data.sql"

# ============================================================================
# STEP 4: SEED ESSENTIAL CONFIG + SYSTEM BOOTSTRAP
# ============================================================================
echo "⚙️  Seeding essential config + system bootstrap..."
go run ./cmd/essential-config-seeder --env ../.env.development 2>&1 | tail -3

# ============================================================================
# STEP 5: SEED DEMO DATA
# ============================================================================
echo "🎭 Seeding demo data..."
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -v ON_ERROR_STOP=1 "$DB_NAME" < "./bootstrap/seed-demo-data.sql"

# ============================================================================
# STEP 6: SEED EMAIL TEMPLATES
# ============================================================================
echo "📧 Seeding email templates..."
go run ./cmd/email-template-seeder --env ../.env.development 2>&1 | tail -3

# ============================================================================
# STEP 7: SEED EXTERNAL SERVICES
# ============================================================================
echo "🔗 Seeding external services..."
go run ./cmd/external-service-seeder --env ../.env.development 2>&1 | tail -3

# ============================================================================
# STEP 8: VALIDATE SEED INTEGRITY
# ============================================================================
echo "✅ Validating seed integrity..."
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -v ON_ERROR_STOP=1 "$DB_NAME" < "../scripts/validate-seed-integrity.sql" >/dev/null

# ============================================================================
# STEP 9: VALIDATE DEMO INTEGRITY
# ============================================================================
echo "✅ Validating demo integrity..."
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -v ON_ERROR_STOP=1 "$DB_NAME" < "../scripts/validate-demo-integrity.sql" >/dev/null

echo ""
echo "✅ Database reset complete!"
echo "   - Schema: Applied via migrations"
echo "   - Essential data: Seeded from backend/bootstrap/seed-essential-data.sql"
echo "   - Demo data: Seeded from backend/bootstrap/seed-demo-data.sql"
echo "   - Email templates: Seeded via email-template-seeder"
echo "   - External services: Seeded via external-service-seeder"
echo "   - Essential config + system bootstrap: Seeded via essential-config-seeder"
echo "   - Seed integrity: Validated via validate-seed-integrity.sql"
echo "   - Demo integrity: Validated via validate-demo-integrity.sql"
