#!/usr/bin/env bash
# =============================================================================
# Pre-deploy preflight check. Runs ON the VM before every deploy.
#
# Validates that everything the deploy depends on is in place — fails loudly
# with a SPECIFIC error instead of letting docker compose surface a generic
# "no such file" or healthcheck-timeout message.
#
# Exit codes:
#   0  all good
#   1  fatal: missing file / invalid config / docker not running
#
# Designed to be cheap (<1s) so we run it on every deploy.
# =============================================================================
set -euo pipefail

APP_DIR="${APP_DIR:-/opt/echo-backend}"
cd "$APP_DIR"

ok=true
fail() { echo "  [FAIL] $*"; ok=false; }
warn() { echo "  [WARN] $*"; }
pass() { echo "  [ OK ] $*"; }

echo "== preflight =="

# ---------------------------------------------------------------------------
# 1. Docker daemon
# ---------------------------------------------------------------------------
if docker info >/dev/null 2>&1; then
  pass "docker daemon reachable"
else
  fail "docker daemon not reachable (try: sudo systemctl status docker)"
fi

# ---------------------------------------------------------------------------
# 2. Disk space (we need at least 5 GB free for image pulls + Postgres data)
# ---------------------------------------------------------------------------
# Portable: df -k always returns 1K blocks. tail+awk picks the available
# column. Then divide by 1024^2 to get GB. Works on macOS + Linux.
free_kb=$(df -k "$APP_DIR" 2>/dev/null | awk 'NR==2 {print $4}')
if [ -n "${free_kb:-}" ]; then
  free_gb=$(( free_kb / 1024 / 1024 ))
  if [ "$free_gb" -ge 5 ]; then
    pass "${free_gb} GB free in $APP_DIR"
  else
    fail "only ${free_gb} GB free in $APP_DIR (need >=5 GB)"
  fi
else
  warn "could not determine free space in $APP_DIR"
fi

# ---------------------------------------------------------------------------
# 3. .env.prod present + mode 600 + no CHANGE_ME placeholders
# ---------------------------------------------------------------------------
if [ ! -f .env.prod ]; then
  fail ".env.prod missing"
else
  perms=$(stat -c '%a' .env.prod 2>/dev/null || stat -f '%Lp' .env.prod 2>/dev/null)
  if [ "$perms" = "600" ]; then
    pass ".env.prod present (mode 600)"
  else
    warn ".env.prod present but mode is $perms (recommend 600 — chmod 600 .env.prod)"
  fi

  if grep -q "CHANGE_ME" .env.prod; then
    fail ".env.prod still contains CHANGE_ME placeholders"
    grep -nH "CHANGE_ME" .env.prod | head -5 | sed 's/^/         /'
  else
    pass "no CHANGE_ME placeholders"
  fi

  # Required keys (without these the stack won't start)
  required=(IMAGE_REGISTRY DOMAIN POSTGRES_PASSWORD REDIS_PASSWORD JWT_SECRET_KEY MEILI_MASTER_KEY)
  missing=()
  for k in "${required[@]}"; do
    if ! grep -E "^${k}=.+" .env.prod >/dev/null; then
      missing+=("$k")
    fi
  done
  if [ ${#missing[@]} -eq 0 ]; then
    pass "all required env vars set"
  else
    fail "required env vars missing or empty: ${missing[*]}"
  fi
fi

# ---------------------------------------------------------------------------
# 4. TLS certs (only required when caddy profile is enabled)
# ---------------------------------------------------------------------------
PROFILES="${PROFILES:-caddy}"
if echo "$PROFILES" | grep -q caddy; then
  if [ -f secrets/cert.pem ] && [ -f secrets/key.pem ]; then
    if openssl x509 -in secrets/cert.pem -noout -checkend 86400 >/dev/null 2>&1; then
      pass "TLS cert valid (>24h to expiry)"
    else
      warn "TLS cert expires within 24h (or unreadable)"
    fi
    pass "TLS key present"
  else
    fail "secrets/cert.pem or secrets/key.pem missing (caddy will fail to start)"
    echo "         scp them from your laptop, or remove --profile caddy"
  fi
fi

# ---------------------------------------------------------------------------
# 5. Compose config parses with the actual env
# ---------------------------------------------------------------------------
if docker compose \
     -f infra/docker/docker-compose.prod.yml \
     --env-file .env.prod \
     --profile caddy --profile full \
     config --quiet >/dev/null 2>&1; then
  pass "compose config parses cleanly"
else
  fail "compose config parse error — run for details:"
  echo "    docker compose -f infra/docker/docker-compose.prod.yml --env-file .env.prod config"
fi

# ---------------------------------------------------------------------------
# 6. GHCR authentication (only if the registry is private — public images
#    work without docker login)
# ---------------------------------------------------------------------------
registry_host=$(grep -E "^IMAGE_REGISTRY=" .env.prod 2>/dev/null | cut -d= -f2 | cut -d/ -f1)
if [ -n "$registry_host" ]; then
  # Try a HEAD on the manifests endpoint with no auth. If 200 → public.
  # If 401/403 → needs login.
  manifest="https://${registry_host}/v2/"
  status=$(curl -sS -o /dev/null -w "%{http_code}" "$manifest" 2>/dev/null || echo "")
  case "$status" in
    200|401)  pass "registry $registry_host reachable (public images or already auth'd)" ;;
    403)
      if [ -f ~/.docker/config.json ] && grep -q "$registry_host" ~/.docker/config.json; then
        pass "registry auth present in ~/.docker/config.json"
      else
        warn "registry $registry_host returned 403 and no auth in ~/.docker/config.json"
        echo "         If your packages are private, run:"
        echo "         echo <PAT> | docker login $registry_host -u <user> --password-stdin"
      fi
      ;;
    *)        warn "registry $registry_host probe returned status=$status (may still work)" ;;
  esac
fi

echo
$ok || { echo "preflight FAILED — fix the [FAIL] items above"; exit 1; }
echo "preflight ok"
