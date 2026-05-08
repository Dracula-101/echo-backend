#!/usr/bin/env bash
# One-time bootstrap for a fresh GCP E2 VM (Ubuntu 22.04 LTS or Debian 12).
# Run this AS THE DEPLOY USER (not root) over SSH:
#
#   gcloud compute ssh echo-backend-vm --zone=us-central1-a \
#     --command='bash -s' < deploy/setup-vm.sh
#
# Idempotent — safe to re-run.

set -euo pipefail

REPO_URL="${REPO_URL:-https://github.com/YOUR_USER/echo-backend.git}"
APP_DIR="${APP_DIR:-/opt/echo-backend}"

echo "[1/5] Installing Docker + git…"
if ! command -v docker >/dev/null 2>&1; then
  sudo apt-get update -y
  sudo apt-get install -y ca-certificates curl gnupg git ufw

  sudo install -m 0755 -d /etc/apt/keyrings

  # Detect distro family — Docker hosts separate repos for ubuntu and debian.
  DISTRO_ID=$(. /etc/os-release && echo "$ID")
  case "$DISTRO_ID" in
    ubuntu|debian) : ;;  # supported
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
  echo "    User '$USER' added to docker group; re-login may be needed."
fi

echo "[2/6] Configuring firewall (HTTP/HTTPS + SSH)…"
# Use raw port rules — the "OpenSSH" application profile only exists on
# Ubuntu, not Debian. Raw ports work on both.
# `--force` is ONLY valid on `ufw enable/disable/reset`, not on `allow`.
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw --force enable || true

echo "[3/6] Cloning repo to $APP_DIR (or pulling if it exists)…"
if [ ! -d "$APP_DIR/.git" ]; then
  sudo mkdir -p "$APP_DIR"
  sudo chown -R "$USER:$USER" "$APP_DIR"
  git clone "$REPO_URL" "$APP_DIR"
else
  git -C "$APP_DIR" fetch --all --prune
fi

echo "[4/6] Creating .env stub if missing…"
if [ ! -f "$APP_DIR/.env" ]; then
  cat > "$APP_DIR/.env" <<'EOF'
# Fill in production values. The compose files reference these.
POSTGRES_USER=echo
POSTGRES_PASSWORD=CHANGE_ME
POSTGRES_DB=echo_db
REDIS_PASSWORD=CHANGE_ME
JWT_SECRET_KEY=CHANGE_ME
KAFKA_BROKERS=kafka:29092
EOF
  echo "    .env created at $APP_DIR/.env — EDIT THIS FILE before deploying."
fi

echo "[5/6] Setting up 2 GB swap (insurance for small VMs)…"
if ! sudo swapon --show | grep -q '/swapfile'; then
  sudo fallocate -l 2G /swapfile
  sudo chmod 600 /swapfile
  sudo mkswap /swapfile >/dev/null
  sudo swapon /swapfile
  echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab > /dev/null
  # Be conservative — only swap when memory is genuinely exhausted.
  echo 'vm.swappiness=10' | sudo tee /etc/sysctl.d/99-swappiness.conf > /dev/null
  sudo sysctl -p /etc/sysctl.d/99-swappiness.conf > /dev/null
  echo "    2 GB swap enabled at /swapfile"
else
  echo "    swap already configured"
fi

echo "[6/6] Configuring Docker daemon for resource-constrained hosts…"
# Cap log size so a noisy service can't fill the boot disk on a small VM.
sudo mkdir -p /etc/docker
echo '{
  "log-driver": "json-file",
  "log-opts": { "max-size": "10m", "max-file": "3" }
}' | sudo tee /etc/docker/daemon.json > /dev/null
sudo systemctl restart docker || true

echo
echo "Bootstrap complete. Next steps:"
echo "  1. Edit $APP_DIR/.env with real production secrets."
echo "  2. Push to main — the deploy workflow takes it from here."
echo "     (Or run a first manual deploy: cd $APP_DIR && \\"
echo "      docker compose -f infra/docker/docker-compose.yml \\"
echo "        -f infra/docker/docker-compose.prod.yml --env-file .env up -d --build)"
