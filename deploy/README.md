# Deploying to a GCP E2 VM via GitHub Actions

Single-VM deploy. No GKE, no Cloud SQL, no Memorystore — everything runs in
docker-compose on one Compute Engine box. Fits the free-tier intent but be
honest with yourself about the resource math (see "Sizing" below).

## One-time setup

### 1. Provision the VM

```bash
PROJECT=your-gcp-project
gcloud compute instances create echo-backend-vm \
  --project=$PROJECT \
  --zone=us-central1-a \
  --machine-type=e2-medium \
  --image-family=ubuntu-2204-lts --image-project=ubuntu-os-cloud \
  --boot-disk-size=50GB --boot-disk-type=pd-balanced \
  --tags=http-server,https-server \
  --metadata=enable-oslogin=FALSE
```

### 2. Add an SSH key for CI

Generate a deploy keypair on your laptop (don't reuse a personal key):

```bash
ssh-keygen -t ed25519 -f ~/.ssh/echo_deploy -C "github-actions"
gcloud compute instances add-metadata echo-backend-vm \
  --zone=us-central1-a \
  --metadata="ssh-keys=deploy:$(cat ~/.ssh/echo_deploy.pub)"
```

### 3. Bootstrap the VM

```bash
ssh -i ~/.ssh/echo_deploy deploy@<VM_PUBLIC_IP> 'bash -s' < deploy/setup-vm.sh
```

This installs Docker + compose plugin, clones the repo to `/opt/echo-backend`,
sets up the firewall, configures log-rotation, and creates a `.env` stub.

### 4. Edit `.env` on the VM

```bash
ssh deploy@<VM_PUBLIC_IP>
sudo -u deploy vi /opt/echo-backend/.env
```

Set real values for `POSTGRES_PASSWORD`, `REDIS_PASSWORD`, `JWT_SECRET_KEY`.
Anything missing here causes services to crash-loop.

### 5. Add GitHub Secrets

In the repo's Settings → Secrets and variables → Actions:

| Secret | Value |
|---|---|
| `VM_SSH_HOST` | The VM's public IP |
| `VM_SSH_USER` | `deploy` |
| `VM_SSH_KEY` | Contents of `~/.ssh/echo_deploy` (the **private** key, full PEM) |
| `VM_SSH_PORT` | `22` (omit if default) |

### 6. First deploy

Push to `main`, or trigger the workflow manually from the Actions tab.

The workflow SSHes in, `git checkout`s the pushed SHA, and runs
`docker compose up -d --build`. First build takes ~10–20 minutes on a small
VM (Go workspace + 10 services). Subsequent deploys reuse the build cache.

## Sizing — honest numbers

The full stack is **Postgres + Redis + Kafka + Zookeeper + 10 Go services**.
That doesn't fit on free tier. Realistic minimums:

| VM type | RAM | What runs | Verdict |
|---|---|---|---|
| `e2-micro` (free tier) | 1 GB | Nothing | Will OOM during compose up |
| `e2-small` | 2 GB | Postgres + Redis + 2–3 services | Tight; reasonable for a demo |
| `e2-medium` | 4 GB | Most of the stack, no Kafka | Good for testing the API path |
| `e2-standard-2` | 8 GB | Everything including Kafka | Comfortable; ~$50/mo |
| `e2-standard-4` | 16 GB | Everything + headroom | What the prod compose was designed for |

Free options if you can't pay:
- Use the **$300 GCP credit** (90-day trial) on `e2-standard-2` — gives ~6 months.
- Disable the heavy stub services (analytics, notification, media if unused).
- Drop Kafka and use a smaller message bus, OR run Kafka with reduced heap
  (`KAFKA_HEAP_OPTS="-Xmx256m -Xms128m"` in the compose — already set in
  `docker-compose.prod.yml`, but Kafka still wants ~512MB practical floor).

## What this deploy does NOT give you

- HTTPS termination — add Caddy or `nginx-proxy` + Let's Encrypt manually
- Backups for Postgres — set up a cron `pg_dump` to GCS
- Auto-restart on VM reboot — `docker compose --restart unless-stopped` is
  set per service in the compose, but reboot the VM and you're fine
- Multi-instance ws-service — it's one node, so the cross-instance fan-out
  we shipped (P0.1) is unused; that's correct for this topology
- Zero-downtime rolling updates — `up -d --build` cycles services one at a
  time but there will be brief blips

## Common operations

**Tail logs:**
```bash
ssh deploy@<HOST>
cd /opt/echo-backend
docker compose -f infra/docker/docker-compose.yml -f infra/docker/docker-compose.prod.yml logs -f ws-service
```

**Run DB migrations:**
```bash
docker compose -f infra/docker/docker-compose.yml -f infra/docker/docker-compose.prod.yml \
  exec postgres psql -U echo -d echo_db -f /docker-entrypoint-initdb.d/01_schema.sql
```
(Fresh boot already applies `database/init/` via the postgres image's
`docker-entrypoint-initdb.d` mount.)

**Roll back:**
The workflow checks out a specific SHA. To revert, push a revert commit, or
trigger `workflow_dispatch` from a known-good ref.

**Tear it down:**
```bash
gcloud compute instances delete echo-backend-vm --zone=us-central1-a
```
