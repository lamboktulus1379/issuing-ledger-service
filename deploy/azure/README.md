Deploying "gra" to an Azure VM (Nginx + Let's Encrypt)

This guide uses deploy/azure/deploy_gra_vm.sh to build and deploy the Go backend (service name: gra) to an Ubuntu-based Azure VM, fronted by Nginx with TLS via Let’s Encrypt.

Defaults from the script:
- DOMAIN: gra.tulus.space
- APP_PORT: 10011
- SERVICE_NAME: gra
- Systemd unit: /etc/systemd/system/gra.service
- App dir on VM: /opt/gra
- Env file on VM: /etc/opt/gra.env

Prerequisites
- DNS A/AAAA record for your DOMAIN points to the VM public IP.
- VM is reachable via SSH and has outbound internet access.
- On your local machine: bash, ssh, scp, Go toolchain.
- VM OS: Ubuntu/Debian with sudo available. The script installs Nginx and Certbot if missing.
- Open ports: 22 (SSH), 80 (HTTP for ACME), 443 (HTTPS).

Quick start (one command)
Set the minimum required variables, then run the deploy script from the repo root:

```bash
# From go-project repository root
export VM_HOST="<vm-ip-or-host>"
export VM_USER="gra"                       # or your VM user
export VM_SSH_KEY="$HOME/.ssh/id_rsa"     # path to your SSH private key
# Optional overrides
export DOMAIN="gra.tulus.space"           # change if using a different domain
export APP_PORT="10011"
# Strong secret required for JWT signature validation
export SECRET_KEY="<generate-a-strong-random-string>"

bash deploy/azure/deploy_gra_vm.sh
```

What the script does:
- Builds a Linux AMD64 binary locally.
- Creates a systemd unit for gra, and a ready Nginx TLS vhost for your DOMAIN.
- Uploads artifacts to the VM and installs them under /opt/gra.
- Creates/updates /etc/opt/gra.env with your environment settings.
- Enables and restarts the gra service.
- Configures Nginx. Obtains a certificate with Certbot (webroot). Reloads Nginx.

Visit: https://<DOMAIN>

Important environment variables
These are written to /etc/opt/gra.env on the VM and loaded by systemd:

- Core
  - ENV=prod
  - APP_PORT (default 10011)
  - TLS_ENABLED=0 (TLS is terminated at Nginx)
  - SECRET_KEY (REQUIRED for JWT auth)
- Database (optional; default vendor mssql)
  - DB_VENDOR (e.g. mssql)
  - MSSQL_* values: MSSQL_HOST, MSSQL_PORT, MSSQL_DB_NAME, MSSQL_USER, MSSQL_PASSWORD
- CORS
  - ALLOWED_ORIGINS (comma-separated list; wildcards like https://*.tulus.tech supported)
  - CORS_ALLOWED_ORIGINS (alternative variable if you prefer to separate concerns)
- YouTube (optional)
  - YOUTUBE_CLIENT_ID, YOUTUBE_CLIENT_SECRET, YOUTUBE_REDIRECT_URL
  - YOUTUBE_API_KEY, YOUTUBE_CHANNEL_ID
  - YOUTUBE_ACCESS_TOKEN, YOUTUBE_REFRESH_TOKEN
  - YOUTUBE_ENABLED (default true), YOUTUBE_MODE (default live)

You can pass any of these as environment variables when invoking the deploy script (they’ll be persisted into /etc/opt/gra.env).

Restarting the service and applying env changes
- Edit env file on the VM:

```bash
# On the VM
sudoedit /etc/opt/gra.env
```

- Apply changes by restarting the service (daemon-reload not required unless the unit file changed):

```bash
sudo systemctl restart gra
# Check status and logs
systemctl status --no-pager gra
journalctl -u gra -n 100 --no-pager
```

- To reload only Nginx after changing Nginx config:

```bash
sudo nginx -t && sudo systemctl reload nginx
```

Redeploying new code (rolling update)
Re-run the deploy script from your workstation. It preserves your existing config-prod.json and /etc/opt/gra.env:

```bash
# From go-project repository root on your machine
VM_HOST=<vm-ip> VM_USER=<vm-user> VM_SSH_KEY=~/.ssh/id_rsa DOMAIN=<your-domain> \
SECRET_KEY=<same-as-before> bash deploy/azure/deploy_gra_vm.sh
```

The script will rebuild, upload, restart the service, and reload Nginx if needed.

TLS and Nginx details
- Nginx site files are under /etc/nginx/sites-available/<DOMAIN> with a symlink in sites-enabled/.
- Certificates are stored at /etc/letsencrypt/live/<DOMAIN>/.
- If you need to (re)issue certificates manually:

```bash
# On the VM
sudo certbot certonly --webroot -w /var/www/certbot -d <DOMAIN> \
  --non-interactive --agree-tos -m <your-email>
```

Health checks
- Local app (on the VM):

```bash
curl -fsS http://127.0.0.1:10011/ -m 5 || true
curl -fsS http://127.0.0.1:10011/healthz -m 5 || true
```

- Public endpoint:

```bash
curl -k -fsS https://<DOMAIN>/ -m 10 || true
```

Troubleshooting
- 401s from API after login: Ensure SECRET_KEY in /etc/opt/gra.env matches the one used to sign JWTs.
- CORS errors: Confirm ALLOWED_ORIGINS includes your frontend origins. Wildcards like https://*.tulus.space are supported.
- Cert not installed yet: The script first enables HTTP-only Nginx, then attempts Certbot. If cert files are missing, rerun the Certbot command later and reload Nginx.
- Service not starting: Check journalctl -u gra -n 100 and validate that /opt/gra/gra-server exists and is executable.

Uninstall (optional)
```bash
sudo systemctl stop gra
sudo systemctl disable gra
sudo rm -f /etc/systemd/system/gra.service
sudo rm -rf /opt/gra
sudo systemctl daemon-reload
# Keep Nginx site/certs if the domain is still in use
```

---

For a more bare-metal, non-Azure flow, also see deploy/DEPLOYMENT.md (targets gra.tulus.tech). This Azure README aligns specifically with deploy/azure/deploy_gra_vm.sh.
