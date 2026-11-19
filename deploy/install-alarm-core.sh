#!/bin/bash
# Installs the alarm-core binary as a systemd service on Debian 13.
# Idempotent: safe to re-run (won't overwrite an existing config).
#
# Build the binary first, from the repo root (needs Go; setup.sh can do this for you):
#   go build -o deploy/alarm-core ./cmd/alarm-core
# Building on another machine: GOOS=linux GOARCH=amd64 go build -o deploy/alarm-core ./cmd/alarm-core
#
# Usage (run on the target host, from this directory):
#   ./install-alarm-core.sh
set -euo pipefail

if [ "$(id -u)" -ne 0 ]; then
    echo "Run as root." >&2
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_DIR="/opt/alarm-core"
BIN_SRC="${SCRIPT_DIR}/alarm-core"

if [ ! -f "$BIN_SRC" ]; then
    echo "Error: binary not found at ${BIN_SRC}." >&2
    echo "The binary is not part of the repository. Build it first (from the repo root, needs Go):" >&2
    echo "  go build -o deploy/alarm-core ./cmd/alarm-core" >&2
    echo "(setup.sh can do this for you; see 'Voraussetzungen und Build' in the README)" >&2
    exit 1
fi

echo "[1/4] Creating service user ..."
id alarm-core &>/dev/null || useradd --system --no-create-home --shell /usr/sbin/nologin alarm-core
# If Asterisk is installed, join its group so alarm-core can write TTS
# output / uploaded announcements into its sounds directory.
getent group asterisk >/dev/null 2>&1 && usermod -aG asterisk alarm-core
if [ -d /var/lib/asterisk/sounds/custom ]; then
    chmod g+w /var/lib/asterisk/sounds/custom
fi

echo "[2/4] Installing binary ..."
mkdir -p "${INSTALL_DIR}/bin"
install -m 755 -o alarm-core -g alarm-core "$BIN_SRC" "${INSTALL_DIR}/bin/alarm-core"

echo "[3/4] Installing config ..."
mkdir -p /etc/alarm-core
if [ ! -f /etc/alarm-core/alarm-core.env ]; then
    cp "${SCRIPT_DIR}/alarm-core.env.example" /etc/alarm-core/alarm-core.env
    chmod 640 /etc/alarm-core/alarm-core.env
    chown root:alarm-core /etc/alarm-core/alarm-core.env
    echo "  Created /etc/alarm-core/alarm-core.env from template -> fill in secrets before starting."
else
    echo "  /etc/alarm-core/alarm-core.env already exists, leaving untouched."
fi

echo "[4/4] Installing systemd unit ..."
cp "${SCRIPT_DIR}/alarm-core.service" /etc/systemd/system/alarm-core.service
systemctl daemon-reload
systemctl enable alarm-core >/dev/null

echo "Done. Edit /etc/alarm-core/alarm-core.env, then: systemctl start alarm-core"
