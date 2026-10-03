#!/usr/bin/env bash
#
# uninstall.sh - Remove mikrotik-controller from a Ubuntu/Debian system.
#
#   sudo ./deploy/uninstall.sh            # keep the PostgreSQL data
#   sudo ./deploy/uninstall.sh --purge    # also drop the DB role/database
#
set -euo pipefail

APP_NAME="mikrotik-controller"
SYSTEM_USER="${APP_NAME%%-*}"   # -> "mikrotik"
ENV_DST="/etc/default/${APP_NAME}"
SVC_DST="/etc/systemd/system/${APP_NAME}.service"
APP_DIR="/opt/${APP_NAME}"

log() { echo "[uninstall] $*"; }

[[ "$(id -u)" -eq 0 ]] || { echo "[uninstall][ERROR] Must be run as root." >&2; exit 1; }

if systemctl is-active --quiet "${APP_NAME}" 2>/dev/null; then
    log "Stopping ${APP_NAME}..."
    systemctl stop "${APP_NAME}" || true
fi
systemctl disable -q "${APP_NAME}" 2>/dev/null || true
rm -f "$SVC_DST"
systemctl daemon-reload || true
log "Removed systemd unit."

rm -f "$ENV_DST"
log "Removed env file."

if [[ -d "$APP_DIR" ]]; then
    log "Removing $APP_DIR ..."
    rm -rf "$APP_DIR"
else
    log "$APP_DIR not present - nothing to remove."
fi

# Optionally purge PostgreSQL artifacts
if [[ "${1:-}" == "--purge" ]]; then
    if command -v psql >/dev/null 2>&1; then
        log "Dropping database 'pisowifi' and role 'pisowifi'..."
        PSQL_BIN="$(dirname "$(command -v psql)")"
        sudo -u postgres "${PSQL_BIN}/psql" -tAc "DROP DATABASE IF EXISTS pisowifi;" 2>/dev/null || true
        sudo -u postgres "${PSQL_BIN}/psql" -tAc "DROP ROLE IF EXISTS pisowifi;" 2>/dev/null || true
    else
        log "psql not found; skipping database purge."
    fi
else
    log "Kept PostgreSQL role/database. Re-run with --purge to remove them."
fi

id "${SYSTEM_USER}" >/dev/null 2>&1 && { log "Removing system user '${SYSTEM_USER}'..."; userdel --system "${SYSTEM_USER}" 2>/dev/null || true; } || true

log "Uninstall complete."
