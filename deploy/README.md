# Production deploy

Single-VM deploy that runs anywhere — GCP, Hetzner, DigitalOcean, AWS,
bare metal. The compose stack is sized via env vars so it scales from a
2 GB box to a 16 GB box without touching YAML.

```
┌───────────────────┐     push to main     ┌──────────────────────┐
│  GitHub Actions   │ ──────────────────▶  │  ghcr.io/<repo>/...  │
│  build matrix     │     :sha-<short>     │       (registry)     │
│  + Trivy scan     │     :latest          │                      │
└─────────┬─────────┘                      └──────────┬───────────┘
          │                                            │ pull (parallel)
          │ ssh deploy@vm                              ▼
          │                                  ┌────────────────────┐
          └────────────────────────────────▶ │   VM (any size)    │
              vm-deploy.sh <sha>             │  docker compose    │
              + compose up --wait            │  + Caddy (HTTPS)   │
                                             └────────────────────┘
```

## Why this shape

- **Build in CI, not on the VM.** Compiling Go on a small box is slow and
  unreliable. CI builds on a beefy GitHub runner, pushes to GHCR with
  layer cache + registry cache. The VM only `pull`s — typically 30s for
  the whole stack.
- **`compose up --wait`** blocks until every healthcheck reports healthy.
  No polling loops, no flaky sleep-and-pray.
- **Migrations as a one-shot dependency.** App services declare
  `depends_on: { migrate: completed_successfully }`, so a failing
  migration aborts the deploy instead of leaving a half-broken stack.
- **Caddy fronts the public internet.** Automatic HTTPS via Let's Encrypt
  when you set `DOMAIN`; HTTP-only fallback (`DOMAIN=:80`) for first-boot
  validation against the raw IP.
- **Hardened defaults.** `no-new-privileges`, `cap_drop: [ALL]`,
  `read_only` root FS where possible, capped logs, security headers in
  Caddy (HSTS, X-Frame-Options, etc.), env file at mode 600, UFW open
  only for SSH/HTTP/HTTPS.
- **Everything is env-configurable.** Resource limits, Postgres tuning,
  Kafka heap, log retention — all live in `.env.prod`. No compose edits
  needed when you upgrade the box.

---

## One-time setup

### 1. Provision the VM

Any 2 GB+ Linux box works. Examples:

```bash
# GCP
gcloud compute instances create echo-backend-vm \
  --zone=us-central1-a \
  --machine-type=e2-small \
  --image-family=ubuntu-2204-lts --image-project=ubuntu-os-cloud \
  --boot-disk-size=50GB

# Hetzner
hcloud server create --type cx22 --image ubuntu-22.04 --name echo-backend

# DigitalOcean
doctl compute droplet create echo-backend \
  --image ubuntu-22-04-x64 --size s-2vcpu-2gb --region nyc1
```

### 2. Add a deploy SSH key (don't reuse a personal key)

```bash
ssh-keygen -t ed25519 -f ~/.ssh/echo_deploy -C "github-actions"
ssh-copy-id -i ~/.ssh/echo_deploy.pub deploy@<VM_IP>
```

### 3. Bootstrap the VM

```bash
ssh deploy@<VM_IP> \
  "REPO_URL=https://github.com/<you>/echo-backend.git bash -s" \
  < deploy/setup-vm.sh
```

The script installs Docker + compose, opens 22/80/443, clones to
`/opt/echo-backend`, allocates swap, caps Docker logs, enables
`live-restore`, **pre-pulls all infra images** (so the first deploy is
fast), enables `unattended-upgrades`, and registers a nightly Postgres
backup cron.

### 4. Fill in `.env.prod` on the VM

```bash
ssh deploy@<VM_IP>
sudo -u deploy vi /opt/echo-backend/.env.prod
```

Mandatory:

| Var | Notes |
|---|---|
| `IMAGE_REGISTRY` | `ghcr.io/<github-user>/echo-backend` |
| `POSTGRES_PASSWORD` / `DB_PASSWORD` | Same value, `openssl rand -base64 32` |
| `REDIS_PASSWORD` | `openssl rand -base64 32` |
| `JWT_SECRET_KEY` | `openssl rand -base64 64` |
| `DOMAIN` | Hostname pointed at the VM, or `:80` for plain HTTP |
| `ACME_EMAIL` | Where Let's Encrypt sends expiry warnings |

Optional (defaults are sized for ~2 GB; bump on bigger VMs):

`RES_*` resource limits, `PG_SHARED_BUFFERS`, `KAFKA_HEAP_OPTS`,
`GOMEMLIMIT`, etc. See the example file for the full list.

### 5. Add GitHub repo secrets

Settings → Secrets and variables → Actions:

| Secret | Value |
|---|---|
| `VM_SSH_HOST` | The VM's public IP |
| `VM_SSH_USER` | `deploy` |
| `VM_SSH_KEY`  | Contents of `~/.ssh/echo_deploy` (full PEM) |
| `VM_SSH_PORT` | `22` (omit if default) |

GHCR push uses the built-in `GITHUB_TOKEN`. For private packages, log in
on the VM once: `echo <PAT> | docker login ghcr.io -u <user> --password-stdin`.

### 6. First deploy

Push to `main` (or trigger **Build & Deploy** manually). The workflow:

1. **build** (~3–5 min): 8 service images in parallel; GHA cache + GHCR
   registry cache layered. Subsequent runs reuse layers heavily.
2. **scan** (~1 min, non-blocking): Trivy scans each image for CRITICAL
   / HIGH CVEs.
3. **deploy** (~30s after pull): SSHes into the VM, runs
   `vm-deploy.sh <sha>`, which pulls images in parallel and `up --wait`s
   the stack with a 180s timeout. Failure dumps per-service logs.

---

## Local prod-test (catch errors before pushing)

Mirrors production on your laptop using the same compose files. Builds
images locally rather than pulling — Dockerfile bugs and missing env
vars fail fast on your machine, not on the VM.

```bash
cp .env.prod.example .env.prod
$EDITOR .env.prod              # set CHANGE_ME values; DOMAIN=:80 is fine

make prod-local-up             # builds + starts the stack with --wait
make prod-local-health         # the same gate the VM uses
make prod-local-logs SVC=ws-service
make prod-local-migrate        # rerun migrations standalone
make prod-local-down           # stop (keeps volumes)
make prod-local-nuke           # stop + wipe volumes
```

Endpoints when running locally:

- API gateway: `http://localhost:8080/health`
- WebSocket:   `ws://localhost:8086/ws`
- Postgres:    `localhost:5432`
- Redis:       `localhost:6379`

The local stack does **not** start Caddy — you reach the gateway
directly. That's the only intended difference. Everything else (image
build, migrations, healthchecks, env wiring, restart policies, hardening
options) is identical, so any failure surface in prod is reproducible
locally.

To exercise optional services too:

```bash
make prod-local-up PROFILES="full"
```

---

## Operations

### Tail logs on the VM

```bash
ssh deploy@<VM_IP>
cd /opt/echo-backend
docker compose -f infra/docker/docker-compose.prod.yml --env-file .env.prod \
  --profile caddy logs -f ws-service
```

### Manual deploy of a specific SHA

```bash
ssh deploy@<VM_IP>
cd /opt/echo-backend
PROFILES="caddy" bash deploy/vm-deploy.sh <sha>
```

### Rollback

Trigger **Build & Deploy** via `workflow_dispatch` with the previous SHA
in the `sha` input. The build job is auto-skipped (images already in
GHCR), so rollback is just the deploy step — under a minute.

### Backups

`infra/scripts/backup-postgres.sh` runs nightly via cron. Dumps land in
`/var/backups/echo` gzipped, retained 14 days by default. Logs go to
`/var/log/echo-backup.log`.

Set `BACKUP_REMOTE_URL=gs://bucket/path/` or `s3://bucket/path/` in
`.env.prod` and the script will also ship dumps off-box (requires
`gsutil` or `aws` CLI installed). Off-box upload failures are
non-fatal — the local copy is always good.

### Scaling up

You don't need to touch compose. Edit `.env.prod`:

```diff
-RES_GO_MEMORY=256M
+RES_GO_MEMORY=1G
-PG_SHARED_BUFFERS=128MB
+PG_SHARED_BUFFERS=2GB
-KAFKA_HEAP_OPTS=-Xmx384m -Xms256m
+KAFKA_HEAP_OPTS=-Xmx2g -Xms1g
```

Then re-deploy. New cgroup limits apply on container recreate.

### Tear it down

```bash
gcloud compute instances delete echo-backend-vm --zone=us-central1-a
# or:  hcloud server delete echo-backend
# or:  doctl compute droplet delete echo-backend
```

---

## What this setup does NOT give you

- **Multi-VM HA** — single box. ws-service runs as one node, so the
  cross-instance fan-out hooks in code stay dormant.
- **Zero-downtime deploys** — `compose up --wait` cycles services with
  brief blips. Acceptable for most apps; switch to a load-balanced
  multi-instance setup if you need true rolling updates.
- **Read replicas, PgBouncer** — Postgres runs in-process. If your
  dataset outgrows the box, lift it to a managed service.
- **Automatic SBOM / image signing** — wired hooks aren't there.
  `cosign sign` is one workflow step away if you need supply-chain
  attestations.
