#!/usr/bin/env bash
# =============================================================================
# Runs ON the deploy VM. Invoked by the GitHub Actions workflow over SSH,
# but can also be run by hand:
#
#   bash /opt/echo-backend/deploy/vm-deploy.sh <git-sha>
#
# Flow:
#   1) preflight check (env, certs, disk, docker, compose syntax)
#   2) capture the currently-deployed SHA (for rollback)
#   3) git fetch + checkout the requested SHA
#   4) docker compose pull (parallel)
#   5) docker compose up -d --wait (blocks until healthy)
#   6) on --wait failure: dump diagnostics, rollback to the previous SHA,
#      and exit non-zero so CI surfaces the failure
#   7) prune dangling images
#
# Idempotent — safe to re-run with the same SHA.
# =============================================================================
set -euo pipefail

APP_DIR="${APP_DIR:-/opt/echo-backend}"
SHA="${1:-}"
WAIT_TIMEOUT="${WAIT_TIMEOUT:-180}"

if [[ -z "$SHA" ]]; then
  echo "usage: $0 <git-sha>" >&2
  exit 2
fi

cd "$APP_DIR"

# ---------------------------------------------------------------------------
# Step 1: preflight
# ---------------------------------------------------------------------------
PROFILES="${PROFILES:-caddy}" bash "$APP_DIR/deploy/preflight.sh"

# ---------------------------------------------------------------------------
# Step 2: capture the currently-running SHA for potential rollback.
#
# We read it from the api-gateway container's image tag (which is the most
# representative of "what was deployed"). On first deploy there's nothing
# to roll back to — that's fine, we still attempt the deploy.
# ---------------------------------------------------------------------------
PREV_SHA=""
if prev_image=$(docker inspect --format='{{.Config.Image}}' \
        echo-backend-api-gateway-1 2>/dev/null); then
  # Image looks like .../api-gateway:sha-abcdef123456
  if [[ "$prev_image" == *":sha-"* ]]; then
    PREV_SHA="${prev_image##*:sha-}"
    echo "[deploy] previous deployed sha: $PREV_SHA"
  fi
fi

# ---------------------------------------------------------------------------
# Step 3: checkout the new SHA
# ---------------------------------------------------------------------------
echo "[deploy] checking out $SHA"
git fetch --all --prune --quiet
git checkout --quiet --force "$SHA"

export IMAGE_TAG="sha-${SHA:0:12}"

# Profiles: comma- or space-separated. `caddy` is on by default in prod.
PROFILE_FLAGS=()
for p in ${PROFILES//,/ }; do PROFILE_FLAGS+=(--profile "$p"); done

COMPOSE=(docker compose
  -f infra/docker/docker-compose.prod.yml
  --env-file .env.prod
  "${PROFILE_FLAGS[@]}"
)

start_ts=$(date +%s)

# ---------------------------------------------------------------------------
# Step 4: pull images (parallel, default behaviour in compose v2)
# ---------------------------------------------------------------------------
echo "[deploy] pulling images (tag=$IMAGE_TAG, profiles=$PROFILES)"
"${COMPOSE[@]}" pull --policy always --quiet

# ---------------------------------------------------------------------------
# Step 5: up -d --wait
# ---------------------------------------------------------------------------
echo "[deploy] applying stack with --wait (timeout=${WAIT_TIMEOUT}s)"
deploy_failed=false
if ! "${COMPOSE[@]}" up -d \
       --remove-orphans \
       --no-build \
       --pull never \
       --wait \
       --wait-timeout "$WAIT_TIMEOUT"; then
  deploy_failed=true
fi

# ---------------------------------------------------------------------------
# Step 6: failure path — diagnostics + auto-rollback
# ---------------------------------------------------------------------------
if [[ "$deploy_failed" == true ]]; then
  echo
  echo "[deploy] !! --wait failed at $(date -u +%FT%TZ) !!" >&2
  echo "[deploy] dumping diagnostics:" >&2
  bash "$APP_DIR/deploy/health-gate.sh" "$PROFILES" || true

  # One-shot services (db-init, migrate) don't appear in health-gate
  # because they exit. Dump their logs explicitly — they're the most
  # common cause of cascade failure in the dependency chain.
  for svc in db-init migrate; do
    echo
    echo "[deploy] === last 80 lines from $svc ===" >&2
    "${COMPOSE[@]}" logs --tail=80 --no-color "$svc" 2>&1 | sed 's/^/  /' >&2 || true
  done

  if [[ -n "$PREV_SHA" ]] && [[ "$PREV_SHA" != "${SHA:0:12}" ]]; then
    echo
    echo "[deploy] AUTO-ROLLBACK to previous sha $PREV_SHA"
    git checkout --quiet --force "$PREV_SHA" || git checkout --quiet --force "$PREV_SHA"~0 || true
    export IMAGE_TAG="sha-${PREV_SHA}"
    if "${COMPOSE[@]}" up -d --remove-orphans --no-build --pull never \
          --wait --wait-timeout 120; then
      echo "[deploy] rolled back successfully — stack is on $PREV_SHA"
    else
      echo "[deploy] ROLLBACK FAILED — manual intervention required" >&2
    fi
  else
    echo "[deploy] no previous SHA to roll back to (first deploy or same SHA)" >&2
  fi

  exit 1
fi

end_ts=$(date +%s)
echo "[deploy] all services healthy in $((end_ts - start_ts))s"

# ---------------------------------------------------------------------------
# Step 7: prune dangling images / build cache (keeps disk usage in check)
# ---------------------------------------------------------------------------
echo "[deploy] pruning old images (>72h, dangling)"
docker image prune -af --filter "until=72h" >/dev/null
docker builder prune -f --keep-storage 2GB >/dev/null 2>&1 || true

echo "[deploy] currently running:"
"${COMPOSE[@]}" ps --format "table {{.Service}}\t{{.Status}}\t{{.Image}}"
