#!/usr/bin/env bash
set -euo pipefail

# Local macOS deploy: run Go app on HTTP:10010 and terminate TLS with Homebrew Nginx + mkcert
# Requirements: Homebrew nginx (brew install nginx), mkcert (brew install mkcert), nss (for Firefox)

APP_PORT=${APP_PORT:-10010}
DOMAIN=${DOMAIN:-gra.tulus.tech}

# Homebrew paths (adjust if Intel vs Apple Silicon)
if [[ -d "/opt/homebrew" ]]; then
  NGINX_ETC="/opt/homebrew/etc/nginx"
else
  NGINX_ETC="/usr/local/etc/nginx"
fi
SERVERS_DIR="$NGINX_ETC/servers"
CERTS_DIR="$NGINX_ETC/certs"
CONF_DST="$SERVERS_DIR/${DOMAIN}.conf"

mkdir -p "$SERVERS_DIR" "$CERTS_DIR"

# Generate local trusted certs via mkcert
if ! command -v mkcert >/dev/null 2>&1; then
  echo "mkcert not found. Install with: brew install mkcert nss" >&2
  exit 1
fi
mkcert -install >/dev/null 2>&1 || true

if [[ ! -f "$CERTS_DIR/${DOMAIN}.pem" || ! -f "$CERTS_DIR/${DOMAIN}-key.pem" ]]; then
  (cd "$CERTS_DIR" && mkcert "$DOMAIN")
fi

# Install nginx vhost
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
sed "s/gra.tulus.tech/${DOMAIN}/g" "$SCRIPT_DIR/local-nginx.conf" > "$CONF_DST"

# Replace cert paths if Apple Silicon or Intel differ
if [[ "$NGINX_ETC" == "/usr/local/etc/nginx" ]]; then
  sed -i '' "s#/opt/homebrew/etc/nginx/certs#${CERTS_DIR}#g" "$CONF_DST"
fi

# Test and restart nginx
brew services list >/dev/null 2>&1 || { echo "Homebrew not available" >&2; exit 1; }
if ! brew services list | grep -q nginx; then
  echo "Starting nginx via Homebrew..."
  brew services start nginx
else
  echo "Reloading nginx via Homebrew..."
  brew services restart nginx
fi

# Edit /etc/hosts to resolve DOMAIN to 127.0.0.1, if missing
if ! grep -q "\s${DOMAIN}$" /etc/hosts; then
  echo "Adding ${DOMAIN} to /etc/hosts -> 127.0.0.1 (requires sudo)"
  echo "127.0.0.1 ${DOMAIN}" | sudo tee -a /etc/hosts >/dev/null
fi

# Build and run the app (foreground)
echo "Building app..."
mkdir -p bin
go build -o bin/server ./

echo "Starting app on HTTP :${APP_PORT} (TLS disabled; Nginx provides HTTPS for ${DOMAIN})"
APP_PORT=${APP_PORT} TLS_ENABLED=0 bin/server
