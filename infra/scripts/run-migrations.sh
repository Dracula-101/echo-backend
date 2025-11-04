#!/bin/bash

# Echo Backend - Database Migrations Runner
# This script applies incremental migrations to the database
set -e

# Get the absolute path to the project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
MIGRATIONS_DIR="$PROJECT_ROOT/migrations/postgres"

echo "[INFO] Echo Backend - Database Migrations Runner"
echo "[INFO] Project root: $PROJECT_ROOT"
echo ""

# Parse command line arguments
COMMAND=${1:-up}  # up, down, force, version
VERSION=${2:-}

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

# Build database URL
DATABASE_URL="postgresql://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=disable"

# Check if PostgreSQL is running
echo "[INFO] Checking PostgreSQL connection..."
if ! PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d $POSTGRES_DB -c '\q' 2>/dev/null; then
    echo "[ERROR] Cannot connect to PostgreSQL. Please check your connection settings."
    exit 1
fi
echo "[SUCCESS] PostgreSQL connection verified"
echo ""

# Function to run migrations manually
run_migration() {
    local direction=$1
    local version=$2

    if [ "$direction" = "up" ]; then
        echo "[INFO] Applying migrations..."

        for migration_file in "$MIGRATIONS_DIR"/*_*.up.sql; do
            if [ -f "$migration_file" ]; then
                filename=$(basename "$migration_file")
                migration_version=$(echo "$filename" | sed 's/^0*\([0-9]*\)_.*/\1/')

                # Check if already applied
                EXISTS=$(PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d $POSTGRES_DB -tAc \
                    "SELECT 1 FROM schema_migrations WHERE version = $migration_version" 2>/dev/null || echo "0")

                if [ "$EXISTS" = "0" ]; then
                    echo "   ⬆ Applying migration $filename..."
                    PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d $POSTGRES_DB -f "$migration_file" 2>&1 | grep -v "NOTICE" || true
                    echo "   ✓ Migration $migration_version applied"
                else
                    echo "   ⏭  Migration $migration_version already applied (skipping)"
                fi
            fi
        done
    elif [ "$direction" = "down" ]; then
        echo "[INFO] Rolling back migrations..."

        # Get list of applied migrations in reverse order
        APPLIED_VERSIONS=$(PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d $POSTGRES_DB -tAc \
            "SELECT version FROM schema_migrations ORDER BY version DESC" 2>/dev/null || echo "")

        if [ -z "$APPLIED_VERSIONS" ]; then
            echo "[INFO] No migrations to rollback"
            return
        fi

        # Rollback each migration
        for version in $APPLIED_VERSIONS; do
            migration_file=$(find "$MIGRATIONS_DIR" -name "${version}_*.down.sql" -o -name "0${version}_*.down.sql" -o -name "00${version}_*.down.sql" | head -1)

            if [ -f "$migration_file" ]; then
                filename=$(basename "$migration_file")
                echo "   ⬇ Rolling back migration $filename..."
                PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d $POSTGRES_DB -f "$migration_file" 2>&1 | grep -v "NOTICE" || true
                echo "   ✓ Migration $version rolled back"

                # Only rollback one migration unless VERSION is specified
                if [ -z "$VERSION" ]; then
                    break
                fi
            else
                echo "   [WARNING] Rollback file for migration $version not found"
            fi
        done
    fi
}

# Function to show migration status
show_status() {
    echo "[INFO] Migration Status:"
    echo ""

    # Check if schema_migrations table exists
    TABLE_EXISTS=$(PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d $POSTGRES_DB -tAc \
        "SELECT 1 FROM information_schema.tables WHERE table_name = 'schema_migrations'" 2>/dev/null || echo "0")

    if [ "$TABLE_EXISTS" = "0" ]; then
        echo "   [INFO] No migrations applied yet"
        echo ""
        echo "   Available migrations:"
        for migration_file in "$MIGRATIONS_DIR"/*_*.up.sql; do
            if [ -f "$migration_file" ]; then
                filename=$(basename "$migration_file")
                echo "   ○ $filename (pending)"
            fi
        done
        return
    fi

    # Get applied migrations
    PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d $POSTGRES_DB <<EOF
SELECT
    version,
    description,
    applied_at
FROM schema_migrations
ORDER BY version;
EOF

    echo ""
    echo "   Available migrations:"
    for migration_file in "$MIGRATIONS_DIR"/*_*.up.sql; do
        if [ -f "$migration_file" ]; then
            filename=$(basename "$migration_file")
            migration_version=$(echo "$filename" | sed 's/^0*\([0-9]*\)_.*/\1/')

            EXISTS=$(PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d $POSTGRES_DB -tAc \
                "SELECT 1 FROM schema_migrations WHERE version = $migration_version" 2>/dev/null || echo "0")

            if [ "$EXISTS" = "1" ]; then
                echo "   ✓ $filename (applied)"
            else
                echo "   ○ $filename (pending)"
            fi
        fi
    done
}

# Execute command
case "$COMMAND" in
    up)
        run_migration "up"
        echo ""
        echo "[SUCCESS] Migrations applied successfully!"
        ;;
    down)
        if [ -z "$VERSION" ]; then
            echo "[WARNING] This will rollback the last migration"
            read -p "Continue? [y/N] " -n 1 -r
            echo
            if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                echo "[INFO] Cancelled"
                exit 0
            fi
        fi
        run_migration "down" "$VERSION"
        echo ""
        echo "[SUCCESS] Migration(s) rolled back successfully!"
        ;;
    status)
        show_status
        ;;
    force)
        echo "[WARNING] This will force re-apply migration version $VERSION"
        if [ -z "$VERSION" ]; then
            echo "[ERROR] VERSION must be specified for force command"
            exit 1
        fi
        read -p "Continue? [y/N] " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            echo "[INFO] Cancelled"
            exit 0
        fi

        # Delete migration record
        PGPASSWORD=$POSTGRES_PASSWORD psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d $POSTGRES_DB -c \
            "DELETE FROM schema_migrations WHERE version = $VERSION"

        # Re-apply
        run_migration "up"
        echo "[SUCCESS] Migration forced successfully!"
        ;;
    *)
        echo "Usage: $0 {up|down|status|force VERSION}"
        echo ""
        echo "Commands:"
        echo "  up              Apply all pending migrations"
        echo "  down            Rollback the last migration"
        echo "  down VERSION    Rollback all migrations down to VERSION"
        echo "  status          Show migration status"
        echo "  force VERSION   Force re-apply a specific migration"
        echo ""
        exit 1
        ;;
esac

echo ""
