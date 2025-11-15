#!/usr/bin/env bash
set -euo pipefail

# infra/scripts/clean-db.sh (improved auto-detect)
# Usage: ./clean-db.sh [--yes] [--schemas=a,b] [--container name]
# If a container is found it will exec into it. Otherwise prints helpful hints.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# locate .env (only used for local fallback)
ENV_FILE=""
for candidate in "$SCRIPT_DIR/.env" "$SCRIPT_DIR/../.env" "$SCRIPT_DIR/../../.env" "$SCRIPT_DIR/../../../.env"; do
  [ -f "$candidate" ] && ENV_FILE="$candidate" && break
done

if [ -n "$ENV_FILE" ]; then
  set -o allexport
  eval $(grep -E '^[A-Za-z_][A-Za-z0-9_]*=' "$ENV_FILE")
  set +o allexport
fi

POSTGRES_HOST=${POSTGRES_HOST:-postgres}
POSTGRES_PORT=${POSTGRES_PORT:-5432}
POSTGRES_USER=${POSTGRES_USER:-echo}
POSTGRES_PASSWORD=${POSTGRES_PASSWORD:-change_me_in_production}
POSTGRES_DB=${POSTGRES_DB:-echo_db}

POSTGRES_CONTAINER=${POSTGRES_CONTAINER:-}
DO_RUN=false
SCHEMAS=""

while [[ $# -gt 0 ]]; do
  case $1 in
    --yes) DO_RUN=true; shift ;;
    --schemas) SCHEMAS="$2"; shift 2 ;;
    --schemas=*) SCHEMAS="${1#*=}"; shift ;;
    --container) POSTGRES_CONTAINER="$2"; shift 2 ;;
    --container=*) POSTGRES_CONTAINER="${1#*=}"; shift ;;
    --help|-h) echo "Usage: $0 [--yes] [--schemas=a,b] [--container name]"; exit 0 ;;
    *) echo "Unknown arg: $1"; exit 1 ;;
  esac
done

SCHEMA_FILTER_SQL=""
if [ -n "$SCHEMAS" ]; then
  IFS=',' read -r -a arr <<< "$SCHEMAS"
  quoted=$(printf ",%s" "${arr[@]}")
  quoted="${quoted:1}"
  quoted_list=$(echo "$quoted" | sed "s/,/','/g")
  SCHEMA_FILTER_SQL="AND table_schema IN ('$quoted_list')"
fi

SQL_GENERATOR=$(cat <<'SQL'
WITH targets AS (
  SELECT table_schema, table_name
  FROM information_schema.tables
  WHERE table_type = 'BASE TABLE'
    AND table_schema NOT IN ('pg_catalog','information_schema')
    AND table_name NOT IN ('goose_db_version','schema_migrations','migrations')
    /* SCHEMA_FILTER_PLACEHOLDER */
)
SELECT
  'TRUNCATE TABLE ' || string_agg(quote_ident(table_schema) || '.' || quote_ident(table_name), ', ') || ' RESTART IDENTITY CASCADE;'
FROM targets;
SQL
)

if [ -n "$SCHEMA_FILTER_SQL" ]; then
  SQL_GENERATOR="${SQL_GENERATOR/\/\* SCHEMA_FILTER_PLACEHOLDER \*\/ /$SCHEMA_FILTER_SQL }"
else
  SQL_GENERATOR="${SQL_GENERATOR/\/\* SCHEMA_FILTER_PLACEHOLDER \*\/ / }"
fi

echo "[INFO] Target DB: ${POSTGRES_USER}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}"
[ -n "$SCHEMAS" ] && echo "[INFO] Limiting to schemas: $SCHEMAS"

# Helper: check container exists in docker ps output
container_exists() {
  local name="$1"
  docker ps --format '{{.Names}}' | grep -xFq "$name" 2>/dev/null
}

# Auto-detect a container if not provided
DETECTED_CONTAINER="${POSTGRES_CONTAINER:-}"
if [ -z "$DETECTED_CONTAINER" ]; then
  # 1) exact name match on POSTGRES_HOST
  if container_exists "$POSTGRES_HOST"; then
    DETECTED_CONTAINER="$POSTGRES_HOST"
  else
    # 2) any container with image containing 'postgres' (most reliable)
    DETECTED_CONTAINER="$(docker ps --format '{{.Names}}\t{{.Image}}' \
      | awk -F'\t' 'tolower($2) ~ /postgres/ {print $1; exit}')"
    if [ -z "$DETECTED_CONTAINER" ]; then
      # 3) any container name containing 'postgres'
      DETECTED_CONTAINER="$(docker ps --format '{{.Names}}' | grep -i 'postgres' | head -n1 || true)"
    fi
  fi
fi

# Function to test if host resolves locally
host_resolves_locally() {
  getent ahosts "$POSTGRES_HOST" > /dev/null 2>&1 || ping -c1 -W1 "$POSTGRES_HOST" > /dev/null 2>&1 || return 1
  return 0
}

# If we have a container, run SQL generator inside container; else prepare local plan
TRUNCATE_SQL=""
if [ -n "$DETECTED_CONTAINER" ]; then
  echo "[INFO] Using container: $DETECTED_CONTAINER"
  # run generator inside container (container env should already have credentials)
  TRUNCATE_SQL=$(docker exec -i "$DETECTED_CONTAINER" \
    psql -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" -h "${POSTGRES_HOST}" -p "${POSTGRES_PORT}" -At -q -v ON_ERROR_STOP=1 -c "$SQL_GENERATOR" 2>/dev/null || true)
else
  echo "[WARN] No Postgres container auto-detected."
  if host_resolves_locally; then
    echo "[INFO] POSTGRES_HOST resolves locally; using local psql."
    export PGPASSWORD="${POSTGRES_PASSWORD}"
    PSQL_LOCAL="psql -h ${POSTGRES_HOST} -p ${POSTGRES_PORT} -U ${POSTGRES_USER} -d ${POSTGRES_DB} -At -q -v ON_ERROR_STOP=1"
    TRUNCATE_SQL=$($PSQL_LOCAL -c "$SQL_GENERATOR" || true)
  else
    echo ""
    echo "[ERROR] Host '${POSTGRES_HOST}' does not resolve from this machine, and no running Docker container was found."
    echo "  * If Postgres runs in Docker, pass --container <container-name> (example: infra/scripts/clean-db.sh --container myproject_postgres_1 --yes)"
    echo "  * Or run 'docker ps' / 'docker compose ps' to find the container name."
    echo ""
    exit 1
  fi
fi

if [ -z "$TRUNCATE_SQL" ]; then
  echo "[INFO] No user tables found to truncate (after filters). Nothing to do."
  exit 0
fi

echo ""
echo "===== GENERATED SQL ====="
echo "$TRUNCATE_SQL"
echo "========================="
echo ""

run_truncate_inner() {
  if [ -n "$DETECTED_CONTAINER" ]; then
    docker exec -i "$DETECTED_CONTAINER" \
      psql -U "${POSTGRES_USER}" -d "${POSTGRES_DB}" -h "${POSTGRES_HOST}" -p "${POSTGRES_PORT}" -v ON_ERROR_STOP=1 -c "$TRUNCATE_SQL"
  else
    $PSQL_LOCAL -c "$TRUNCATE_SQL"
  fi
}

if [ "$DO_RUN" = true ]; then
  run_truncate_inner
else
  echo -n "Are you sure? [y/N] "
  if [ -t 0 ]; then
    read -r answer
    if [[ "$answer" =~ ^[Yy]$ ]]; then
      run_truncate_inner
    else
      echo "[DRY RUN] Not executed. Re-run with --yes to force execution."
    fi
  else
    echo "[DRY RUN] Non-interactive shell. Re-run with --yes to perform truncation."
  fi
fi
