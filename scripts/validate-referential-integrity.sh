#!/bin/bash

# Script to validate database referential integrity after migration
# This script checks if all foreign key constraints, unique constraints, and indexes are properly set up

echo "==============================================="
echo "Database Referential Integrity Validation"
echo "==============================================="
echo ""

# Source environment variables
if [ -f "../.env.development" ]; then
    export $(grep -v '^#' ../.env.development | xargs)
    echo "✅ Loaded environment from .env.development"
elif [ -f "../.env.clear.com" ]; then
    source ../.env.clear.com
    echo "✅ Loaded environment from .env.clear.com"
else
    echo "❌ Environment file not found. Please ensure .env.development or .env.clear.com exists."
    exit 1
fi

# Database connection string
DB_URL="postgresql://${IF_DATABASE_USER}:${IF_DATABASE_PASSWORD}@${IF_DATABASE_HOST}:${IF_DATABASE_PORT}/${IF_DATABASE_NAME}"

echo "🔍 Checking database connection..."
if ! psql "$DB_URL" -c "SELECT 1;" > /dev/null 2>&1; then
    echo "❌ Cannot connect to database. Please check your database configuration."
    exit 1
fi
echo "✅ Database connection successful"
echo ""

echo "📊 Checking foreign key constraints..."
echo "==============================================="

# Check foreign key constraints
FK_QUERY="
SELECT 
    tc.table_name,
    tc.constraint_name,
    tc.constraint_type,
    kcu.column_name,
    ccu.table_name AS foreign_table_name,
    ccu.column_name AS foreign_column_name,
    rc.delete_rule,
    rc.update_rule
FROM information_schema.table_constraints AS tc
JOIN information_schema.key_column_usage AS kcu
    ON tc.constraint_name = kcu.constraint_name
    AND tc.table_schema = kcu.table_schema
JOIN information_schema.constraint_column_usage AS ccu
    ON ccu.constraint_name = tc.constraint_name
    AND ccu.table_schema = tc.table_schema
LEFT JOIN information_schema.referential_constraints AS rc
    ON tc.constraint_name = rc.constraint_name
    AND tc.table_schema = rc.constraint_schema
WHERE tc.constraint_type = 'FOREIGN KEY'
    AND tc.table_schema = 'public'
ORDER BY tc.table_name, tc.constraint_name;
"

psql "$DB_URL" -c "$FK_QUERY"

echo ""
echo "🔗 Checking unique constraints and indexes..."
echo "==============================================="

# Check unique constraints
UNIQUE_QUERY="
SELECT 
    tc.table_name,
    tc.constraint_name,
    string_agg(kcu.column_name, ', ' ORDER BY kcu.ordinal_position) AS columns
FROM information_schema.table_constraints AS tc
JOIN information_schema.key_column_usage AS kcu
    ON tc.constraint_name = kcu.constraint_name
    AND tc.table_schema = kcu.table_schema
WHERE tc.constraint_type = 'UNIQUE'
    AND tc.table_schema = 'public'
GROUP BY tc.table_name, tc.constraint_name
ORDER BY tc.table_name, tc.constraint_name;
"

psql "$DB_URL" -c "$UNIQUE_QUERY"

echo ""
echo "📋 Checking table constraints (CHECK constraints)..."
echo "==============================================="

# Check table constraints
CHECK_QUERY="
SELECT 
    tc.table_name,
    tc.constraint_name,
    cc.check_clause
FROM information_schema.table_constraints AS tc
JOIN information_schema.check_constraints AS cc
    ON tc.constraint_name = cc.constraint_name
    AND tc.constraint_schema = cc.constraint_schema
WHERE tc.constraint_type = 'CHECK'
    AND tc.table_schema = 'public'
ORDER BY tc.table_name, tc.constraint_name;
"

psql "$DB_URL" -c "$CHECK_QUERY"

echo ""
echo "🗂️ Checking indexes..."
echo "==============================================="

# Check indexes
INDEX_QUERY="
SELECT 
    schemaname,
    tablename,
    indexname,
    indexdef
FROM pg_indexes
WHERE schemaname = 'public'
    AND indexname NOT LIKE '%_pkey'  -- Exclude primary key indexes
ORDER BY tablename, indexname;
"

psql "$DB_URL" -c "$INDEX_QUERY"

echo ""
echo "✅ Validation complete!"
echo ""
echo "📝 Summary of key integrity features to verify:"
echo "   - Foreign key constraints with proper CASCADE rules"
echo "   - Unique constraints preventing data duplication"
echo "   - CHECK constraints ensuring data validity"
echo "   - Performance indexes on foreign key columns"
echo ""
echo "🔍 Please review the output above to ensure all expected constraints are present."