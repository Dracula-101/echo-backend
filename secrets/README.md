# Production secrets directory

Mounted into the Caddy container at `/etc/caddy/certs` (read-only).

Required files for production:
- `cert.pem` — Cloudflare Origin Cert (or any TLS cert chain)
- `key.pem`  — matching private key

These are NOT committed to git (see `.gitignore`). On the VM:

```bash
scp -i ~/.ssh/echo_deploy ./secrets/cert.pem ./secrets/key.pem \
    deploy@<VM_IP>:/opt/echo-backend/secrets/
ssh deploy@<VM_IP> 'chmod 600 /opt/echo-backend/secrets/*.pem'
```
