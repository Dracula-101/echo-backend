# Production deploy guide (echo-app.net on a GCP e2-small VM)

End-to-end runbook for deploying this backend to a single GCP e2-small VM,
fronted by Cloudflare with an Origin Certificate, with Postgres / Redis /
Kafka running as containers on the VM and Cloudflare R2 as the only
external dependency. Every command below should be runnable verbatim
(replace bracketed placeholders).

## 0. Prerequisites on your laptop

You need these locally before starting:

- `gcloud` CLI authenticated against the right GCP project.
- A Cloudflare account that owns `echo-app.net` (or whatever domain).
- Cloudflare Origin Certificate generated, with `cert.pem` and `key.pem`
  saved somewhere on disk. If you do not have them yet, generate them in
  the Cloudflare dashboard at SSL/TLS → Origin Server → Create Certificate.
- `docker` and `docker compose` installed (only required if you also want
  to test the prod stack on your laptop; not required for VM-only deploys).
- This repository cloned and the `.env.prod` file filled in (already done
  in the working tree if you have followed the earlier steps).

## 1. Provision the GCP VM

Create the VM. Adjust the project, zone, and machine type if needed.

```bash
gcloud compute instances create echo-backend-vm \
  --project=<YOUR_GCP_PROJECT> \
  --zone=us-central1-a \
  --machine-type=e2-small \
  --image-family=ubuntu-2204-lts \
  --image-project=ubuntu-os-cloud \
  --boot-disk-size=50GB \
  --boot-disk-type=pd-balanced \
  --tags=http-server,https-server
```

Get the external IP back (you will use it everywhere):

```bash
gcloud compute instances describe echo-backend-vm \
  --zone=us-central1-a \
  --format='get(networkInterfaces[0].accessConfigs[0].natIP)'
```

Throughout the rest of this document, `<VM_IP>` refers to that address.

## 2. Generate a deploy SSH key

Use a dedicated keypair, not your personal one.

```bash
ssh-keygen -t ed25519 -f ~/.ssh/echo_deploy -C "github-actions"
```

Press enter twice for an empty passphrase (CI cannot type a passphrase).

## 3. Create the `deploy` user on the VM and install the public key

GCP VMs do not ship with a `deploy` user. The metadata path is the
cleanest way to create one:

```bash
gcloud compute instances add-metadata echo-backend-vm \
  --zone=us-central1-a \
  --metadata="ssh-keys=deploy:$(cat ~/.ssh/echo_deploy.pub) github-actions"
```

Wait about thirty seconds for GCP to propagate the metadata, then verify:

```bash
ssh -i ~/.ssh/echo_deploy deploy@<VM_IP> 'whoami; sudo -n whoami'
```

You should see exactly:

```
deploy
root
```

If you instead get `Permission denied (publickey)`, the metadata has not
propagated yet, or your project has OS Login enabled. Diagnose:

```bash
gcloud compute ssh echo-backend-vm --zone=us-central1-a \
  --command='id deploy 2>&1; sudo cat /home/deploy/.ssh/authorized_keys 2>&1 | head -2'
```

If `deploy` does not exist, create it manually inside the VM:

```bash
gcloud compute ssh echo-backend-vm --zone=us-central1-a
# inside the VM:
sudo useradd -m -s /bin/bash deploy
sudo usermod -aG sudo deploy
echo 'deploy ALL=(ALL) NOPASSWD:ALL' | sudo tee /etc/sudoers.d/deploy
sudo chmod 0440 /etc/sudoers.d/deploy
sudo mkdir -p /home/deploy/.ssh
sudo chmod 700 /home/deploy/.ssh

# paste the literal contents of ~/.ssh/echo_deploy.pub between the
# PUBKEY markers (do NOT paste the placeholder text):
sudo tee /home/deploy/.ssh/authorized_keys > /dev/null <<'PUBKEY'
ssh-ed25519 AAAA... github-actions
PUBKEY
sudo chmod 600 /home/deploy/.ssh/authorized_keys
sudo chown -R deploy:deploy /home/deploy/.ssh
exit
```

Then re-run the verification command above.

## 4. Bootstrap the VM

From the repo root on your laptop:

```bash
ssh -i ~/.ssh/echo_deploy deploy@<VM_IP> \
  "REPO_URL=https://github.com/Dracula-101/echo-backend.git bash -s" \
  < deploy/setup-vm.sh
```

This script is idempotent. It installs Docker, the compose plugin,
unattended security upgrades, opens ports 22 / 80 / 443 in `ufw`, clones
the repository to `/opt/echo-backend`, allocates 2 GB swap, configures
the Docker daemon (capped logs, live-restore, BuildKit, raised ulimits),
pre-pulls the infrastructure images so the first deploy is fast, and
registers a nightly Postgres backup cron at 03:17 UTC.

Expected final output ends with:

```
==> Bootstrap complete.
```

## 5. Configure Cloudflare

In the Cloudflare dashboard for `echo-app.net`:

1. **DNS tab**: add an A record. Name `echo-app.net`, IPv4 `<VM_IP>`,
   proxy status orange (proxied).
2. **SSL/TLS tab → Overview**: set encryption mode to **Full (strict)**.
3. **SSL/TLS → Origin Server**: confirm you have an Origin Certificate
   that includes `echo-app.net` and that the matching `cert.pem` and
   `key.pem` are saved on your laptop.

Verify DNS is propagating from your laptop:

```bash
dig +short echo-app.net
```

You should see one of Cloudflare's edge IPs (104.x or 172.x), not your
VM's IP. That is correct, the orange cloud is proxying.

## 6. Copy secrets to the VM

The VM needs three things on disk: the env file, the Cloudflare Origin
Certificate, and the matching private key. None of these belong in git.

```bash
ssh -i ~/.ssh/echo_deploy deploy@<VM_IP> \
  'mkdir -p /opt/echo-backend/secrets && chmod 700 /opt/echo-backend/secrets'

scp -i ~/.ssh/echo_deploy \
    .env.prod \
    deploy@<VM_IP>:/opt/echo-backend/.env.prod

scp -i ~/.ssh/echo_deploy \
    <path/to/cert.pem> \
    <path/to/key.pem> \
    deploy@<VM_IP>:/opt/echo-backend/secrets/

ssh -i ~/.ssh/echo_deploy deploy@<VM_IP> \
  'chmod 600 /opt/echo-backend/.env.prod /opt/echo-backend/secrets/*.pem'
```

Replace `<path/to/cert.pem>` and `<path/to/key.pem>` with the actual
paths on your laptop. Both files must already include the full content
returned by Cloudflare; do not concatenate them or strip the BEGIN/END
markers.

## 7. Add GitHub Actions secrets

Go to https://github.com/Dracula-101/echo-backend/settings/secrets/actions
and create three repository secrets:

| Name | Value |
| --- | --- |
| `VM_SSH_HOST` | `<VM_IP>` |
| `VM_SSH_USER` | `deploy` |
| `VM_SSH_KEY`  | The full contents of `~/.ssh/echo_deploy` (the private key, including the `-----BEGIN OPENSSH PRIVATE KEY-----` and `-----END OPENSSH PRIVATE KEY-----` lines) |

To copy the private key contents into your clipboard:

```bash
pbcopy < ~/.ssh/echo_deploy
```

Then paste into the `VM_SSH_KEY` value field.

The optional `VM_SSH_PORT` defaults to 22 and can be omitted.

GHCR push uses the built-in `GITHUB_TOKEN`, so no additional secret is
needed for image publishing.

## 8. Trigger the first deploy

Push the `main` branch (or trigger the workflow manually):

```bash
git push origin main
```

Watch the run at:

```
https://github.com/Dracula-101/echo-backend/actions
```

The workflow does three jobs in order:

1. **build** — builds 8 service images in parallel on a GitHub-hosted
   runner, pushes to GHCR with both `:sha-<short>` and `:latest` tags.
   Uses two-tier cache (GHA + GHCR registry) so subsequent builds are
   fast.
2. **scan** — Trivy scans each image for CRITICAL and HIGH CVEs.
   Informational only by default.
3. **deploy** — SSHes into the VM, runs `deploy/vm-deploy.sh <sha>`,
   which pulls the images in parallel, runs the migration one-shot, then
   `compose up -d --wait` blocks until every healthcheck reports healthy
   (180 second timeout). On failure it dumps the last 40 log lines from
   each unhealthy container.

If anything fails, the workflow exits non-zero and nothing on the VM
changes (the previous deploy keeps running).

## 9. Verify the deploy

Once the workflow goes green:

```bash
curl -i https://echo-app.net/caddy-health
```

Expected: `HTTP/2 200` with body `ok`.

Test the API:

```bash
curl -i https://echo-app.net/api/v1/health
```

(Adjust the path to whatever your api-gateway exposes.)

WebSocket:

```bash
wscat -c wss://echo-app.net/ws
```

## 10. Operations

### Tail logs

```bash
ssh -i ~/.ssh/echo_deploy deploy@<VM_IP>
cd /opt/echo-backend
docker compose \
  -f infra/docker/docker-compose.prod.yml \
  --env-file .env.prod \
  --profile caddy \
  logs -f ws-service
```

Replace `ws-service` with any service name.

### Run a migration manually

```bash
ssh -i ~/.ssh/echo_deploy deploy@<VM_IP>
cd /opt/echo-backend
docker compose \
  -f infra/docker/docker-compose.prod.yml \
  --env-file .env.prod \
  --profile caddy \
  run --rm migrate
```

### Bootstrap the initial schema (only on a fresh Postgres)

The `migrate` service only applies `database/migrations/*.sql`, which
are idempotent drift fixes. On a brand-new Postgres you also need to
load the initial schema once:

```bash
ssh -i ~/.ssh/echo_deploy deploy@<VM_IP>
cd /opt/echo-backend
docker compose \
  -f infra/docker/docker-compose.prod.yml \
  --env-file .env.prod \
  --profile caddy \
  exec -T postgres psql -U echo -d echo_db < db-dump.sql
```

### Manual deploy of a specific git SHA

```bash
ssh -i ~/.ssh/echo_deploy deploy@<VM_IP>
cd /opt/echo-backend
PROFILES="caddy" bash deploy/vm-deploy.sh <full-git-sha>
```

### Rollback

Trigger the workflow manually with the previous SHA:

```
GitHub → Actions → Build & Deploy → Run workflow
  Branch: main
  Git SHA to deploy: <previous-sha>
  Compose profiles: caddy
```

The build job is automatically skipped (the previous SHA's images are
already in GHCR), so a rollback only takes the time of the SSH deploy
step (about 30 seconds).

### Backups

`infra/scripts/backup-postgres.sh` runs nightly via cron (registered by
the bootstrap script). Dumps land in `/var/backups/echo` gzipped,
retained 14 days. Logs go to `/var/log/echo-backup.log`.

To enable off-box backup uploads, set `BACKUP_REMOTE_URL` in
`.env.prod` to either `gs://bucket/path/` or `s3://bucket/path/`, then
re-copy the env file:

```bash
scp -i ~/.ssh/echo_deploy .env.prod deploy@<VM_IP>:/opt/echo-backend/.env.prod
```

The script needs `gsutil` or `aws` CLI installed on the VM for the
upload to work; otherwise it logs a warning and keeps the local copy.

### Force-pull latest images and restart

```bash
ssh -i ~/.ssh/echo_deploy deploy@<VM_IP>
cd /opt/echo-backend
docker compose \
  -f infra/docker/docker-compose.prod.yml \
  --env-file .env.prod \
  --profile caddy \
  pull --policy always

docker compose \
  -f infra/docker/docker-compose.prod.yml \
  --env-file .env.prod \
  --profile caddy \
  up -d --remove-orphans --wait --wait-timeout 180
```

### Tear down the VM

```bash
gcloud compute instances delete echo-backend-vm --zone=us-central1-a
```

## 11. Local prod-test (mirror of prod on your laptop)

Use this to catch issues before pushing.

```bash
cp .env.prod.example .env.prod
$EDITOR .env.prod                 # set CHANGE_ME values

make prod-local-up                # builds locally, brings up the stack
make prod-local-logs SVC=ws-service
make prod-local-down              # stops, keeps volumes
make prod-local-nuke              # stops, removes volumes
```

Endpoints (host-bound for local access):

- API gateway: http://localhost:8080/health
- WebSocket: ws://localhost:8086/ws
- Postgres: localhost:5432
- Redis: localhost:6379

The local stack does not start Caddy. You hit the api-gateway directly.
Everything else (image build, migrations, healthchecks, env wiring,
hardening) is identical to production.

## 12. Common failures and fixes

### `Permission denied (publickey)` when SSHing as deploy

Either the public key was not installed, the wrong key is being offered,
or OS Login is intercepting. Re-read section 3 and run the verification
command. If nothing else works, `gcloud compute ssh` always works as a
fallback to get inside the VM and inspect.

### `dependency postgres failed to start`

Postgres container did not flip to healthy in time. Check the actual
container logs:

```bash
ssh -i ~/.ssh/echo_deploy deploy@<VM_IP>
cd /opt/echo-backend
docker compose -f infra/docker/docker-compose.prod.yml --env-file .env.prod \
  logs postgres | tail -50
```

Most common causes: incompatible permissions on the `postgres_data`
volume after switching image versions, or the container has
`read_only: true` set without enough writable tmpfs paths. The compose
file ships with a config that works.

### `failed to compute cache key: ... not found` during build

Some file the Dockerfile tries to COPY does not exist on the CI runner.
Usually because it is `.gitignored` (e.g., MaxMind .mmdb files,
self-signed TLS certs, local `.env` files). Either remove the COPY from
the Dockerfile or add a workflow step that fetches / generates the file
before the docker build runs.

### `unable to resolve action ...`

The action version pin in `.github/workflows/deploy.yml` does not exist.
Check the action's repository releases page on GitHub and update to a
real tag.

### Caddy fails to start with a TLS error

Either `cert.pem` or `key.pem` is missing in `/opt/echo-backend/secrets/`,
the file permissions are wrong (need 600 for the key), or the cert chain
is corrupt. Inspect:

```bash
ssh -i ~/.ssh/echo_deploy deploy@<VM_IP> 'ls -la /opt/echo-backend/secrets/'
ssh -i ~/.ssh/echo_deploy deploy@<VM_IP> \
  'docker compose -f /opt/echo-backend/infra/docker/docker-compose.prod.yml --env-file /opt/echo-backend/.env.prod --profile caddy logs caddy | tail -30'
```

### Cert renewal

Cloudflare Origin Certificates are valid for 15 years. No renewal cron
needed. When you do eventually rotate, just SCP new files over the old
ones in `/opt/echo-backend/secrets/` and `docker compose ... restart caddy`.

## 13. What this setup does NOT give you

- **Multi-VM HA**. Single box; ws-service runs as one node.
- **Zero-downtime deploys**. `compose up --wait` cycles services with
  brief blips.
- **Read replicas, PgBouncer, etc.** Postgres runs in-process. If the
  dataset outgrows the box, lift to a managed service.
- **Automatic SBOM / image signing**. Trivy scans for CVEs only.
  `cosign sign` is one workflow step away if you need supply-chain
  attestations.
