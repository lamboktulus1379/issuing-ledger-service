#!/usr/bin/env bash
set -euo pipefail

# Deploy Go app behind Nginx TLS for gra.tulus.tech (or custom DOMAIN)
# Requirements on deploy host: ssh, scp, sudo privileges, Nginx installed, systemd
# Optional: certbot for automatic certificate issuance

# Config via env vars
SSH_USER="${SSH_USER:-ubuntu}"
SSH_HOST="${SSH_HOST:-}"
SSH_PORT="${SSH_PORT:-22}"
DOMAIN="${DOMAIN:-gra.tulus.tech}"
REMOTE_DIR="${REMOTE_DIR:-/opt/go-project}"
SERVICE_NAME="${SERVICE_NAME:-go-project}"
NGINX_CONF_DIR_REMOTE="${NGINX_CONF_DIR_REMOTE:-/etc/nginx/conf.d}"
CERTBOT="${CERTBOT:-0}"               # 1 to run certbot --nginx
CERTBOT_EMAIL="${CERTBOT_EMAIL:-}"    # required if CERTBOT=1

if [[ -z "${SSH_HOST}" ]]; then
  echo "Error: SSH_HOST is required (target server)" >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
BUILD_BIN="${REPO_ROOT}/server"
TMP_LOCAL_DIR="$(mktemp -d)"

cleanup() {
  rm -rf "${TMP_LOCAL_DIR}" || true
}
trap cleanup EXIT

# 1) Build the server binary
pushd "${REPO_ROOT}" >/dev/null
  echo "Building server binary..."
  go build -o server ./
popd >/dev/null

# 2) Prepare templated files (inject DOMAIN)
SERVICE_SRC="${REPO_ROOT}/deploy/systemd/go-project.service"
NGINX_SRC="${REPO_ROOT}/deploy/nginx.gra.tulus.tech.conf"
SERVICE_DST="${TMP_LOCAL_DIR}/go-project.service"
NGINX_DST="${TMP_LOCAL_DIR}/nginx.conf"

# Replace gra.tulus.tech with provided DOMAIN for convenience
sed "s/gra.tulus.tech/${DOMAIN}/g" "${SERVICE_SRC}" > "${SERVICE_DST}"
sed "s/gra.tulus.tech/${DOMAIN}/g" "${NGINX_SRC}" > "${NGINX_DST}"

# 3) Ship artifacts to remote
REMOTE_TMP="/tmp/go-project-deploy-$(date +%s)"
ssh -p "${SSH_PORT}" "${SSH_USER}@${SSH_HOST}" "mkdir -p ${REMOTE_TMP}"

scp -P "${SSH_PORT}" "${BUILD_BIN}" "${SSH_USER}@${SSH_HOST}:${REMOTE_TMP}/server"
scp -P "${SSH_PORT}" "${SERVICE_DST}" "${SSH_USER}@${SSH_HOST}:${REMOTE_TMP}/go-project.service"
scp -P "${SSH_PORT}" "${NGINX_DST}" "${SSH_USER}@${SSH_HOST}:${REMOTE_TMP}/nginx.conf"

# 4) Install on remote (needs sudo)
ssh -t -p "${SSH_PORT}" "${SSH_USER}@${SSH_HOST}" bash -lc "'
  set -euo pipefail
  echo Installing to ${REMOTE_DIR}
  sudo mkdir -p ${REMOTE_DIR}
  sudo mv ${REMOTE_TMP}/server ${REMOTE_DIR}/server
  sudo chmod +x ${REMOTE_DIR}/server

  # systemd unit
  sudo mv ${REMOTE_TMP}/go-project.service /etc/systemd/system/${SERVICE_NAME}.service
  sudo systemctl daemon-reload
  sudo systemctl enable ${SERVICE_NAME}
  sudo systemctl restart ${SERVICE_NAME}
  sudo systemctl status --no-pager ${SERVICE_NAME} || true

  # nginx vhost
  sudo mv ${REMOTE_TMP}/nginx.conf ${NGINX_CONF_DIR_REMOTE}/${DOMAIN}.conf
'
"

# 5) Handle certificates and reload nginx
if [[ "${CERTBOT}" == "1" ]]; then
  if [[ -z "${CERTBOT_EMAIL}" ]]; then
    echo "CERTBOT=1 set but CERTBOT_EMAIL is empty; skipping certbot." >&2
  else
    echo "Requesting/renewing certificate via certbot for ${DOMAIN}..."
    ssh -t -p "${SSH_PORT}" "${SSH_USER}@${SSH_HOST}" sudo certbot --nginx -d "${DOMAIN}" --non-interactive --agree-tos -m "${CERTBOT_EMAIL}" || true
  fi
fi

# Now test and reload nginx (works whether certs existed prior or certbot just installed them)
ssh -t -p "${SSH_PORT}" "${SSH_USER}@${SSH_HOST}" bash -lc "'
  set -e
  sudo nginx -t
  sudo systemctl reload nginx
'"

echo "Deployment complete. Visit: https://${DOMAIN}/"
