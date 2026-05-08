#!/usr/bin/env bash
# =============================================================================
# Laptop-side: bring a VM in sync with the local repo state, then trigger
# a deploy. Replaces the multi-step manual scp/ssh dance.
#
# Usage:
#   bash deploy/sync-vm.sh <VM_IP> [path/to/cert.pem path/to/key.pem]
#
# What it does:
#   1) test SSH (skips if already cached known_hosts)
#   2) bootstrap the VM (idempotent — installs docker etc. on first run)
#   3) scp .env.prod over (if it exists locally)
#   4) scp cert.pem / key.pem over (if paths provided)
#   5) run preflight ON the VM
#   6) trigger a deploy of the current local HEAD
#
# Step (6) requires that HEAD has been pushed to origin/main and CI built
# the images. If not, we instead point you at `gh workflow run` or skip.
# =============================================================================
set -euo pipefail

VM_IP="${1:-}"
CERT_PEM="${2:-}"
KEY_PEM="${3:-}"

SSH_KEY="${SSH_KEY:-$HOME/.ssh/echo_deploy}"
APP_DIR="${APP_DIR:-/opt/echo-backend}"
REPO_URL="${REPO_URL:-$(git config --get remote.origin.url 2>/dev/null || echo)}"

if [[ -z "$VM_IP" ]]; then
  echo "usage: $0 <VM_IP> [cert.pem key.pem]" >&2
  echo "  VM_IP        external IP of the deploy VM"
  echo "  cert.pem     optional path to TLS cert (Cloudflare Origin Cert)"
  echo "  key.pem      optional path to TLS private key"
  echo
  echo "env overrides:"
  echo "  SSH_KEY      private key (default: ~/.ssh/echo_deploy)"
  echo "  APP_DIR      VM repo dir (default: /opt/echo-backend)"
  echo "  REPO_URL     git origin URL (auto-detected from git config)"
  exit 2
fi

ssh_vm() {
  ssh -i "$SSH_KEY" -o StrictHostKeyChecking=accept-new \
      "deploy@$VM_IP" "$@"
}
scp_vm() {
  scp -i "$SSH_KEY" -o StrictHostKeyChecking=accept-new "$@"
}

step() { echo; echo "==> $*"; }

# ---------------------------------------------------------------------------
step "[1/6] SSH check"
# ---------------------------------------------------------------------------
if ssh_vm 'whoami' >/dev/null 2>&1; then
  echo "    SSH ok as deploy@$VM_IP"
else
  echo "    SSH failed. Common fixes:" >&2
  echo "      ssh-keygen -R $VM_IP                           # if host key changed" >&2
  echo "      gcloud compute instances add-metadata ...      # if deploy user missing" >&2
  echo "    See deploy/README.md section 3 for details." >&2
  exit 1
fi

# ---------------------------------------------------------------------------
step "[2/6] Bootstrap + sync repo on VM (idempotent)"
# ---------------------------------------------------------------------------
if ssh_vm "test -d $APP_DIR/.git && test -x /usr/bin/docker"; then
  echo "    VM already bootstrapped (docker installed, repo cloned)"
  echo "    Updating working tree to origin/main..."
  ssh_vm "cd $APP_DIR && git fetch --all --prune --quiet && git reset --hard --quiet origin/main"
else
  echo "    Running setup-vm.sh..."
  ssh_vm "REPO_URL=$REPO_URL bash -s" < deploy/setup-vm.sh
fi

# ---------------------------------------------------------------------------
step "[3/6] Sync .env.prod"
# ---------------------------------------------------------------------------
if [[ -f .env.prod ]]; then
  echo "    Copying .env.prod ($(wc -c < .env.prod) bytes)"
  scp_vm .env.prod "deploy@$VM_IP:$APP_DIR/.env.prod"
  ssh_vm "chmod 600 $APP_DIR/.env.prod"
else
  echo "    No local .env.prod — skipping (VM may already have one)"
fi

# ---------------------------------------------------------------------------
step "[4/6] Sync TLS certs"
# ---------------------------------------------------------------------------
if [[ -n "$CERT_PEM" && -n "$KEY_PEM" ]]; then
  if [[ -f "$CERT_PEM" && -f "$KEY_PEM" ]]; then
    echo "    Copying $CERT_PEM and $KEY_PEM"
    ssh_vm "mkdir -p $APP_DIR/secrets && chmod 700 $APP_DIR/secrets"
    scp_vm "$CERT_PEM" "deploy@$VM_IP:$APP_DIR/secrets/cert.pem"
    scp_vm "$KEY_PEM"  "deploy@$VM_IP:$APP_DIR/secrets/key.pem"
    ssh_vm "chmod 600 $APP_DIR/secrets/*.pem"
  else
    echo "    cert.pem or key.pem not found at provided paths" >&2
    exit 1
  fi
elif ssh_vm "test -f $APP_DIR/secrets/cert.pem"; then
  echo "    Certs already on VM — skipping (pass paths to overwrite)"
else
  echo "    No certs on VM and no paths provided." >&2
  echo "    Re-run with: bash $0 $VM_IP <cert.pem> <key.pem>" >&2
  exit 1
fi

# ---------------------------------------------------------------------------
step "[5/6] Preflight on VM"
# ---------------------------------------------------------------------------
ssh_vm "cd $APP_DIR && bash deploy/preflight.sh"

# ---------------------------------------------------------------------------
step "[6/6] Trigger deploy"
# ---------------------------------------------------------------------------
local_sha=$(git rev-parse HEAD)
remote_sha=$(git ls-remote origin main 2>/dev/null | awk '{print $1}')

if [[ "$local_sha" != "$remote_sha" ]]; then
  echo "    Local HEAD ($local_sha) differs from origin/main ($remote_sha)."
  echo "    Push first:  git push origin main"
  echo
  echo "    Or to deploy the LOCAL HEAD directly from the VM (no GHCR images):"
  echo "    ssh -i $SSH_KEY deploy@$VM_IP 'cd $APP_DIR && git pull && make vm-up'"
  exit 0
fi

if command -v gh >/dev/null 2>&1; then
  echo "    Triggering 'Build & Deploy' workflow at SHA $local_sha"
  gh workflow run deploy.yml --ref main || true
  echo "    Watch: gh run watch (or visit the Actions tab)"
else
  echo "    'gh' CLI not installed — open the Actions tab to watch:"
  echo "    https://github.com/$(git config --get remote.origin.url \
      | sed -E 's#.*github.com[:/]##; s#\.git$##')/actions"
fi

echo
echo "==> sync-vm.sh complete."
echo "    Once GHA goes green, verify:"
echo "      curl -i https://\$DOMAIN/caddy-health"
