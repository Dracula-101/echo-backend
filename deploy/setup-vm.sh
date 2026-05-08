#!/usr/bin/env bash
# =============================================================================
# One-time bootstrap for a fresh Linux VM (Ubuntu 22.04+ or Debian 12+).
# Works on any cloud — GCP, Hetzner, DigitalOcean, AWS, bare metal.
#
# Run as the deploy user (NOT root):
#
#   ssh deploy@<VM_IP> 'bash -s' < deploy/setup-vm.sh
#
# Or with the repo URL pre-set:
#
#   ssh deploy@<VM_IP> "REPO_URL=https://github.com/you/echo-backend.git bash -s" \
#     < deploy/setup-vm.sh
#
# Idempotent — re-running is safe.
# =============================================================================
set -euo pipefail

REPO_URL="${REPO_URL:-https://github.com/YOUR_USER/echo-backend.git}"
APP_DIR="${APP_DIR:-/opt/echo-backend}"
SWAP_SIZE="${SWAP_SIZE:-2G}"
INFRA_IMAGES=(
  postgres:15-alpine
  redis:7-alpine
  caddy:2-alpine
  confluentinc/cp-kafka:7.5.0
  confluentinc/cp-zookeeper:7.5.0
)

step() { echo; echo "==> $*"; }

# ---------------------------------------------------------------------------
step "[1/8] System packages"
# ---------------------------------------------------------------------------
if ! command -v docker >/dev/null 2>&1; then
  sudo apt-get update -y
  # `make` ships separately from build-essential on cloud-minimal images;
  # add it explicitly so the operator-facing make targets work.
  sudo apt-get install -y ca-certificates curl gnupg git ufw jq make \
                          unattended-upgrades apt-listchanges

  sudo install -m 0755 -d /etc/apt/keyrings
  DISTRO_ID=$(. /etc/os-release && echo "$ID")
  case "$DISTRO_ID" in
    ubuntu|debian) : ;;
    *) echo "Unsupported distro: $DISTRO_ID"; exit 1 ;;
  esac

  curl -fsSL "https://download.docker.com/linux/$DISTRO_ID/gpg" \
    | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
  sudo chmod a+r /etc/apt/keyrings/docker.gpg

  ARCH=$(dpkg --print-architecture)
  CODENAME=$(. /etc/os-release && echo "$VERSION_CODENAME")
  echo "deb [arch=$ARCH signed-by=/etc/apt/keyrings/docker.gpg] \
        https://download.docker.com/linux/$DISTRO_ID $CODENAME stable" \
    | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

  sudo apt-get update -y
  sudo apt-get install -y docker-ce docker-ce-cli containerd.io \
                          docker-buildx-plugin docker-compose-plugin

  sudo usermod -aG docker "$USER"
  echo "    Added '$USER' to docker group; re-login for docker without sudo."
fi

# ---------------------------------------------------------------------------
step "[2/8] Firewall (SSH/HTTP/HTTPS only)"
# ---------------------------------------------------------------------------
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow 443/udp     # HTTP/3
sudo ufw --force enable || true

# ---------------------------------------------------------------------------
step "[3/8] Auto security updates"
# ---------------------------------------------------------------------------
echo 'APT::Periodic::Update-Package-Lists "1";
APT::Periodic::Unattended-Upgrade "1";
APT::Periodic::AutocleanInterval "7";' \
  | sudo tee /etc/apt/apt.conf.d/20auto-upgrades > /dev/null
sudo systemctl enable --now unattended-upgrades || true

# ---------------------------------------------------------------------------
step "[4/8] Repo at $APP_DIR"
# ---------------------------------------------------------------------------
if [ ! -d "$APP_DIR/.git" ]; then
  sudo mkdir -p "$APP_DIR"
  sudo chown -R "$USER:$USER" "$APP_DIR"
  git clone "$REPO_URL" "$APP_DIR"
else
  # Force the working tree to match origin/main. Bootstrap is a setup
  # step, not a deploy step — vm-deploy.sh handles per-SHA checkouts.
  git -C "$APP_DIR" fetch --all --prune
  git -C "$APP_DIR" checkout --quiet main
  git -C "$APP_DIR" reset --hard --quiet origin/main
fi

# ---------------------------------------------------------------------------
step "[5/8] .env.prod"
# ---------------------------------------------------------------------------
if [ ! -f "$APP_DIR/.env.prod" ]; then
  cp "$APP_DIR/.env.prod.example" "$APP_DIR/.env.prod"
  chmod 600 "$APP_DIR/.env.prod"
  echo "    Created $APP_DIR/.env.prod from the example."
  echo "    EDIT IT BEFORE RUNNING THE FIRST DEPLOY."
fi

# ---------------------------------------------------------------------------
step "[6/8] ${SWAP_SIZE} swap (insurance for memory pressure)"
# ---------------------------------------------------------------------------
if ! sudo swapon --show | grep -q '/swapfile'; then
  sudo fallocate -l "$SWAP_SIZE" /swapfile
  sudo chmod 600 /swapfile
  sudo mkswap /swapfile >/dev/null
  sudo swapon /swapfile
  echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab > /dev/null
  # Conservative — only swap when memory is genuinely exhausted.
  echo 'vm.swappiness=10' | sudo tee /etc/sysctl.d/99-swappiness.conf > /dev/null
  echo 'vm.overcommit_memory=1' | sudo tee -a /etc/sysctl.d/99-swappiness.conf > /dev/null
  sudo sysctl -p /etc/sysctl.d/99-swappiness.conf > /dev/null
fi

# ---------------------------------------------------------------------------
step "[7/8] Docker daemon (capped logs, live-restore, BuildKit, IPv6 off)"
# ---------------------------------------------------------------------------
sudo mkdir -p /etc/docker
# `registry-mirrors` routes Docker Hub pulls through Google's public
# mirror — eliminates Docker Hub anonymous rate limits (100/6h per IP)
# and is typically faster from a GCP VM. Affects only docker.io pulls;
# ghcr.io (your service images) is untouched.
sudo tee /etc/docker/daemon.json > /dev/null <<'JSON'
{
  "log-driver": "json-file",
  "log-opts": { "max-size": "10m", "max-file": "3" },
  "live-restore": true,
  "features": { "buildkit": true },
  "default-ulimits": { "nofile": { "Name": "nofile", "Soft": 65536, "Hard": 65536 } },
  "userland-proxy": false,
  "registry-mirrors": ["https://mirror.gcr.io"]
}
JSON
sudo systemctl restart docker || true
sudo systemctl enable docker

echo "    Pre-pulling infrastructure images (parallel — first deploy is faster)…"
for img in "${INFRA_IMAGES[@]}"; do
  ( docker pull --quiet "$img" >/dev/null 2>&1 && echo "      ✓ $img" ) &
done
wait

# ---------------------------------------------------------------------------
step "[8/9] Secrets directory"
# ---------------------------------------------------------------------------
# Caddy mounts $APP_DIR/secrets at /etc/caddy/certs. Create it now so the
# bind mount won't fail on first up, even before the user SCPs cert files.
mkdir -p "$APP_DIR/secrets"
chmod 700 "$APP_DIR/secrets"

# ---------------------------------------------------------------------------
step "[9/9] Backup cron"
# ---------------------------------------------------------------------------
# Runs at 03:17 UTC nightly. Adjust the minute to a random value — every
# tutorial uses 0/15/30/45 and the result is bursty load on cloud-storage.
CRON_LINE="17 3 * * * $APP_DIR/infra/scripts/backup-postgres.sh >> /var/log/echo-backup.log 2>&1"
( crontab -l 2>/dev/null | grep -vF "$APP_DIR/infra/scripts/backup-postgres.sh"; \
  echo "$CRON_LINE" ) | crontab -
sudo touch /var/log/echo-backup.log
sudo chown "$USER:$USER" /var/log/echo-backup.log
sudo chmod 0640 /var/log/echo-backup.log

cat <<EOF

============================================================
==> Bootstrap complete.
============================================================

What's done on the VM:
  - Docker + compose installed; deploy user added to docker group
  - ufw open for SSH + HTTP + HTTPS (incl. HTTP/3 over UDP)
  - Repo cloned to $APP_DIR
  - .env.prod created from .env.prod.example (still has placeholders)
  - 2 GB swap, conservative swappiness, raised file descriptors
  - Docker daemon: capped logs, live-restore, BuildKit
  - Infra images pre-pulled (postgres, redis, caddy, kafka, zookeeper)
  - $APP_DIR/secrets/ created (mode 700) for TLS certs
  - Nightly Postgres backup cron at 03:17 UTC

What you still need to do (from your laptop):
  1. SCP the real .env.prod over the placeholder:
       scp -i ~/.ssh/echo_deploy .env.prod \\
         deploy@<VM_IP>:$APP_DIR/.env.prod

  2. SCP your TLS certs (Cloudflare Origin Cert or Let's Encrypt):
       scp -i ~/.ssh/echo_deploy cert.pem key.pem \\
         deploy@<VM_IP>:$APP_DIR/secrets/

  3. (Once) Add 3 GH repo secrets so CI can SSH in:
       VM_SSH_HOST, VM_SSH_USER=deploy, VM_SSH_KEY=<contents of private key>

  4. Push to main — the workflow builds images and runs vm-deploy.sh for you.

Or, to deploy manually from inside this VM:
  cd $APP_DIR && make vm-up

Useful commands once running:
  make vm-status                # what's healthy
  make vm-logs SVC=ws-service   # tail logs
  make vm-restart SVC=foo       # restart one service
  make vm-psql                  # open a psql shell
  make vm-backup                # ad-hoc backup right now
  make vm-init-db               # re-run idempotent schema loader
EOF
