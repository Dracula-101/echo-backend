# Production deploy guide

Single-VM deploy of the echo-backend stack. Postgres, Redis, Kafka, Meilisearch
all run in docker on the VM. Cloudflare R2 is the only external dependency
(media uploads). Caddy fronts everything with TLS via a Cloudflare Origin
Certificate, sat behind Cloudflare's orange-cloud proxy.

```
laptop                    GitHub Actions              VM (e.g. e2-small)
                                                       /opt/echo-backend
git push origin main
       │                        │ build matrix
       ├──────────────────────► │ → ghcr.io
       │                        │
       │                        │ ssh deploy@VM
       │                        ├─── vm-deploy.sh <sha>
       │                        │      │ preflight
       │                        │      │ pull
       │                        │      │ up --wait
       │                        │      │ rollback on failure
       │                        ▼      ▼
                                   echo-app.net
```

## TL;DR — first deploy from a fresh laptop

```bash
# 1. one-time: provision the VM, generate a deploy SSH key, push to GCP metadata
gcloud compute instances create echo-backend-vm \
  --zone=us-central1-a --machine-type=e2-small \
  --image-family=ubuntu-2204-lts --image-project=ubuntu-os-cloud \
  --boot-disk-size=50GB --tags=http-server,https-server

ssh-keygen -t ed25519 -f ~/.ssh/echo_deploy -C "github-actions"

gcloud compute instances add-metadata echo-backend-vm --zone=us-central1-a \
  --metadata="ssh-keys=deploy:$(cat ~/.ssh/echo_deploy.pub) github-actions"

VM_IP=$(gcloud compute instances describe echo-backend-vm \
        --zone=us-central1-a \
        --format='get(networkInterfaces[0].accessConfigs[0].natIP)')

# 2. one-time: GH repo secrets (visit https://github.com/<repo>/settings/secrets/actions)
#    VM_SSH_HOST = $VM_IP
#    VM_SSH_USER = deploy
#    VM_SSH_KEY  = (cat ~/.ssh/echo_deploy)

# 3. one-time: Cloudflare DNS A record echo-app.net → $VM_IP, proxied (orange)
#    SSL/TLS mode → Full (strict)
#    Generate Origin Certificate, save cert.pem + key.pem locally

# 4. fill in .env.prod
cp .env.prod.example .env.prod
$EDITOR .env.prod

# 5. one command: bootstraps + syncs env/certs + triggers CI deploy
make deploy VM_IP=$VM_IP CERT=path/to/cert.pem KEY=path/to/key.pem

# 6. verify
make deploy-status DOMAIN=echo-app.net
```

## Routine deploys (after step 1-5 once)

```bash
git push origin main      # CI builds, scans, deploys
make deploy-watch         # tail the GHA run
make deploy-status        # curl the public health endpoints
```

If the VM was restarted and got a new IP, just re-run step 5 (`make deploy
VM_IP=<new-ip>`) — the script is idempotent.

## What `make deploy` does

[`deploy/sync-vm.sh`](deploy/sync-vm.sh) is the underlying script. It:

1. Tests SSH access as `deploy@$VM_IP`. If it fails it prints the likely
   fix (host key changed, deploy user missing).
2. Bootstraps the VM if `/usr/bin/docker` or the repo isn't there yet.
   Uses [`deploy/setup-vm.sh`](deploy/setup-vm.sh) — installs Docker,
   ufw, swap, log caps, `live-restore`, the backup cron, pre-pulls infra
   images.
3. Copies `.env.prod` from your laptop to the VM, mode 600.
4. Copies `cert.pem` and `key.pem` if you passed them.
5. Runs [`deploy/preflight.sh`](deploy/preflight.sh) on the VM — fails
   loudly if anything's missing.
6. Triggers the "Build & Deploy" GitHub workflow at the local HEAD via
   `gh workflow run`.

## Inside `vm-deploy.sh` (called by CI)

[`deploy/vm-deploy.sh`](deploy/vm-deploy.sh):

1. Runs preflight (same script) — bails fast on bad state.
2. Captures the currently-deployed SHA from the running api-gateway
   container's image tag.
3. `git checkout` the new SHA so compose files match the registry images.
4. `docker compose pull` (parallel).
5. `docker compose up -d --wait --wait-timeout 180s`.
6. **On failure:** dumps last 40 log lines from each unhealthy service via
   [`deploy/health-gate.sh`](deploy/health-gate.sh), then automatically
   rolls back to the previous SHA. The previous stack comes back up; CI
   exits non-zero so you know to investigate.
7. Prunes images older than 72h.

## Make targets quick reference

### Laptop-side ([infra/make/deploy.mk](infra/make/deploy.mk))

| Command | What it does |
|---|---|
| `make deploy VM_IP=<ip> [CERT=… KEY=…]` | End-to-end bootstrap + sync + trigger CI |
| `make deploy-watch` | Tail the latest GHA run |
| `make deploy-status DOMAIN=<host>` | Curl `/caddy-health` and `/api/v1/health` |
| `make deploy-rollback SHA=<sha>` | Dispatch CI to redeploy a previous SHA |
| `make deploy-ssh VM_IP=<ip>` | Open shell on the VM |
| `make deploy-logs VM_IP=<ip> SVC=<svc>` | Tail one service's logs over SSH |

### Local prod-test ([infra/make/prod.mk](infra/make/prod.mk))

| Command | What it does |
|---|---|
| `make prod-local-up` | Build images + bring up the prod stack on your laptop |
| `make prod-local-down` | Stop (keep volumes) |
| `make prod-local-nuke` | Stop + delete volumes |
| `make prod-local-logs SVC=<svc>` | Tail one service |

### VM-side ([infra/make/vm.mk](infra/make/vm.mk)) — run inside the VM

| Command | What it does |
|---|---|
| `make vm-preflight` | Validate env/certs/docker/compose state |
| `make vm-up` | Pull latest images and bring up the stack |
| `make vm-down` | Stop everything |
| `make vm-restart [SVC=foo]` | Restart one or all app services |
| `make vm-status` | Show running services + health |
| `make vm-logs SVC=<svc>` | Tail logs |
| `make vm-shell SVC=<svc>` | Open shell in a container |
| `make vm-psql` | Open psql against the in-docker Postgres |
| `make vm-redis-cli` | redis-cli against in-docker Redis |
| `make vm-init-db` | Re-run idempotent schema loader |
| `make vm-migrate` | Apply pending `database/migrations/*.sql` |
| `make vm-backup` | Trigger a one-off Postgres backup right now |
| `make vm-prune` | Aggressive image / build cache prune |

## .env.prod cheat sheet

`.env.prod.example` documents every variable. The mandatory ones:

| Var | Value |
|---|---|
| `IMAGE_REGISTRY` | `ghcr.io/<github-user>/echo-backend` |
| `DOMAIN` | `echo-app.net` |
| `POSTGRES_PASSWORD` / `DB_PASSWORD` | strong random; both must match |
| `REDIS_PASSWORD` | strong random |
| `JWT_SECRET_KEY` | strong random (≥64 bytes) |
| `MEILI_MASTER_KEY` | strong random |
| `STORAGE_*` | Cloudflare R2 credentials (media-service) |

Optional but recommended (especially for auth flows):
`MAILGUN_*`, `EMAIL_VERIFICATION_*`. SMS / OAuth / Firebase keys are stubs;
fill them in if you wire those features up.

Generate strong values with:

```bash
openssl rand -base64 48 | tr -d '/+=' | head -c 32
```

## Operations

### Tail logs

```bash
make deploy-logs VM_IP=<ip> SVC=ws-service
```

### Manually run migrations

```bash
ssh -i ~/.ssh/echo_deploy deploy@<VM_IP>
cd /opt/echo-backend
make vm-migrate
```

### Manual rollback to a previous SHA

```bash
make deploy-rollback SHA=abcdef1234
make deploy-watch
```

The `build` job is auto-skipped (the previous SHA's images are already in
GHCR), so a rollback only takes the deploy step (~30s). The VM-side
`vm-deploy.sh` also has automatic rollback on `--wait` failure, so manual
rollback is rarely needed.

### Backups

`infra/scripts/backup-postgres.sh` runs nightly via cron at 03:17 UTC
(registered by `setup-vm.sh`). 14-day retention. Optionally ships off-box
to GCS or S3 if you set `BACKUP_REMOTE_URL` in `.env.prod`.

### Tear down

```bash
gcloud compute instances delete echo-backend-vm --zone=us-central1-a
```

## What this setup does NOT give you

- Multi-VM HA (single box; ws-service runs as one node).
- Zero-downtime deploys (`compose up --wait` cycles services with brief blips).
- Read replicas / PgBouncer (Postgres runs in-process; lift to managed if you outgrow the box).
- SBOM / image signing (Trivy CVE scan only — `cosign sign` is one workflow step away).

## Common failures

### `Permission denied (publickey)` when SSHing as deploy
Either the public key wasn't installed, the wrong key is being offered, or
OS Login is intercepting. `gcloud compute ssh` always works as a fallback.

### `dependency postgres failed to start`
Run preflight first: `ssh deploy@<VM_IP> 'cd /opt/echo-backend && make vm-preflight'`.
Then look at logs: `make deploy-logs VM_IP=<ip> SVC=postgres`. Most common
cause: incompatible permissions on the `postgres_data` volume after
switching image versions.

### `failed to compute cache key: ... not found` during build
A file the Dockerfile tries to COPY isn't in git. Common: `*.mmdb` for
location-service (fetched in CI), `services/*/tls/*` (gitignored),
`services/*/.env` (gitignored).

### Caddy can't start: TLS error
`make deploy-logs VM_IP=<ip> SVC=caddy`. Either `cert.pem` / `key.pem` is
missing in `/opt/echo-backend/secrets/`, the key permissions aren't 600,
or the cert chain is corrupt. Re-sync: `make deploy VM_IP=<ip> CERT=…
KEY=…`.

### Cloudflare Origin Cert renewal
Origin Certificates are valid 15 years. When you eventually rotate, scp
the new files over and `ssh deploy@<VM_IP> 'cd /opt/echo-backend && make
vm-restart SVC=caddy'`.
