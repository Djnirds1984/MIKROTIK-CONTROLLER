#!/usr/bin/env bash
#
# install.sh - Install mikrotik-controller as a systemd service on Ubuntu/Debian.
#
# Usage:
#   sudo ./deploy/install.sh            # build natively on the target mini PC
#   sudo ./deploy/install.sh /path/to/mikrotik-controller-linux-amd64   # use prebuilt
#
set -euo pipefail

APP_NAME="mikrotik-controller"
APP_DIR="/opt/${APP_NAME}"
ENV_SRC="$(dirname "$0")/${APP_NAME}.env"
ENV_DST="/etc/default/${APP_NAME}"
SVC_SRC="$(dirname "$0")/${APP_NAME}.service"
SVC_DST="/etc/systemd/system/${APP_NAME}.service"
SYSTEM_USER="${APP_NAME%%-*}"   # -> "mikrotik"

log() { echo "[install] $*"; }
err() { echo "[install][ERROR] $*" >&2; exit 1; }

[[ "$(id -u)" -eq 0 ]] || err "Must be run as root (use sudo)."

# --- Resolve the binary to install ------------------------------------------
BINARY="${1:-}"
if [[ -z "$BINARY" ]]; then
    CANDIDATE="build/${APP_NAME}-linux-amd64"
    if [[ -x "$CANDIDATE" ]]; then
        BINARY="$CANDIDATE"
    elif [[ -x "build/${APP_NAME}" ]]; then
        BINARY="build/${APP_NAME}"
    fi
fi

if [[ -z "$BINARY" || ! -x "$BINARY" ]]; then
    # Fall back to building natively on the target.
    if command -v go >/dev/null 2>&1; then
        log "Building ${APP_NAME} from source (native Linux build)..."
        mkdir -p build
        CGO_ENABLED=0 go build -trimpath -o "build/${APP_NAME}-linux-amd64" .
        BINARY="build/${APP_NAME}-linux-amd64"
    else
        err "No prebuilt binary supplied and 'go' is not installed.\n      Run 'make linux-amd64' on a dev machine and pass the path, e.g.:\n      sudo ./deploy/install.sh build/mikrotik-controller-linux-amd64"
    fi
fi

log "Installing binary: $BINARY -> ${APP_DIR}/${APP_NAME}"
mkdir -p "$APP_DIR"

# Stop a previously running instance if present
if systemctl is-active --quiet "${APP_NAME}" 2>/dev/null; then
    log "Stopping running ${APP_NAME} service..."
    systemctl stop "${APP_NAME}" || true
fi

install -m 0755 -o root -g root "$BINARY" "${APP_DIR}/${APP_NAME}"

# Reference firmware files (hotspot HTML, NodeMCU sketch) - not used at runtime,
# but handy to have on the same box when configuring MikroTik routers.
if [[ -d "firmware" ]]; then
    log "Copying firmware reference files..."
    rm -rf "${APP_DIR}/firmware"
    cp -r firmware "${APP_DIR}/firmware"
    chown -R "${SYSTEM_USER}:${SYSTEM_USER}" "${APP_DIR}/firmware" 2>/dev/null || true
fi

# --- System user -------------------------------------------------------------
if ! id "${SYSTEM_USER}" >/dev/null 2>&1; then
    log "Creating system user '${SYSTEM_USER}'..."
    useradd --system --no-create-home --shell /usr/sbin/nologin "${SYSTEM_USER}"
fi
chown -R "${SYSTEM_USER}:${SYSTEM_USER}" "$APP_DIR"

# --- Environment file --------------------------------------------------------
if [[ ! -f "$ENV_DST" ]]; then
    if [[ -f "$ENV_SRC" ]]; then
        install -m 0640 -o root -g "${SYSTEM_USER}" "$ENV_SRC" "$ENV_DST"
        log "Wrote $ENV_DST"
    else
        err "Env template not found at $ENV_SRC"
    fi
else
    log "$ENV_DST already exists - leaving it untouched (edit it manually to change settings)."
fi

# --- systemd unit ------------------------------------------------------------
install -m 0644 -o root -g root "$SVC_SRC" "$SVC_DST"
log "Wrote $SVC_DST"

systemctl daemon-reload || true
systemctl enable -q "${APP_NAME}" 2>/dev/null || true
log "Starting ${APP_NAME}..."
systemctl restart "${APP_NAME}"

sleep 2
if systemctl is-active --quiet "${APP_NAME}"; then
    PORT="$(. "$ENV_DST" 2>/dev/null && echo "${MIKROTIK_PORT:-8080}")"
    log ""
    log "✅ ${APP_NAME} is running."
    log "   UI      : http://<mini-pc-ip>:${PORT}/"
    log "   Logs    : journalctl -fu ${APP_NAME}"
    log ""
    log "   Next steps:"
    log "   1. ./deploy/setup-postgres.sh   (if you haven't configured PostgreSQL yet)"
    log "   2. Edit /etc/default/${APP_NAME} for non-default DB credentials, then"
    log "      'sudo systemctl restart ${APP_NAME}'"
else
    log ""
    err "Service failed to stay running. Inspect with:\n      journalctl -u ${APP_NAME} -n 50 --no-pager"
fi
