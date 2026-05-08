#!/usr/bin/env bash
# =============================================================================
# Runs ON the deploy VM. Invoked by the GitHub Actions workflow over SSH,
# but can also be run by hand:
#
#   bash /opt/echo-backend/deploy/vm-deploy.sh <git-sha>
#
# What it does:
#   1) git fetch + checkout the requested SHA (so compose files match images)
#   2) docker compose pull --parallel (images are pre-built by CI)
#   3) compose up -d --wait (blocks until every container is healthy)
#   4) on --wait failure, run health-gate.sh for richer diagnostics
#   5) prune dangling images
#
# Idempotent — safe to re-run with the same SHA.
# Exits non-zero on failure so CI surfaces the error.
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

echo "[deploy] checking out $SHA"
git fetch --all --prune --quiet
git checkout --quiet --force "$SHA"

# Image tag — must match what CI pushed. Truncate to keep it short but
# unique enough to never collide in practice.
export IMAGE_TAG="sha-${SHA:0:12}"

# Profiles: comma- or space-separated. `caddy` is on by default in prod
# (real public ingress). Add `full` to also bring up the heavier services.
PROFILES="${PROFILES:-caddy}"
PROFILE_FLAGS=()
for p in ${PROFILES//,/ }; do PROFILE_FLAGS+=(--profile "$p"); done

COMPOSE=(docker compose
  -f infra/docker/docker-compose.prod.yml
  --env-file .env.prod
  "${PROFILE_FLAGS[@]}"
)

start_ts=$(date +%s)

echo "[deploy] pulling images (tag=$IMAGE_TAG, profiles=$PROFILES)"
# Parallel pulls — compose v2 does this by default, but be explicit so a
# future flag flip doesn't slow us down.
"${COMPOSE[@]}" pull --policy always --quiet

echo "[deploy] applying stack with --wait (timeout=${WAIT_TIMEOUT}s)"
# `--wait` blocks until every service with a healthcheck reports healthy
# (and one-shots like `migrate` exit successfully). Faster and less noisy
# than polling. If it times out, we fall back to health-gate.sh for the
# richer per-service diagnostic dump.
if ! "${COMPOSE[@]}" up -d \
       --remove-orphans \
       --no-build \
       --pull never \
       --wait \
       --wait-timeout "$WAIT_TIMEOUT"; then
  echo "[deploy] --wait failed, dumping detailed diagnostics" >&2
  bash "$APP_DIR/deploy/health-gate.sh" "$PROFILES" || true
  exit 1
fi

end_ts=$(date +%s)
echo "[deploy] all services healthy in $((end_ts - start_ts))s"

echo "[deploy] pruning old images (>72h, dangling)"
docker image prune -af --filter "until=72h" >/dev/null
docker builder prune -f --keep-storage 2GB >/dev/null 2>&1 || true

echo "[deploy] currently running:"
"${COMPOSE[@]}" ps --format "table {{.Service}}\t{{.Status}}\t{{.Image}}"
