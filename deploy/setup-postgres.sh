#!/usr/bin/env bash
#
# setup-postgres.sh
#
# Installs PostgreSQL (apt) and provisions the `pisowifi` role + database
# that mikrotik-controller expects. Reads connection defaults from
# /etc/default/mikrotik-controller if present, otherwise generates a strong
# password and writes it back there.
#
# Safe to re-run (idempotent).
#
set -euo pipefail

ENV_FILE="/etc/default/mikrotik-controller"

log() { echo "[setup-postgres] $*"; }
err() { echo "[setup-postgres][ERROR] $*" >&2; exit 1; }

[[ "$(id -u)" -eq 0 ]] || err "Must be run as root (use sudo)."

# --- Read existing env values if the file exists -----------------------------
DB_USER="pisowifi"
DB_NAME="pisowifi"
DB_HOST="localhost"
DB_PORT="5432"
DB_PASS=""
if [[ -f "$ENV_FILE" ]]; then
    # shellcheck disable=SC1090
    set -a; . "$ENV_FILE"; set +a
    DB_USER="${PISOWIFI_DB_USER:-$DB_USER}"
    DB_NAME="${PISOWIFI_DB_NAME:-$DB_NAME}"
    DB_HOST="${PISOWIFI_DB_HOST:-$DB_HOST}"
    DB_PORT="${PISOWIFI_DB_PORT:-$DB_PORT}"
    DB_PASS="${PISOWIFI_DB_PASSWORD:-}"
fi

# Generate a strong password if none configured yet
if [[ -z "$DB_PASS" ]]; then
    DB_PASS="$(tr -dc 'A-Za-z0-9!@#$%^&*()-_=+' </dev/urandom | head -c 24)"
    log "Generated strong DB password for user '${DB_USER}'."
fi

# --- Install PostgreSQL if missing -------------------------------------------
if ! command -v psql >/dev/null 2>&1; then
    log "Installing PostgreSQL via apt..."
    apt-get update -y
    DEBIAN_FRONTEND=noninteractive apt-get install -y postgresql postgresql-contrib
else
    log "PostgreSQL already installed."
fi

# Ensure the cluster is running
PG_BIN="$(dirname "$(command -v pg_lsclusters 2>/dev/null)" 2>/dev/null || true)"
if systemctl list-unit-files postgresql.service >/dev/null 2>&1; then
    systemctl enable postgresql || true
    systemctl start postgresql || systemctl restart postgresql || true
fi

log "PostgreSQL service status:"
systemctl is-active postgresql || true

# --- Provision role + database (use the local socket, trust/peer auth) -------
PG_SUDO_USER="postgres"
run_psql() {
    sudo -u "$PG_SUDO_USER" psql -p "${DB_PORT}" -tAc "$1" 2>&1 || true
}

# Create role (idempotent)
if [[ -z "$(run_psql "SELECT 1 FROM pg_roles WHERE rolname='${DB_USER}'")" ]]; then
    log "Creating role '${DB_USER}'..."
    sudo -u "$PG_SUDO_USER" psql -p "${DB_PORT}" -c "CREATE USER ${DB_USER} WITH PASSWORD '${DB_PASS}';" >/dev/null
else
    log "Role '${DB_USER}' exists; updating password..."
    sudo -u "$PG_SUDO_USER" psql -p "${DB_PORT}" -c "ALTER USER ${DB_USER} WITH PASSWORD '${DB_PASS}';" >/dev/null
fi

# Create database (idempotent)
if [[ -z "$(run_psql "SELECT 1 FROM pg_database WHERE datname='${DB_NAME}'")" ]]; then
    log "Creating database '${DB_NAME}'..."
    sudo -u "$PG_SUDO_USER" psql -p "${DB_PORT}" -c "CREATE DATABASE ${DB_NAME} OWNER ${DB_USER};" >/dev/null
else
    log "Database '${DB_NAME}' already exists."
fi

# Allow the app to create extensions if ever needed by pgcrypto etc.
sudo -u "$PG_SUDO_USER" psql -p "${DB_PORT}" -d "${DB_NAME}" -c "GRANT ALL PRIVILEGES ON DATABASE ${DB_NAME} TO ${DB_USER};" >/dev/null

# --- Persist password into the env file so the service can use it -----------
if [[ -f "$ENV_FILE" ]]; then
    if grep -q '^PISOWIFI_DB_PASSWORD=' "$ENV_FILE"; then
        # Update existing line (preserve quoting style)
        if grep -q "^PISOWIFI_DB_PASSWORD=.*#"; then
            sed -i -E "s|^PISOWIFI_DB_PASSWORD=.*|PISOWIFI_DB_PASSWORD='${DB_PASS}'|" "$ENV_FILE"
        else
            sed -i -E "s|^PISOWIFI_DB_PASSWORD=.*|PISOWIFI_DB_PASSWORD='${DB_PASS}'|" "$ENV_FILE"
        fi
    else
        # Append
        {
            echo ""
            echo "PISOWIFI_DB_HOST=${DB_HOST}"
            echo "PISOWIFI_DB_PORT=${DB_PORT}"
            echo "PISOWIFI_DB_USER=${DB_USER}"
            echo "PISOWIFI_DB_PASSWORD='${DB_PASS}'"
            echo "PISOWIFI_DB_NAME=${DB_NAME}"
        } >> "$ENV_FILE"
    fi
else
    mkdir -p "$(dirname "$ENV_FILE")"
    cat > "$ENV_FILE" <<EOF
# PostgreSQL connection for mikrotik-controller
# (written by deploy/setup-postgres.sh)
PISOWIFI_DB_HOST=${DB_HOST}
PISOWIFI_DB_PORT=${DB_PORT}
PISOWIFI_DB_USER=${DB_USER}
PISOWIFI_DB_PASSWORD='${DB_PASS}'
PISOWIFI_DB_NAME=${DB_NAME}
EOF
    chmod 600 "$ENV_FILE"
fi

log ""
log "PostgreSQL provisioning complete."
log "  role : ${DB_USER}"
log "  db   : ${DB_NAME}"
log "  host : ${DB_HOST}"
log "  port : ${DB_PORT}"
log ""
log "Connection string for the controller:"
log "  postgres://${DB_USER}:${DB_PASS}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable"
