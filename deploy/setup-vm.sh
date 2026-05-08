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
  sudo apt-get install -y ca-certificates curl gnupg git ufw jq \
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
sudo tee /etc/docker/daemon.json > /dev/null <<'JSON'
{
  "log-driver": "json-file",
  "log-opts": { "max-size": "10m", "max-file": "3" },
  "live-restore": true,
  "features": { "buildkit": true },
  "default-ulimits": { "nofile": { "Name": "nofile", "Soft": 65536, "Hard": 65536 } },
  "userland-proxy": false
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
step "[8/8] Backup cron"
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

==> Bootstrap complete.

Next steps:
  1. Edit $APP_DIR/.env.prod with real production secrets.
       openssl rand -base64 48 | tr -d '\n='   # generate strong values
  2. (Optional, only for private GHCR images)
       echo <PAT> | docker login ghcr.io -u <github-user> --password-stdin
  3. Push to main, or trigger the "Build & Deploy" workflow manually.

Manual first deploy:
  cd $APP_DIR
  bash deploy/vm-deploy.sh \$(git rev-parse HEAD)
EOF
