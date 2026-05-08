#!/bin/bash
# =============================================================================
# Echo Backend — production migration runner.
#
# Applies database/migrations/*.sql in filename order, idempotently. All
# migration files MUST be safe to re-run.
#
# Used by the `migrate` service in docker-compose.prod.yml. Connection
# params come from PG* env vars set by the compose service block.
#
# Verbose by design — every step is logged so `docker logs migrate` is
# enough to diagnose any failure.
# =============================================================================
set -euo pipefail

MIGRATIONS_DIR="${MIGRATIONS_DIR:-/migrations}"

echo "[migrate] migrations dir: $MIGRATIONS_DIR"
echo "[migrate] target: ${PGUSER:-?}@${PGHOST:-?}:${PGPORT:-5432}/${PGDATABASE:-?}"

# 1. Connectivity sanity check. Fail FAST and CLEARLY if Postgres isn't
#    reachable or credentials are wrong, instead of a cryptic psql exit
#    code from the first migration file.
if ! psql -v ON_ERROR_STOP=1 -tAc 'SELECT 1' >/dev/null 2>&1; then
  echo "[migrate] FAIL: cannot connect to Postgres" >&2
  echo "[migrate]   PGHOST=${PGHOST:-} PGPORT=${PGPORT:-} PGUSER=${PGUSER:-} PGDATABASE=${PGDATABASE:-}" >&2
  exit 2
fi
echo "[migrate] connection ok"

# 2. Directory check.
if [ ! -d "$MIGRATIONS_DIR" ]; then
  echo "[migrate] WARN: $MIGRATIONS_DIR is not a directory — nothing to apply"
  exit 0
fi

# 3. List candidate files (sorted).  `nullglob` makes the glob expand to
#    nothing rather than the literal pattern when no files match.
shopt -s nullglob
files=("$MIGRATIONS_DIR"/*.sql)
shopt -u nullglob

if [ "${#files[@]}" -eq 0 ]; then
  echo "[migrate] no .sql files in $MIGRATIONS_DIR — nothing to do"
  exit 0
fi

echo "[migrate] found ${#files[@]} migration file(s)"

# 4. Apply each in order. ON_ERROR_STOP=1 means psql aborts at the first
#    SQL error and returns non-zero — we propagate it.
applied=0
for f in "${files[@]}"; do
  name=$(basename "$f")
  echo "[migrate]   → applying $name"
  if psql -v ON_ERROR_STOP=1 -f "$f"; then
    applied=$((applied + 1))
  else
    rc=$?
    echo "[migrate] FAIL: $name returned exit $rc" >&2
    exit 3
  fi
done

echo "[migrate] done — applied $applied/${#files[@]} migration(s)"
