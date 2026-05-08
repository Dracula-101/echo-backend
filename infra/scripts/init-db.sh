#!/bin/bash
# =============================================================================
# Echo Backend — database initializer.
#
# Used in TWO contexts:
#   1) Local dev: invoked by a developer or a Makefile target. Reads
#      $PROJECT_ROOT/.env for credentials.
#   2) Production: called by the `db-init` compose service. Credentials
#      come from container env (POSTGRES_HOST=postgres etc.). The repo is
#      not laid out the same way, so DATABASE_DIR is supplied by the
#      compose mount.
#
# Idempotent — exits 0 if the schema is already loaded.
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." 2>/dev/null && pwd || echo /)"
# DATABASE_DIR is overrideable so the same script runs unmodified inside
# the prod db-init container.
DATABASE_DIR="${DATABASE_DIR:-$PROJECT_ROOT/database}"

FORCE_RECREATE=false
SKIP_INDEXES=false
SKIP_CRON=false

while [[ "$#" -gt 0 ]]; do
    case $1 in
        --force)        FORCE_RECREATE=true ;;
        --skip-indexes) SKIP_INDEXES=true ;;
        --skip-cron)    SKIP_CRON=true ;;
        *) echo "[ERROR] Unknown parameter: $1"; exit 1 ;;
    esac
    shift
done

# Source PROJECT_ROOT/.env only if present AND we don't already have
# credentials from the container env.
if [ -f "$PROJECT_ROOT/.env" ] && [ -z "${POSTGRES_HOST:-}" ]; then
    set -a
    source <(grep -v '^#' "$PROJECT_ROOT/.env" | grep '=')
    set +a
fi

POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_USER="${POSTGRES_USER:-echo}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-echo_password}"
POSTGRES_DB="${POSTGRES_DB:-echo_db}"

export PGPASSWORD="$POSTGRES_PASSWORD"

# ── Temp file that accumulates all SQL to run in one transaction ──────────────
TRANSACTION_FILE=$(mktemp /tmp/echo_init_XXXXXX.sql)
TRANSACTION_FAILED=false

cleanup() {
    rm -f "$TRANSACTION_FILE"
}
trap cleanup EXIT

psql_root() {
    psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d postgres "$@"
}

psql_db() {
    psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" "$@"
}

# Logs the file and appends a \i directive to the transaction script.
# Nothing is executed yet — commit_transaction does that.
queue_file() {
    local file="$1"
    local label="${2:-$(basename "$file")}"
    if [ ! -f "$file" ]; then
        echo "   [WARN] Not found: $file (skipped)"
        return 0
    fi
    echo "   + $label"
    printf '\\i %s\n' "$file" >> "$TRANSACTION_FILE"
}

# Wraps the accumulated SQL in BEGIN/COMMIT and runs it as a single session.
# Any error triggers an automatic ROLLBACK (ON_ERROR_STOP + open transaction).
commit_transaction() {
    echo ""
    echo "[ T ] Executing transaction..."

    # Wrap the queued \i directives in an explicit transaction
    local wrapped
    wrapped=$(mktemp /tmp/echo_init_wrapped_XXXXXX.sql)
    {
        echo "BEGIN;"
        cat "$TRANSACTION_FILE"
        echo "COMMIT;"
    } > "$wrapped"

    local output
    if ! output=$(psql_db -v ON_ERROR_STOP=1 -f "$wrapped" --quiet 2>&1); then
        rm -f "$wrapped"
        # Strip noise, surface only real errors
        echo "$output" | grep -v "^NOTICE:" | grep -v "^$" | grep -v "^DO$" | grep -v "^BEGIN$" | grep -v "^ROLLBACK$"
        echo ""
        echo "   [ERROR] Transaction failed — all changes have been rolled back."
        exit 1
    fi

    rm -f "$wrapped"
    echo "   Transaction committed."
}

# ── Header ────────────────────────────────────────────────────────────────────
echo ""
echo "======================================================"
echo " Echo Backend — Database Initialization"
echo "======================================================"
echo " Host:     $POSTGRES_HOST:$POSTGRES_PORT"
echo " User:     $POSTGRES_USER"
echo " Database: $POSTGRES_DB"
echo "======================================================"

# ── [1/8] Connection check ────────────────────────────────────────────────────
echo ""
echo "[1/8] Verifying PostgreSQL connection..."
if ! psql_root -c '\q' > /dev/null 2>&1; then
    echo "[ERROR] Cannot connect to PostgreSQL at $POSTGRES_HOST:$POSTGRES_PORT"
    echo "        Ensure Postgres is running and credentials are correct."
    exit 1
fi
echo "      Connected."

# ── [2/8] Database creation ───────────────────────────────────────────────────
# CREATE DATABASE cannot run inside a transaction — kept outside intentionally.
echo ""
echo "[2/8] Preparing database..."

DB_EXISTS=$(psql_root -tAc \
    "SELECT 1 FROM pg_database WHERE datname='$POSTGRES_DB'" 2>/dev/null || echo "")

if [ "$DB_EXISTS" = "1" ] && [ "$FORCE_RECREATE" = true ]; then
    echo "      Dropping: $POSTGRES_DB"
    psql_root -c \
        "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname='$POSTGRES_DB';" \
        > /dev/null
    psql_root -c "DROP DATABASE $POSTGRES_DB;" > /dev/null
    DB_EXISTS=""
fi

if [ "$DB_EXISTS" != "1" ]; then
    echo "      Creating: $POSTGRES_DB"
    psql_root -c "CREATE DATABASE $POSTGRES_DB ENCODING 'UTF8';" > /dev/null
else
    TABLE_EXISTS=$(psql_db -tAc \
        "SELECT 1 FROM information_schema.tables
         WHERE table_schema='auth' AND table_name='users'" 2>/dev/null || echo "")
    if [ "$TABLE_EXISTS" = "1" ] && [ "$FORCE_RECREATE" = false ]; then
        echo "      Already initialized. Use --force to reinitialize."
        exit 0
    fi
fi

# ── Steps 3–8: queued into one transaction ────────────────────────────────────

echo ""
echo "[3/8] Enabling extensions..."
queue_file "$DATABASE_DIR/extensions.sql" "extensions.sql"

echo ""
echo "[4/8] Creating schemas..."
for f in \
    "auth.schema.sql" \
    "user.schema.sql" \
    "message.schema.sql" \
    "media.schema.sql" \
    "notification.schema.sql" \
    "analytics.schema.sql" \
    "location.schema.sql"
do
    queue_file "$DATABASE_DIR/schemas/$f" "$f"
done

echo ""
echo "[5/8] Creating functions..."
for f in \
    "auth.functions.sql" \
    "auth.trigger_functions.sql" \
    "users.functions.sql" \
    "users.trigger_functions.sql" \
    "media.functions.sql" \
    "media.trigger_functions.sql" \
    "messages.functions.sql" \
    "messages.trigger_functions.sql" \
    "notifications.functions.sql" \
    "notifications.trigger_functions.sql"
do
    queue_file "$DATABASE_DIR/functions/$f" "$f"
done

echo ""
echo "[6/8] Creating triggers..."
for dir in auth users media messages notifications; do
    dir_path="$DATABASE_DIR/triggers/$dir"
    if [ ! -d "$dir_path" ]; then
        echo "   [WARN] Not found: triggers/$dir"
        continue
    fi
    while IFS= read -r -d '' f; do
        queue_file "$f" "$dir/$(basename "$f")"
    done < <(find "$dir_path" -maxdepth 1 -name '*.sql' -print0 | sort -zV)
done

echo ""
echo "[7/8] Applying RLS policies..."
for f in \
    "auth.rls.sql" \
    "users.rls.sql" \
    "media.rls.sql" \
    "messages.rls.sql" \
    "notifications.rls.sql"
do
    queue_file "$DATABASE_DIR/rls/$f" "$f"
done

echo ""
if [ "$SKIP_INDEXES" = false ]; then
    echo "[8/8] Creating indexes..."
    for f in \
        "auth.indexes.sql" \
        "users.indexes.sql" \
        "media.indexes.sql" \
        "messages.indexes.sql" \
        "notifications.indexes.sql"
    do
        queue_file "$DATABASE_DIR/indexes/$f" "$f"
    done
else
    echo "[8/8] Skipping indexes (--skip-indexes)"
fi

# ── Flush the transaction ─────────────────────────────────────────────────────
commit_transaction

# ── pg_cron (outside transaction — it writes to a shared catalog) ─────────────
if [ "$SKIP_CRON" = false ]; then
    echo ""
    echo "[ + ] Scheduling pg_cron jobs..."
    if [ -f "$DATABASE_DIR/pg_cron.sql" ]; then

        # CREATE EXTENSION is what creates the cron schema — the scheduler
        # background worker only runs jobs, it does not create the schema.
        echo "      Creating pg_cron extension..."
        if ! ext_output=$(psql_db -v ON_ERROR_STOP=1 --quiet \
                -c "CREATE EXTENSION IF NOT EXISTS pg_cron;" 2>&1); then
            echo "$ext_output" | grep -v "^NOTICE:" | grep -v "^$"
            echo ""
            echo "   [ERROR] Could not create pg_cron extension."
            echo "           Ensure shared_preload_libraries includes 'pg_cron' and Postgres is restarted."
            echo "           Run with --skip-cron to bypass this step."
            exit 1
        fi

        if ! cron_output=$(psql_db -v ON_ERROR_STOP=1 -f "$DATABASE_DIR/pg_cron.sql" --quiet 2>&1); then
            echo "$cron_output" | grep -v "^NOTICE:" | grep -v "^$"
            echo ""
            echo "   [ERROR] pg_cron scheduling failed."
            echo "           Ensure shared_preload_libraries includes 'pg_cron' and Postgres is restarted."
            echo "           Run with --skip-cron to bypass this step."
            exit 1
        fi
        echo "      pg_cron jobs scheduled."
    fi
fi

# ── Summary ───────────────────────────────────────────────────────────────────
echo ""
echo "======================================================"
echo " Summary"
echo "======================================================"

psql_db --quiet -c "
SELECT
    schemaname     AS \"Schema\",
    COUNT(*)::TEXT AS \"Tables\"
FROM pg_tables
WHERE schemaname IN
    ('auth','users','media','messages','notifications','analytics','location')
GROUP BY schemaname
ORDER BY schemaname;" 2>/dev/null || true

TRIGGER_COUNT=$(psql_db -tAc "
    SELECT COUNT(*) FROM information_schema.triggers
    WHERE trigger_schema IN
        ('auth','users','media','messages','notifications')" 2>/dev/null || echo "?")

INDEX_COUNT=$(psql_db -tAc "
    SELECT COUNT(*) FROM pg_indexes
    WHERE schemaname IN
        ('auth','users','media','messages','notifications','analytics','location')" 2>/dev/null || echo "?")

echo ""
echo " Triggers : $TRIGGER_COUNT"
echo " Indexes  : $INDEX_COUNT"
echo "======================================================"
echo " Done. '$POSTGRES_DB' is ready."
echo "======================================================"
echo ""
echo " Usage:"
echo "   ./init-db.sh                # initialize (skip if already done)"
echo "   ./init-db.sh --force        # drop and recreate"
echo "   ./init-db.sh --skip-indexes # skip index creation (faster for dev)"
echo "   ./init-db.sh --skip-cron    # skip pg_cron job scheduling"
echo ""