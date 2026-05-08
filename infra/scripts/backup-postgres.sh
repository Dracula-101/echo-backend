#!/usr/bin/env bash
# =============================================================================
# Postgres backup for the prod stack. Runs `pg_dump` from inside the
# running container (no pg_dump on the host needed) and writes a
# timestamped, gzipped dump to BACKUP_DIR.
#
# If BACKUP_REMOTE_URL is set in .env.prod (gs:// or s3://) the dump is
# also shipped off-box. Off-box backups are non-fatal — the local copy
# always exists, and a network blip shouldn't fail the cron.
#
# Wired up by setup-vm.sh as a cron entry. To run by hand:
#   bash /opt/echo-backend/infra/scripts/backup-postgres.sh
# =============================================================================
set -euo pipefail

APP_DIR="${APP_DIR:-/opt/echo-backend}"
cd "$APP_DIR"

# Source env so POSTGRES_USER/DB/BACKUP_* are available.
set -a; . ./.env.prod; set +a

BACKUP_DIR="${BACKUP_DIR:-/var/backups/echo}"
RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-14}"
REMOTE_URL="${BACKUP_REMOTE_URL:-}"

mkdir -p "$BACKUP_DIR"
ts=$(date -u +"%Y%m%dT%H%M%SZ")
out="$BACKUP_DIR/echo-${POSTGRES_DB}-${ts}.sql.gz"

echo "[backup] dumping → $out"
docker compose \
  -f infra/docker/docker-compose.prod.yml \
  --env-file .env.prod \
  exec -T postgres \
    pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
            --no-owner --clean --if-exists \
  | gzip -9 > "$out"

# Sanity check — a healthy dump is at least a few KB.
size=$(stat -c%s "$out" 2>/dev/null || stat -f%z "$out")
if (( size < 4096 )); then
  echo "[backup] FAIL — dump suspiciously small ($size bytes)" >&2
  rm -f "$out"
  exit 1
fi
echo "[backup] local ok — $(du -h "$out" | cut -f1)"

# Off-box upload, if configured. Failures here log but do not fail the
# cron — the local copy is still good.
if [[ -n "$REMOTE_URL" ]]; then
  case "$REMOTE_URL" in
    gs://*)
      if command -v gsutil >/dev/null; then
        echo "[backup] uploading → $REMOTE_URL"
        gsutil -q cp "$out" "$REMOTE_URL" || echo "[backup] WARN: gsutil upload failed" >&2
      else
        echo "[backup] WARN: gsutil not installed — skipping off-box upload" >&2
      fi
      ;;
    s3://*)
      if command -v aws >/dev/null; then
        echo "[backup] uploading → $REMOTE_URL"
        aws s3 cp --quiet "$out" "$REMOTE_URL" || echo "[backup] WARN: aws s3 upload failed" >&2
      else
        echo "[backup] WARN: aws CLI not installed — skipping off-box upload" >&2
      fi
      ;;
    *)
      echo "[backup] WARN: unknown BACKUP_REMOTE_URL scheme: $REMOTE_URL" >&2
      ;;
  esac
fi

echo "[backup] pruning local backups older than ${RETENTION_DAYS}d"
find "$BACKUP_DIR" -type f -name "echo-*.sql.gz" \
     -mtime +"$RETENTION_DAYS" -print -delete
