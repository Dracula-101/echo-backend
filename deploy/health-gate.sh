#!/usr/bin/env bash
# =============================================================================
# Polls `docker compose ps` until every service reports `healthy` (or
# `running` for ones with no healthcheck). Exits non-zero on timeout so the
# CI deploy step fails loudly instead of declaring success on a half-up
# stack.
#
# Usage:
#   deploy/health-gate.sh                       # base profile only
#   deploy/health-gate.sh "caddy full"          # match the deploy invocation
# =============================================================================
set -euo pipefail

APP_DIR="${APP_DIR:-/opt/echo-backend}"
TIMEOUT_SECONDS="${TIMEOUT_SECONDS:-180}"
POLL_INTERVAL_SECONDS=5

cd "$APP_DIR"

PROFILES="${1:-}"
PROFILE_FLAGS=()
for p in $PROFILES; do PROFILE_FLAGS+=(--profile "$p"); done

COMPOSE=(docker compose
  -f infra/docker/docker-compose.prod.yml
  --env-file .env.prod
  "${PROFILE_FLAGS[@]}")

deadline=$(( $(date +%s) + TIMEOUT_SECONDS ))

while :; do
  # `compose ps --format json` emits one JSON object per line.
  # We treat anything that isn't `healthy` or `running` (no-healthcheck
  # services) as "not ready yet". `migrate` is excluded because it exits
  # cleanly after running and shows up as `exited`.
  unready=$("${COMPOSE[@]}" ps --format json \
    | jq -rs '
        [ .[] | select(.Service != "migrate") ]
        | map(select(
            (.Health // "") != "healthy"
            and ((.Health // "") != "" or .State != "running")
          ))
        | map("\(.Service) [\(.State)/\(.Health // "n/a")]")
        | .[]
      ' 2>/dev/null || echo "compose-ps-failed")

  if [[ -z "$unready" ]]; then
    echo "[health-gate] all services healthy"
    exit 0
  fi

  if (( $(date +%s) >= deadline )); then
    echo "[health-gate] TIMEOUT after ${TIMEOUT_SECONDS}s. Still unready:" >&2
    echo "$unready" >&2
    echo >&2
    echo "[health-gate] last logs from each unhealthy service:" >&2
    while IFS= read -r line; do
      svc=$(awk '{print $1}' <<<"$line")
      echo "----- $svc -----" >&2
      "${COMPOSE[@]}" logs --tail=40 "$svc" >&2 || true
    done <<<"$unready"
    exit 1
  fi

  echo "[health-gate] waiting on:"
  echo "$unready" | sed 's/^/  /'
  sleep "$POLL_INTERVAL_SECONDS"
done
