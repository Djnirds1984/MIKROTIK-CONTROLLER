# MikroTik Controller — Installation Guide

Full instructions to install **mikrotik-controller** on **Ubuntu**, **Debian**, and **Armbian** (all Debian-based, same procedure) from a fresh operating system to a running service.

The app is a single statically-linked Go binary with all web UI templates and assets embedded at build time. The only runtime dependency is **PostgreSQL**. Once installed, it runs as a `systemd` service that starts on boot and restarts on crash.

---

## Table of Contents

1. [Requirements](#1-requirements)
2. [Check the target machine architecture](#2-check-the-target-machine-architecture)
3. [Install Go on the build machine](#3-install-go-on-the-build-machine)
4. [Build the binary](#4-build-the-binary)
5. [Copy files to the target machine](#5-copy-files-to-the-target-machine)
6. [Set up PostgreSQL](#6-set-up-postgresql)
7. [Install the service](#7-install-the-service)
8. [Verify the installation](#8-verify-the-installation)
9. [Configuration](#9-configuration)
10. [Firewall (UFW)](#10-firewall-ufw)
11. [Updating to a new version](#11-updating-to-a-new-version)
12. [Uninstalling](#12-uninstalling)
13. [Troubleshooting](#13-troubleshooting)

---

## 1. Requirements

| Component | Minimum | Notes |
|-----------|---------|-------|
| OS | Ubuntu 22.04+, Debian 12, Armbian (bookworm/trixie base) | amd64, arm64, or armv7 |
| Database | PostgreSQL 12+ | Installed automatically by the setup script |
| Network | Port `8080` (web UI) | PostgreSQL stays on localhost only |
| Go (build machine only) | 1.26+ | Not needed on the target if you copy a pre-built binary |

---

## 2. Check the target machine architecture

SSH into the machine you want to install on (or open a terminal on it):

```bash
uname -m
```

| Output | Binary to build/use | Typical hardware |
|--------|---------------------|------------------|
| `x86_64` | `linux-amd64` | Mini PCs, servers, VMs |
| `aarch64` | `linux-arm64` | Raspberry Pi 4/5, Orange Pi, Banana Pi, Rock Pi (64-bit Armbian) |
| `armv7l` | `linux-arm` | Older / 32-bit Armbian boards |

---

## 3. Install Go on the build machine

You need Go installed on whichever computer you build on (this can be the target machine itself, or your Windows/Linux PC).

**Ubuntu / Debian / Armbian:**

```bash
sudo apt update
sudo apt install -y golang
```

Or download the latest version from [go.dev/dl](https://go.dev/dl/):

```bash
wget https://go.dev/dl/go1.26.4.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.26.4.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

**Windows:** download the installer from [go.dev/dl](https://go.dev/dl/) and run it.

Verify:

```bash
go version
```

---

## 4. Build the binary

Get the source code:

```bash
git clone https://github.com/Djnirds1984/MIKROTIK-CONTROLLER
cd MIKROTIK-CONTROLLER
```

### Option A — Build all Linux binaries (recommended)

```bash
make release
```

This produces three binaries in `build/`:

```
build/mikrotik-controller-linux-amd64
build/mikrotik-controller-linux-arm64
build/mikrotik-controller-linux-arm
```

### Option B — Build for one target only

```bash
make linux-amd64     # x86_64
make linux-arm64     # aarch64 (most Armbian boards)
make linux-arm       # armv7l (32-bit)
```

### Option C — Cross-compile from Windows (PowerShell)

```powershell
$env:GOOS="linux"; $env:GOARCH="arm64"; $env:CGO_ENABLED=0
go build -trimpath -o build/mikrotik-controller-linux-arm64 .
```

Repeat with `GOARCH="amd64"` or `GOARCH="arm"` for the other targets.

> **Note:** the binary is built with `CGO_ENABLED=0` (pure Go, static), so a binary built on any machine runs on any Ubuntu/Debian/Armbian of the same CPU architecture — no glibc/musl issues.

---

## 5. Copy files to the target machine

Copy the repo (or at least the `deploy/` folder and the built binary) to the target:

```bash
scp -r . user@<machine-ip>
```

Or clone directly on the target machine and just copy the binary over:

```bash
# on the target:
git clone https://github.com/Djnirds1984/MIKROTIK-CONTROLLER

# on the build machine:
scp build/mikrotik-controller-linux-arm64 user@<machine-ip>:~/MIKROTIK-CONTROLLER/build/
```

---

## 6. Set up PostgreSQL

On the target machine, from the repo directory:

```bash
cd MIKROTIK-CONTROLLER
sudo ./deploy/setup-postgres.sh
```

This script is **idempotent** (safe to re-run) and:

- Installs `postgresql` via `apt` if it's not already installed
- Enables and starts the PostgreSQL service
- Creates the `pisowifi` database user with a strong random password
- Creates the `pisowifi` database owned by that user
- Writes the credentials to `/etc/default/mikrotik-controller`

> **No manual SQL is needed.** The app creates its own schema (auto-migrations) on first startup — just point it at a fresh, empty database, which this script already did.

---

## 7. Install the service

Run the installer, passing the binary that matches your CPU architecture (see step 2):

```bash
# x86_64:
sudo ./deploy/install.sh build/mikrotik-controller-linux-amd64

# aarch64 (64-bit Armbian):
sudo ./deploy/install.sh build/mikrotik-controller-linux-arm64

# armv7l (32-bit):
sudo ./deploy/install.sh build/mikrotik-controller-linux-arm
```

> **Armbian / ARM tip:** always pass the correct ARM binary explicitly like above. Without an argument, `install.sh` looks for `build/mikrotik-controller-linux-amd64` first, or falls back to building natively on the board (which requires Go installed there).

What the installer does:

1. Stops any running instance of the service
2. Installs the binary to `/opt/mikrotik-controller/mikrotik-controller`
3. Copies the `firmware/` reference files (hotspot HTML pages, NodeMCU sketch) alongside it — handy when configuring MikroTik routers
4. Creates an unprivileged system user `mikrotik` (no login shell)
5. Writes `/etc/default/mikrotik-controller` (only if absent — your edits are preserved on reinstall)
6. Installs the `systemd` unit `/etc/systemd/system/mikrotik-controller.service`
7. Enables the service at boot and starts it now

---

## 8. Verify the installation

```bash
# service state (should say "active (running)")
sudo systemctl status mikrotik-controller

# live logs
sudo journalctl -fu mikrotik-controller
```

Open the web UI from any device on the same network:

```
http://<machine-ip>:8080/
```

Find the machine's IP with `ip a` or `hostname -I` if you're not sure.

---

## 9. Configuration

All settings live in `/etc/default/mikrotik-controller`. Edit and restart:

```bash
sudo nano /etc/default/mikrotik-controller
sudo systemctl restart mikrotik-controller
```

| Variable | Default | Description |
|----------|---------|-------------|
| `MIKROTIK_PORT` | `8080` | TCP port the web UI listens on |
| `PISOWIFI_DB_HOST` | `localhost` | PostgreSQL host |
| `PISOWIFI_DB_PORT` | `5432` | PostgreSQL port |
| `PISOWIFI_DB_USER` | `pisowifi` | Database user |
| `PISOWIFI_DB_PASSWORD` | *(auto-generated)* | Database password (written by `setup-postgres.sh`) |
| `PISOWIFI_DB_NAME` | `pisowifi` | Database name |

Useful service commands:

```bash
sudo systemctl start   mikrotik-controller
sudo systemctl stop    mikrotik-controller
sudo systemctl restart mikrotik-controller
sudo systemctl enable  mikrotik-controller   # auto-start on boot (done by installer)
sudo journalctl -u mikrotik-controller -n 50 --no-pager   # last 50 log lines
```

---

## 10. Firewall (UFW)

If UFW is enabled on the machine, allow LAN access to the UI while keeping the port closed to the internet:

```bash
sudo ufw allow from 192.168.1.0/24 to any port 8080
```

> PostgreSQL on port `5432` should **never** be exposed — the app talks to it over localhost.

---

## 11. Updating to a new version

On the build machine:

```bash
cd MIKROTIK-CONTROLLER
git pull
make release
```

Copy the new binary to the target, then re-run the installer (it stops the service, swaps the binary, and restarts — your config and database are untouched):

```bash
scp build/mikrotik-controller-linux-arm64 user@<machine-ip>:~/MIKROTIK-CONTROLLER/build/
# on the target:
sudo ./deploy/install.sh build/mikrotik-controller-linux-arm64
```

---

## 12. Uninstalling

```bash
sudo ./deploy/uninstall.sh            # removes service + binary, keeps database data
sudo ./deploy/uninstall.sh --purge    # also DROPs the database and role
```

---

## 13. Troubleshooting

| Symptom | Likely cause | Fix |
|---------|--------------|-----|
| Service shows `failed` right after start | Wrong DB password, or port 8080 already in use | `journalctl -u mikrotik-controller -n 50 --no-pager` shows the exact error. Re-run `sudo ./deploy/setup-postgres.sh` or change `MIKROTIK_PORT` |
| `exec format error` when starting | Wrong-architecture binary | Check `uname -m` on the target and install the matching `linux-amd64` / `linux-arm64` / `linux-arm` binary |
| UI unreachable from other devices | Firewall or wrong IP | `sudo systemctl status mikrotik-controller`, then `curl http://localhost:8080/` on the target itself; open the port in UFW (step 10) |
| Port 8080 already in use | Another web server (nginx, etc.) | Change `MIKROTIK_PORT` in `/etc/default/mikrotik-controller` and restart |
| PostgreSQL not running | Service not started | `sudo systemctl start postgresql`; check cluster with `pg_lsclusters` |
| "no prebuilt binary supplied and 'go' is not installed" | Installer couldn't find a binary and the target has no Go | Pass the binary path to `install.sh` (step 7), or install Go on the target |

---

## Appendix: what gets installed where

| Path | Purpose |
|------|---------|
| `/opt/mikrotik-controller/mikrotik-controller` | The application binary |
| `/opt/mikrotik-controller/firmware/` | Hotspot HTML pages + NodeMCU sketch (reference only) |
| `/etc/default/mikrotik-controller` | Configuration (port, DB credentials) |
| `/etc/systemd/system/mikrotik-controller.service` | systemd unit (hardened: `ProtectSystem=strict`, runs as `mikrotik` user) |
| PostgreSQL role/database `pisowifi` | Application data, auto-migrated on first boot |
