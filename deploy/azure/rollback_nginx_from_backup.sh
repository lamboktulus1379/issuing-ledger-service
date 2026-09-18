#!/usr/bin/env bash

set -euo pipefail

# Roll back remote VM Nginx configuration from a local snapshot
# - Supports restoring from a local directory snapshot or a .tgz archive
# - Creates a remote backup (/var/backups/nginx-backup-<ts>.tgz) before applying changes
# - Validates with `nginx -t` and reloads; auto-reverts if validation fails
#
# Defaults are tailored for the current project/VM; override via flags or env vars.
#
# Usage examples:
#   ./rollback_nginx_from_backup.sh --backup ./nginx-backups/20250905-094603-gra-57.158.26.137 \
#       --host 57.158.26.137 --user gra --key /path/to/gra-ssh.pem --yes
#
#   ./rollback_nginx_from_backup.sh --backup ./nginx-backups/20250905-094603-gra-57.158.26.137.tgz --yes
#
#   # Use latest local backup automatically
#   ./rollback_nginx_from_backup.sh --yes

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEFAULT_BACKUP_ROOT="$SCRIPT_DIR/nginx-backups"

# Load optional local overrides (not committed; see .gitignore)
if [[ -f "$SCRIPT_DIR/.env" ]]; then
  # shellcheck disable=SC1090
  source "$SCRIPT_DIR/.env"
fi

# Defaults (override with .env or flags)
# Keep conservative, repo-safe defaults. Require HOST unless provided.
HOST="${HOST:-}"
USER="${USER_NAME:-gra}"
KEY="${KEY:-$HOME/.ssh/gra-ssh.pem}"
REMOTE_NGINX_DIR="${REMOTE_NGINX_DIR:-/etc/nginx}"
BACKUP_PATH=""
DRY_RUN=0
ASSUME_YES=0

usage() {
  cat <<EOF
Rollback Nginx config on remote VM from a local backup.

Options:
  --backup PATH            Local backup path (dir or .tgz). Defaults to latest under $DEFAULT_BACKUP_ROOT
  --host HOST              Remote host (default: $HOST)
  --user USER              Remote user (default: $USER)
  --key PATH               SSH private key path (default: $KEY)
  --remote-nginx-dir PATH  Remote Nginx dir (default: $REMOTE_NGINX_DIR)
  --dry-run                Show actions without applying
  --yes                    Non-interactive; skip confirmation prompt
  -h, --help               Show help

Environment overrides:
  HOST, USER_NAME, KEY, REMOTE_NGINX_DIR

Examples:
  $0 --backup "$DEFAULT_BACKUP_ROOT/<timestamp>-gra-57.158.26.137" --yes
  $0 --backup "$DEFAULT_BACKUP_ROOT/<timestamp>-gra-57.158.26.137.tgz" --yes
  $0 --yes   # Use the most recent snapshot under $DEFAULT_BACKUP_ROOT
EOF
}

err() { echo "[ERR] $*" >&2; }
log() { echo "[INFO] $*"; }

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || { err "Missing required command: $1"; exit 1; }
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --backup)
        BACKUP_PATH="$2"; shift 2 ;;
      --host)
        HOST="$2"; shift 2 ;;
      --user)
        USER="$2"; shift 2 ;;
      --key)
        KEY="$2"; shift 2 ;;
      --remote-nginx-dir)
        REMOTE_NGINX_DIR="$2"; shift 2 ;;
      --dry-run)
        DRY_RUN=1; shift ;;
      --yes)
        ASSUME_YES=1; shift ;;
      -h|--help)
        usage; exit 0 ;;
      *)
        err "Unknown argument: $1"; usage; exit 1 ;;
    esac
  done
}

select_latest_backup() {
  [[ -d "$DEFAULT_BACKUP_ROOT" ]] || { err "Backup root not found: $DEFAULT_BACKUP_ROOT"; exit 1; }
  local latest
  latest=$(ls -1t "$DEFAULT_BACKUP_ROOT" | head -n1 || true)
  [[ -n "$latest" ]] || { err "No backups found in $DEFAULT_BACKUP_ROOT"; exit 1; }
  BACKUP_PATH="$DEFAULT_BACKUP_ROOT/$latest"
}

resolve_backup() {
  if [[ -z "${BACKUP_PATH:-}" ]]; then
    select_latest_backup
  fi
  if [[ ! -e "$BACKUP_PATH" ]]; then
    err "Backup path does not exist: $BACKUP_PATH"; exit 1
  fi
}

confirm() {
  if [[ $ASSUME_YES -eq 1 ]]; then return 0; fi
  echo "About to restore Nginx on $USER@$HOST from backup: $BACKUP_PATH"
  read -r -p "Proceed? (yes/no) " ans
  [[ "$ans" == "yes" ]] || { err "Aborted by user"; exit 1; }
}

remote() {
  ssh -i "$KEY" -o StrictHostKeyChecking=no "$USER@$HOST" "$@"
}

remote_sudo() {
  ssh -i "$KEY" -o StrictHostKeyChecking=no "$USER@$HOST" "sudo bash -lc '$*'"
}

copy_to_remote() {
  scp -i "$KEY" -o StrictHostKeyChecking=no "$@"
}

main() {
  parse_args "$@"
  require_cmd ssh; require_cmd scp; require_cmd tar
  resolve_backup
  confirm

  TS="$(date +%Y%m%d%H%M%S)"
  STAGING="/tmp/nginx-restore-$TS"
  REMOTE_BACKUP_TGZ="/var/backups/nginx-backup-$TS.tgz"

  log "Checking SSH connectivity to $USER@$HOST ..."
  remote 'echo ok' >/dev/null || { err "SSH connection failed"; exit 1; }

  log "Creating remote backup of current $REMOTE_NGINX_DIR at $REMOTE_BACKUP_TGZ ..."
  if [[ $DRY_RUN -eq 0 ]]; then
    remote_sudo "mkdir -p /var/backups && tar -C / -czf '$REMOTE_BACKUP_TGZ' '${REMOTE_NGINX_DIR#/}'"
  else
    log "DRY-RUN: would run: tar -C / -czf '$REMOTE_BACKUP_TGZ' '${REMOTE_NGINX_DIR#/}'"
  fi

  log "Preparing remote staging directory $STAGING ..."
  if [[ $DRY_RUN -eq 0 ]]; then
    remote_sudo "rm -rf '$STAGING' && mkdir -p '$STAGING'"
  else
    log "DRY-RUN: would create $STAGING"
  fi

  if [[ -d "$BACKUP_PATH" ]]; then
    log "Transferring backup directory to remote staging ..."
    if [[ $DRY_RUN -eq 0 ]]; then
      copy_to_remote -r "$BACKUP_PATH" "$USER@$HOST:$STAGING/"
    else
      log "DRY-RUN: would scp -r '$BACKUP_PATH' '$USER@$HOST:$STAGING/'"
    fi
  else
    log "Transferring backup archive to remote staging ..."
    if [[ $DRY_RUN -eq 0 ]]; then
      copy_to_remote "$BACKUP_PATH" "$USER@$HOST:$STAGING/backup.tgz"
      remote_sudo "tar -xzf '$STAGING/backup.tgz' -C '$STAGING'"
    else
      log "DRY-RUN: would scp '$BACKUP_PATH' '$USER@$HOST:$STAGING/backup.tgz' and extract"
    fi
  fi

  # Find source directory inside staging that contains nginx.conf or sites-available
  log "Locating source config directory inside staging ..."
  SRC_DIR_CMD='\nset -e
    CAND="$STAGING"
    if [ ! -f "$STAGING/nginx.conf" ] && [ ! -d "$STAGING/sites-available" ]; then
      # pick first child directory if present
      CHILD=$(ls -1 "$STAGING" | head -n1 || true)
      if [ -n "$CHILD" ] && [ -d "$STAGING/$CHILD" ]; then
        CAND="$STAGING/$CHILD"
      fi
    fi
    echo $CAND
  '
  SRC_DIR=$(ssh -i "$KEY" -o StrictHostKeyChecking=no "$USER@$HOST" "bash -lc '$SRC_DIR_CMD'")
  log "Using SRC_DIR: $SRC_DIR"

  # Copy into place on remote
  apply_cmds=$(cat <<'EOS'
set -e
SRC_DIR="$1"
REMOTE_NGINX_DIR="$2"

mkdir -p "$REMOTE_NGINX_DIR"

# Copy nginx.conf if present
if [ -f "$SRC_DIR/nginx.conf" ]; then
  cp -a "$SRC_DIR/nginx.conf" "$REMOTE_NGINX_DIR/nginx.conf"
fi

# Helper to sync directory content if exists
sync_dir() {
  local name="$1"
  if [ -d "$SRC_DIR/$name" ]; then
    mkdir -p "$REMOTE_NGINX_DIR/$name"
    cp -a "$SRC_DIR/$name/." "$REMOTE_NGINX_DIR/$name/"
  fi
}

sync_dir sites-available
sync_dir sites-enabled
sync_dir conf.d
sync_dir snippets

# Optional: Let's Encrypt renewal configs (safe; no private keys)
if [ -d "$SRC_DIR/letsencrypt-renewal" ]; then
  mkdir -p /etc/letsencrypt/renewal
  cp -a "$SRC_DIR/letsencrypt-renewal/." /etc/letsencrypt/renewal/
fi

nginx -t
EOS
  )

  log "Applying files to remote $REMOTE_NGINX_DIR ..."
  if [[ $DRY_RUN -eq 0 ]]; then
    # shellcheck disable=SC2029
    remote_sudo "bash -lc '$(printf "%q" "$apply_cmds")' _ "$SRC_DIR" "$REMOTE_NGINX_DIR" || {
      echo '[ERR] Validation failed, restoring previous backup...' >&2;
      tar -C / -xzf '$REMOTE_BACKUP_TGZ';
      nginx -t || true;
      exit 1;
    }"
  else
    log "DRY-RUN: would copy configs and run nginx -t"
  fi

  log "Reloading Nginx ..."
  if [[ $DRY_RUN -eq 0 ]]; then
    remote_sudo "systemctl reload nginx"
    log "Reloaded successfully. Rollback completed."
  else
    log "DRY-RUN: would reload nginx"
  fi

  log "Done. Remote backup at: $REMOTE_BACKUP_TGZ"
}

main "$@"
