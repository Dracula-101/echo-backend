#!/bin/bash

# Echo Backend - Database Initialization Script
set -e

# Get the absolute path to the project root (2 levels up from this script)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
DATABASE_DIR="$PROJECT_ROOT/database"

echo "[INFO] Initializing Echo Backend Database..."
echo "[INFO] Project root: $PROJECT_ROOT"

# Parse command line arguments
FORCE_RECREATE=false
SKIP_INDEXES=false

while [[ "$#" -gt 0 ]]; do
    case $1 in
        --force) FORCE_RECREATE=true ;;
        --skip-indexes) SKIP_INDEXES=true ;;
        *) echo "Unknown parameter: $1"; exit 1 ;;
    esac
    shift
done

# Load environment variables
if [ -f "$PROJECT_ROOT/.env" ]; then
    export $(cat "$PROJECT_ROOT/.env" | grep -v '#' | awk '/=/ {print $1}')
fi

# Set default values
POSTGRES_HOST=${POSTGRES_HOST:-localhost}
POSTGRES_PORT=${POSTGRES_PORT:-5432}
POSTGRES_USER=${POSTGRES_USER:-echo}
POSTGRES_PASSWORD=${POSTGRES_PASSWORD:-echo_password}
POSTGRES_DB=${POSTGRES_DB:-echo_db}

# Check if PostgreSQL is running (connect to postgres default database)
echo "[INFO] Checking PostgreSQL connection..."
if ! PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d postgres -c '\q' 2>/dev/null; then
    echo "[ERROR] Cannot connect to PostgreSQL. Please check your connection settings."
    exit 1
fi
echo "[SUCCESS] PostgreSQL connection verified"

# Check if database exists
DB_EXISTS=$(PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='$POSTGRES_DB'" 2>/dev/null || echo "0")

if [ "$DB_EXISTS" = "1" ] && [ "$FORCE_RECREATE" = true ]; then
    echo "[WARNING] Dropping existing database: $POSTGRES_DB"
    PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d postgres -c "DROP DATABASE IF EXISTS $POSTGRES_DB;"
    DB_EXISTS="0"
fi

# Create main database
if [ "$DB_EXISTS" = "0" ]; then
    echo "[INFO] Creating database: $POSTGRES_DB"
    PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d postgres -c "CREATE DATABASE $POSTGRES_DB;"
else
    echo "[INFO] Database already exists: $POSTGRES_DB"
fi

# Check if schemas are already initialized (check for auth.users table)
TABLE_EXISTS=$(PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d $POSTGRES_DB -tAc "
    SELECT 1 
    FROM information_schema.tables 
    WHERE table_schema = 'auth' AND table_name = 'users';" 2>/dev/null || echo "0")     
if [ "$TABLE_EXISTS" = "1" ] && [ "$FORCE_RECREATE" = false ]; then
    echo "[INFO] Database schemas already initialized. Use --force to reinitialize."
    exit 0
fi

# Enable extensions
echo "[INFO] Enabling PostgreSQL extensions..."
PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d $POSTGRES_DB <<EOF
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "btree_gin";
CREATE EXTENSION IF NOT EXISTS "btree_gist";
EOF

# Initialize schemas in the correct order
# auth-schema must be loaded first as it contains auth.users table that others reference
echo "[INFO] Initializing database schemas..."

SCHEMA_ORDER=(
    "auth-schema.sql"
    "user-schema.sql"
    "message-schema.sql"
    "media-schema.sql"
    "notification-schema.sql"
    "analytics-schema.sql"
    "location-ip-schema.sql"
)

for schema_file in "${SCHEMA_ORDER[@]}"; do
    schema_path="$DATABASE_DIR/schemas/$schema_file"
    if [ -f "$schema_path" ]; then
        echo "   ✓ Loading $schema_file"
        PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d $POSTGRES_DB -f "$schema_path" 2>&1 | grep -v "NOTICE" | grep -v "^ERROR:" || true
    else
        echo "[WARNING] Schema file not found: $schema_file"
    fi
done

echo "[INFO] Creating database functions..."
if [ -d "$DATABASE_DIR/functions" ]; then
    FUNCTION_COUNT=0
    for func in "$DATABASE_DIR/functions"/*.sql; do
        if [ -f "$func" ]; then
            echo "   ✓ Loading $(basename $func)"
            PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d $POSTGRES_DB -f "$func" 2>&1 | grep -v "NOTICE" || true
            FUNCTION_COUNT=$((FUNCTION_COUNT + 1))
        fi
    done
    echo "   [INFO] Loaded $FUNCTION_COUNT function files"
else
    echo "   [WARNING] Functions directory not found"
fi

echo "[INFO] Creating database triggers..."
if [ -d "$DATABASE_DIR/triggers" ]; then
    TRIGGER_COUNT=0
    for trigger in "$DATABASE_DIR/triggers"/*.sql; do
        if [ -f "$trigger" ]; then
            echo "   ✓ Loading $(basename $trigger)"
            PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d $POSTGRES_DB -f "$trigger" 2>&1 | grep -v "NOTICE" || true
            TRIGGER_COUNT=$((TRIGGER_COUNT + 1))
        fi
    done
    echo "   [INFO] Loaded $TRIGGER_COUNT trigger files"
else
    echo "   [WARNING] Triggers directory not found"
fi

echo "[INFO] Creating database views..."
if [ -d "$DATABASE_DIR/views" ]; then
    VIEW_COUNT=0
    for view in "$DATABASE_DIR/views"/*.sql; do
        if [ -f "$view" ]; then
            echo "   ✓ Loading $(basename $view)"
            PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d $POSTGRES_DB -f "$view" 2>&1 | grep -v "NOTICE" || true
            VIEW_COUNT=$((VIEW_COUNT + 1))
        fi
    done
    echo "   [INFO] Loaded $VIEW_COUNT view files"
else
    echo "   [WARNING] Views directory not found"
fi

if [ "$SKIP_INDEXES" = false ]; then
    echo "[INFO] Creating performance indexes..."
    if [ -d "$DATABASE_DIR/indexes" ]; then
        INDEX_COUNT=0
        for index in "$DATABASE_DIR/indexes"/*.sql; do
            if [ -f "$index" ]; then
                echo "   ✓ Loading $(basename $index)"
                PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d $POSTGRES_DB -f "$index" 2>&1 | grep -v "NOTICE" || true
                INDEX_COUNT=$((INDEX_COUNT + 1))
            fi
        done
        echo "   [INFO] Loaded $INDEX_COUNT index files"
    else
        echo "   [WARNING] Indexes directory not found"
    fi
else
    echo "[INFO] Skipping index creation (--skip-indexes flag set)"
fi

# Display database statistics
echo ""
echo "[INFO] Database Statistics:"
PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d $POSTGRES_DB <<EOF || true
SELECT 
    schemaname as "Schema",
    COUNT(*) as "Tables"
FROM pg_tables 
WHERE schemaname IN ('auth', 'users', 'messages', 'media', 'notifications', 'analytics', 'location')
GROUP BY schemaname
ORDER BY schemaname;
EOF

echo ""
echo "[SUCCESS] Database initialization completed successfully!"
echo ""
echo "Database: $POSTGRES_DB"
echo "Host: $POSTGRES_HOST:$POSTGRES_PORT"
echo ""
echo "Usage:"
echo "  ./init-db.sh              # Initialize database (skip if exists)"
echo "  ./init-db.sh --force      # Drop and recreate database"
echo "  ./init-db.sh --skip-indexes # Skip index creation (faster for dev)"
