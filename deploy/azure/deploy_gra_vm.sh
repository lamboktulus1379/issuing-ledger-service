#!/usr/bin/env bash
set -euo pipefail

# Deploy go-project to Azure VM behind Nginx + Let's Encrypt as gra.tulus.space

# Config (override via environment variables)
DOMAIN="${DOMAIN:-gra.tulus.space}"
APP_PORT="${APP_PORT:-10011}"
SERVICE_NAME="${SERVICE_NAME:-gra}"

# Database (override via environment variables)
# Defaults assume Azure SQL; set MSSQL_PASSWORD via env before running for production
DB_VENDOR="${DB_VENDOR:-mssql}"
MSSQL_HOST="${MSSQL_HOST:-gra-db.database.windows.net}"
MSSQL_PORT="${MSSQL_PORT:-1433}"
MSSQL_DB_NAME="${MSSQL_DB_NAME:-typing}"
MSSQL_USER="${MSSQL_USER:-gra}"
MSSQL_PASSWORD="${MSSQL_PASSWORD:-}"

# VM connection (override these for your environment)
VM_HOST="${VM_HOST:-57.158.26.137}"
VM_USER="${VM_USER:-gra}"
VM_SSH_KEY="${VM_SSH_KEY:-$HOME/Projects/Typing/deploy/infra-azure/gra-ssh.pem}"

PROJECT_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
BUILD_DIR="$PROJECT_ROOT/dist"
RELEASE_TGZ="$BUILD_DIR/${SERVICE_NAME}-release.tgz"

echo "==> Loading YouTube env (if present) from $PROJECT_ROOT/config.env"
# Best-effort import of YouTube-related variables from local config.env
if [[ -f "$PROJECT_ROOT/config.env" ]]; then
  while IFS='=' read -r key val; do
    [[ -z "$key" ]] && continue
    [[ "$key" =~ ^# ]] && continue
    key_trimmed="${key//[[:space:]]/}"
    # Strip surrounding quotes from value
    val_trimmed="${val%\r}"
    val_trimmed="${val_trimmed%\n}"
    val_trimmed="${val_trimmed#\' }"; val_trimmed="${val_trimmed#\' }"; val_trimmed="${val_trimmed%\' }"; val_trimmed="${val_trimmed%\' }"
    val_trimmed="${val_trimmed#\"}"; val_trimmed="${val_trimmed%\"}"
    case "$key_trimmed" in
      YOUTUBE_CLIENT_ID|YOUTUBE_CLIENT_SECRET|YOUTUBE_REDIRECT_URL|YOUTUBE_API_KEY|YOUTUBE_CHANNEL_ID|YOUTUBE_ACCESS_TOKEN|YOUTUBE_REFRESH_TOKEN|SECRET_KEY|ALLOWED_ORIGINS|CORS_ALLOWED_ORIGINS)
        export "$key_trimmed"="$val_trimmed";;
    esac
  done < <(grep -E '^(YOUTUBE_|SECRET_KEY|ALLOWED_ORIGINS|CORS_ALLOWED_ORIGINS)' "$PROJECT_ROOT/config.env" | grep -v '^#' || true)
fi

echo "==> Building Linux AMD64 binary"
mkdir -p "$BUILD_DIR"
pushd "$PROJECT_ROOT" >/dev/null
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "$BUILD_DIR/${SERVICE_NAME}-server"
popd >/dev/null

echo "==> Preparing release archive"
cat >"$BUILD_DIR/gra.service" <<EOF
[Unit]
Description=Go Project (${SERVICE_NAME})
After=network.target

[Service]
User=${VM_USER}
Group=${VM_USER}
Environment=ENV=prod
Environment=APP_PORT=${APP_PORT}
Environment=TLS_ENABLED=0
WorkingDirectory=/opt/${SERVICE_NAME}
ExecStart=/opt/${SERVICE_NAME}/${SERVICE_NAME}-server
Restart=always
RestartSec=3
KillMode=process

[Install]
WantedBy=multi-user.target
EOF

cat >"$BUILD_DIR/nginx-${DOMAIN}.conf" <<EOF
server {
    listen 80;
    server_name ${DOMAIN};
    location /.well-known/acme-challenge/ { root /var/www/certbot; }
  location / { return 301 https://\$host\$request_uri; }
}

server {
    listen 443 ssl http2;
    server_name ${DOMAIN};

    ssl_certificate /etc/letsencrypt/live/${DOMAIN}/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/${DOMAIN}/privkey.pem;
    include /etc/letsencrypt/options-ssl-nginx.conf;
    ssl_dhparam /etc/letsencrypt/ssl-dhparams.pem;

    location / {
        proxy_pass http://127.0.0.1:${APP_PORT};
  proxy_set_header Host \$host;
  proxy_set_header X-Real-IP \$remote_addr;
  proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
  proxy_set_header X-Forwarded-Proto https;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
    }

    location /healthz {
        proxy_pass http://127.0.0.1:${APP_PORT}/healthz;
  proxy_set_header Host \$host;
    }
}
EOF

# Optional production config (can be edited after first deploy)
cat >"$BUILD_DIR/config-prod.json" <<EOF
{
  "app": {
    "port": 10011,
    "secretKey": "change_me",
    "tlsEnabled": false
  },
  "logger": { "format": "2006-02-01" },
  "youtube": {
    "apiKey": "",
    "clientId": "",
    "clientSecret": "",
    "redirectURI": "https://${DOMAIN}/auth/youtube/callback",
    "channelId": "",
    "scopes": [
      "https://www.googleapis.com/auth/youtube.readonly",
      "https://www.googleapis.com/auth/youtube.upload",
      "https://www.googleapis.com/auth/youtube.force-ssl"
    ]
  },
  "oauth": {
    "facebook": {
      "clientId": "",
      "clientSecret": "",
      "redirectURI": "https://${DOMAIN}/auth/facebook/callback"
    },
    "twitter": {"clientId": "", "clientSecret": "", "redirectURI": ""}
  }
}
EOF

# Create archive without macOS extended attributes/metadata
COPYFILE_DISABLE=1 tar czf "$RELEASE_TGZ" -C "$BUILD_DIR" "${SERVICE_NAME}-server" "gra.service" "nginx-${DOMAIN}.conf" config-prod.json

echo "==> Uploading release to VM ${VM_USER}@${VM_HOST}"
ssh_opts="-i \"$VM_SSH_KEY\" -o StrictHostKeyChecking=no"
scp -i "$VM_SSH_KEY" -o StrictHostKeyChecking=no "$RELEASE_TGZ" "${VM_USER}@${VM_HOST}:/tmp/${SERVICE_NAME}-release.tgz"

echo "==> Running remote install"
ssh -i "$VM_SSH_KEY" -o StrictHostKeyChecking=no "${VM_USER}@${VM_HOST}" \
  SERVICE_NAME="$SERVICE_NAME" \
  DOMAIN="$DOMAIN" \
  APP_PORT="$APP_PORT" \
  DB_VENDOR="$DB_VENDOR" \
  MSSQL_HOST="$MSSQL_HOST" \
  MSSQL_PORT="$MSSQL_PORT" \
  MSSQL_DB_NAME="$MSSQL_DB_NAME" \
  MSSQL_USER="$MSSQL_USER" \
  MSSQL_PASSWORD="$MSSQL_PASSWORD" \
  YOUTUBE_CLIENT_ID="${YOUTUBE_CLIENT_ID-}" \
  YOUTUBE_CLIENT_SECRET="${YOUTUBE_CLIENT_SECRET-}" \
  YOUTUBE_REDIRECT_URL="${YOUTUBE_REDIRECT_URL-}" \
  YOUTUBE_API_KEY="${YOUTUBE_API_KEY-}" \
  YOUTUBE_CHANNEL_ID="${YOUTUBE_CHANNEL_ID-}" \
  YOUTUBE_ACCESS_TOKEN="${YOUTUBE_ACCESS_TOKEN-}" \
  YOUTUBE_REFRESH_TOKEN="${YOUTUBE_REFRESH_TOKEN-}" \
  YOUTUBE_ENABLED="${YOUTUBE_ENABLED-}" \
  YOUTUBE_MODE="${YOUTUBE_MODE-}" \
  SECRET_KEY="${SECRET_KEY-}" \
  ALLOWED_ORIGINS="${ALLOWED_ORIGINS-}" \
  CORS_ALLOWED_ORIGINS="${CORS_ALLOWED_ORIGINS-}" \
  'bash -s' <<'EOSSH'
set -eo pipefail

sudo mkdir -p /opt/$SERVICE_NAME
# Preserve existing config if present
if [ -f "/opt/$SERVICE_NAME/config-prod.json" ]; then sudo cp "/opt/$SERVICE_NAME/config-prod.json" "/tmp/config-prod.json.bak"; fi
sudo tar xzf /tmp/$SERVICE_NAME-release.tgz -C /opt/$SERVICE_NAME --no-same-owner
if [ -f "/tmp/config-prod.json.bak" ]; then sudo mv "/tmp/config-prod.json.bak" "/opt/$SERVICE_NAME/config-prod.json"; fi
sudo mv /opt/$SERVICE_NAME/gra.service /etc/systemd/system/$SERVICE_NAME.service
# config file already extracted to /opt/$SERVICE_NAME/config-prod.json

# Systemd environment (app + database)
sudo mkdir -p /etc/opt
if [ ! -f "/etc/opt/$SERVICE_NAME.env" ]; then
    cat <<EOF | sudo tee "/etc/opt/$SERVICE_NAME.env" >/dev/null
ENV=prod
APP_PORT="$APP_PORT"
TLS_ENABLED=0
DB_VENDOR="$DB_VENDOR"
MSSQL_HOST="$MSSQL_HOST"
MSSQL_PORT="$MSSQL_PORT"
MSSQL_DB_NAME="$MSSQL_DB_NAME"
MSSQL_USER="$MSSQL_USER"
MSSQL_PASSWORD="$MSSQL_PASSWORD"
# YouTube configuration (optional)
YOUTUBE_CLIENT_ID="${YOUTUBE_CLIENT_ID:-}"
YOUTUBE_CLIENT_SECRET="${YOUTUBE_CLIENT_SECRET:-}"
YOUTUBE_REDIRECT_URL="${YOUTUBE_REDIRECT_URL:-https://$DOMAIN/auth/youtube/callback}"
YOUTUBE_API_KEY="${YOUTUBE_API_KEY:-}"
YOUTUBE_CHANNEL_ID="${YOUTUBE_CHANNEL_ID:-}"
YOUTUBE_ACCESS_TOKEN="${YOUTUBE_ACCESS_TOKEN:-}"
YOUTUBE_REFRESH_TOKEN="${YOUTUBE_REFRESH_TOKEN:-}"
YOUTUBE_ENABLED="${YOUTUBE_ENABLED:-true}"
YOUTUBE_MODE="${YOUTUBE_MODE:-live}"
SECRET_KEY="${SECRET_KEY:-}"
# CORS: comma-separated list, supports wildcards like https://*.tulus.tech
ALLOWED_ORIGINS="${ALLOWED_ORIGINS:-https://tulus.tech,https://admin.tulus.tech,https://tulus.space,https://admin.tulus.space,https://user.tulus.space,https://typing.tulus.space,https://score.tulus.space,https://gra.tulus.space,https://gra.tulus.tech,https://simamora.tech,https://admin.simamora.tech,http://localhost:4200,http://localhost:4201,https://localhost:4200,https://localhost:4201,https://\*.tulus.tech,https://\*.tulus.space}"
CORS_ALLOWED_ORIGINS="${CORS_ALLOWED_ORIGINS:-}"
EOF
else
  # Ensure variables exist; append missing ones without touching DB and other app envs
  for K in YOUTUBE_CLIENT_ID YOUTUBE_CLIENT_SECRET YOUTUBE_REDIRECT_URL YOUTUBE_API_KEY YOUTUBE_CHANNEL_ID YOUTUBE_ACCESS_TOKEN YOUTUBE_REFRESH_TOKEN YOUTUBE_ENABLED YOUTUBE_MODE SECRET_KEY ALLOWED_ORIGINS CORS_ALLOWED_ORIGINS; do
    # Use indirect expansion on remote, with default empty to avoid unbound
    V="${!K:-}"
    # sensible defaults when not provided
    if [ -z "$V" ]; then
      case "$K" in
        YOUTUBE_REDIRECT_URL) V="https://$DOMAIN/auth/youtube/callback";;
        YOUTUBE_ENABLED) V="true";;
        YOUTUBE_MODE) V="live";;
        ALLOWED_ORIGINS) V="https://tulus.tech,https://admin.tulus.tech,https://tulus.space,https://admin.tulus.space,https://user.tulus.space,https://typing.tulus.space,https://score.tulus.space,https://gra.tulus.space,https://gra.tulus.tech,https://simamora.tech,https://admin.simamora.tech,http://localhost:4200,http://localhost:4201,https://localhost:4200,https://localhost:4201,https://\*.tulus.tech,https://\*.tulus.space";;
      esac
    fi
    if ! sudo grep -q "^$K=" "/etc/opt/$SERVICE_NAME.env"; then
      # Append as KEY="VALUE"
      echo "$K=\"$V\"" | sudo tee -a "/etc/opt/$SERVICE_NAME.env" >/dev/null
    fi
  done
fi
sudo chmod 600 /etc/opt/$SERVICE_NAME.env || true
sudo mkdir -p /etc/systemd/system/$SERVICE_NAME.service.d
cat <<EOF | sudo tee /etc/systemd/system/$SERVICE_NAME.service.d/override.conf >/dev/null
[Service]
EnvironmentFile=/etc/opt/$SERVICE_NAME.env
EOF

# Systemd
sudo systemctl daemon-reload
sudo systemctl enable $SERVICE_NAME || true
sudo systemctl restart $SERVICE_NAME

# Nginx + Certbot
if ! command -v nginx >/dev/null 2>&1; then sudo apt-get update && sudo apt-get install -y nginx; fi
if ! command -v certbot >/dev/null 2>&1; then sudo apt-get update && sudo apt-get install -y certbot python3-certbot-nginx; fi
sudo mkdir -p /etc/nginx/sites-available /etc/nginx/sites-enabled /var/www/certbot

# Phase 1: minimal HTTP config for ACME challenge (template + sed)
cat > /tmp/ng-http.tmpl <<'EOHTTP'
server {
  listen 80;
  server_name __DOMAIN__;
  location /.well-known/acme-challenge/ { root /var/www/certbot; }
  # Optional: proxy HTTP to app so service is reachable pre-cert
  location / {
    proxy_pass http://127.0.0.1:__APP_PORT__;
  # Basic proxy without extra headers to avoid shell var expansion issues
  }
}
EOHTTP
sed -e "s/__DOMAIN__/$DOMAIN/g" -e "s/__APP_PORT__/$APP_PORT/g" /tmp/ng-http.tmpl | sudo tee /etc/nginx/sites-available/$DOMAIN >/dev/null
sudo ln -sf /etc/nginx/sites-available/$DOMAIN /etc/nginx/sites-enabled/$DOMAIN
sudo nginx -t
# Reload a running master if present (may have been started outside systemd)
if pgrep -x nginx >/dev/null 2>&1; then
  sudo nginx -s reload || true
else
  # Fall back to systemd management
  sudo systemctl reload nginx || sudo systemctl start nginx || sudo systemctl restart nginx || true
fi

# Obtain/renew cert without trying to restart nginx (use webroot)
sudo certbot certonly --webroot -w /var/www/certbot -d $DOMAIN --non-interactive --agree-tos -m "lamboktulus1379@gmail.com" || true

# Phase 2: apply final TLS reverse proxy config (only if cert exists)
CERT_FULL="/etc/letsencrypt/live/$DOMAIN/fullchain.pem"
CERT_KEY="/etc/letsencrypt/live/$DOMAIN/privkey.pem"
if [ -f $CERT_FULL ] && [ -f $CERT_KEY ]; then
  if [ -f "/opt/$SERVICE_NAME/nginx-$DOMAIN.conf" ]; then
    sudo mv "/opt/$SERVICE_NAME/nginx-$DOMAIN.conf" "/etc/nginx/sites-available/$DOMAIN"
    sudo ln -sf "/etc/nginx/sites-available/$DOMAIN" "/etc/nginx/sites-enabled/$DOMAIN"
    if sudo nginx -t; then
      if pgrep -x nginx >/dev/null 2>&1; then
        sudo nginx -s reload || sudo systemctl reload nginx || true
      else
        sudo systemctl reload nginx || sudo systemctl start nginx || sudo systemctl restart nginx || true
      fi
    fi
  fi
else
  echo "WARN: TLS cert for $DOMAIN not found yet. Keeping HTTP config; try running certbot again later."
fi

echo "==> Health checks"
sleep 2
curl -fsS http://127.0.0.1:$APP_PORT/ >/dev/null && echo "Local app OK"
curl -fsS https://$DOMAIN/ -m 10 -k >/dev/null && echo "Public HTTPS OK"
EOSSH

echo "==> Deployment complete!"
echo "Service should be available at:"
echo "  https://$DOMAIN/"
echo ""
echo "To restart service: sudo systemctl restart $SERVICE_NAME"
echo "To check logs: sudo journalctl -u $SERVICE_NAME -f"
echo "To edit env: sudo nano /etc/opt/$SERVICE_NAME.env"

echo "==> Done. Visit: https://${DOMAIN}"

echo "Tips:\n- To override VM details: VM_HOST=... VM_USER=... VM_SSH_KEY=... DOMAIN=gra.tulus.space APP_PORT=${APP_PORT} bash deploy/azure/deploy_gra_vm.sh"
