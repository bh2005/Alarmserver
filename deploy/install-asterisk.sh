#!/bin/bash
# Builds Asterisk from source on Debian 13 (the "asterisk" package no
# longer exists in Debian's repos — verified empirically against a real
# Debian 13 container; only sound/prompt add-on packages remain there),
# configures the AMI user that alarm-core uses for Originate calls, and
# adds a dialplan context + a commented PJSIP trunk template (chan_sip is
# removed in modern Asterisk; PJSIP is the supported channel driver).
# Idempotent: safe to re-run — re-running skips the build if Asterisk is
# already installed, and never overwrites existing config customizations.
#
# NOTE: the source build (configure + make) commonly takes 5-20 minutes
# depending on CPU; this is normal, not a hang.
#
# Usage:
#   ASTERISK_AMI_PASSWORD=secret ./install-asterisk.sh
#
# Optional env vars:
#   ASTERISK_VERSION           (default: 21 — major version branch to build)
#   ASTERISK_AMI_USER          (default: alarmcore)
#   ASTERISK_AMI_ALLOWED_FROM  (default: 127.0.0.1 — set to alarm-core's IP
#                                if it runs on a different host)
#   ASTERISK_ANNOUNCEMENT      (default: alarm-announcement)
#   TTS_ENABLED                (default: false — installs espeak-ng + sox so
#                                alarm-core can speak the alert text instead
#                                of playing a fixed announcement)
set -euo pipefail

if [ "$(id -u)" -ne 0 ]; then
    echo "Run as root." >&2
    exit 1
fi

AMI_USER="${ASTERISK_AMI_USER:-alarmcore}"
AMI_PASSWORD="${ASTERISK_AMI_PASSWORD:?Set ASTERISK_AMI_PASSWORD before running this script}"
AMI_ALLOWED_FROM="${ASTERISK_AMI_ALLOWED_FROM:-127.0.0.1}"
ANNOUNCEMENT_NAME="${ASTERISK_ANNOUNCEMENT:-alarm-announcement}"
TTS_ENABLED="${TTS_ENABLED:-false}"
ASTERISK_VERSION="${ASTERISK_VERSION:-21}"
BUILD_DIR="/usr/src/asterisk-build"

MANAGER_CONF="/etc/asterisk/manager.conf"
EXTENSIONS_CONF="/etc/asterisk/extensions.conf"
PJSIP_CONF="/etc/asterisk/pjsip.conf"

echo "[1/6] Installing build dependencies ..."
apt-get update -qq
DEBIAN_FRONTEND=noninteractive apt-get install -y \
    build-essential wget tar pkg-config \
    libssl-dev libjansson-dev libsqlite3-dev libxml2-dev libxml2-utils \
    uuid-dev libsrtp2-dev libedit-dev libncurses-dev libnewt-dev \
    libcurl4-openssl-dev unixodbc-dev libgsm1-dev libsystemd-dev >/dev/null
if [ "$TTS_ENABLED" = "true" ]; then
    echo "  Installing espeak-ng + sox for offline TTS ..."
    DEBIAN_FRONTEND=noninteractive apt-get install -y espeak-ng sox >/dev/null
fi

echo "[2/6] Creating asterisk system user ..."
id asterisk &>/dev/null || useradd --system --home-dir /var/lib/asterisk --shell /usr/sbin/nologin asterisk

if [ -x /usr/sbin/asterisk ]; then
    echo "[3/6] Asterisk already built and installed, skipping source build."
    SRC_DIR=$(find "$BUILD_DIR" -maxdepth 1 -type d -name "asterisk-${ASTERISK_VERSION}.*" 2>/dev/null | head -1)
else
    echo "[3/6] Downloading and building Asterisk ${ASTERISK_VERSION} from source ..."
    mkdir -p "$BUILD_DIR"
    cd "$BUILD_DIR"
    wget -q "https://downloads.asterisk.org/pub/telephony/asterisk/asterisk-${ASTERISK_VERSION}-current.tar.gz"
    tar xzf "asterisk-${ASTERISK_VERSION}-current.tar.gz"
    SRC_DIR=$(find "$BUILD_DIR" -maxdepth 1 -type d -name "asterisk-${ASTERISK_VERSION}.*" | head -1)
    cd "$SRC_DIR"
    ./configure --with-jansson-bundled
    make -j"$(nproc)"
    make install
    if [ ! -f /etc/asterisk/asterisk.conf ]; then
        make samples
    fi
    make config
fi

echo "[4/6] Installing systemd unit ..."
if [ -n "$SRC_DIR" ] && [ -f "${SRC_DIR}/contrib/systemd/asterisk.service" ]; then
    cp "${SRC_DIR}/contrib/systemd/asterisk.service" /etc/systemd/system/asterisk.service
fi

echo "[5/6] Configuring AMI, dialplan and PJSIP trunk template ..."
if ! grep -q "^\[${AMI_USER}\]" "$MANAGER_CONF" 2>/dev/null; then
    cat >> "$MANAGER_CONF" <<-EOF

[${AMI_USER}]
secret = ${AMI_PASSWORD}
deny = 0.0.0.0/0.0.0.0
permit = ${AMI_ALLOWED_FROM}/255.255.255.255
read = system,call,originate
write = system,call,originate
EOF
    sed -i 's/^enabled[[:space:]]*=[[:space:]]*no/enabled = yes/' "$MANAGER_CONF" || true
else
    echo "  AMI user '${AMI_USER}' already configured, skipping."
fi

if ! grep -q "^\[alarm-outbound\]" "$EXTENSIONS_CONF" 2>/dev/null; then
    cat >> "$EXTENSIONS_CONF" <<-EOF

[alarm-outbound]
; Entered by alarm-core's Originate calls (see ASTERISK_CHANNEL_TEMPLATE).
exten => _X.,1,NoOp(Alarm-Core outbound call to \${EXTEN})
 same => n,Answer()
 same => n,Playback(${ANNOUNCEMENT_NAME})
 same => n,Hangup()
EOF
else
    echo "  [alarm-outbound] context already present, skipping."
fi

if ! grep -q "alarm-core outbound trunk" "$PJSIP_CONF" 2>/dev/null; then
    cat >> "$PJSIP_CONF" <<-'EOF'

; --- alarm-core outbound trunk (FILL IN your provider's details, then
;     uncomment and run: asterisk -rx "pjsip reload") ---
; [out-trunk-auth]
; type = auth
; auth_type = userpass
; username = YOUR_SIP_USERNAME
; password = YOUR_SIP_PASSWORD
;
; [out-trunk-aor]
; type = aor
; contact = sip:YOUR_PROVIDER_HOST
;
; [out-trunk]
; type = endpoint
; context = alarm-outbound
; outbound_auth = out-trunk-auth
; aors = out-trunk-aor
; allow = ulaw,alaw
;
; [out-trunk]
; type = registration
; outbound_auth = out-trunk-auth
; server_uri = sip:YOUR_PROVIDER_HOST
; client_uri = sip:YOUR_SIP_USERNAME@YOUR_PROVIDER_HOST
EOF
    echo "  Template added -> edit ${PJSIP_CONF} with your trunk provider's details."
else
    echo "  Trunk template already present, skipping."
fi

echo "[6/6] Setting ownership and starting Asterisk ..."
mkdir -p /var/lib/asterisk/sounds/custom /var/run/asterisk
for d in /etc/asterisk /var/lib/asterisk /var/log/asterisk /var/spool/asterisk /var/run/asterisk; do
    chown -R asterisk:asterisk "$d" 2>/dev/null || true
done
# Group-writable so alarm-core (member of group "asterisk") can drop TTS
# output and uploaded announcements here without running as this user.
chmod g+w /var/lib/asterisk/sounds/custom
# alarm-core may already exist if install-alarm-core.sh ran first.
id alarm-core &>/dev/null && usermod -aG asterisk alarm-core || true
systemctl daemon-reload
systemctl enable asterisk >/dev/null
systemctl restart asterisk

cat <<-EOF

Done. AMI listening on 127.0.0.1:5038, user '${AMI_USER}'.

Manual steps still required:
  1. Place ${ANNOUNCEMENT_NAME}.wav (8kHz mono) under /var/lib/asterisk/sounds/custom/
     (skip this if TTS_ENABLED=true — the alert text is spoken instead)
  2. Fill in the outbound trunk in ${PJSIP_CONF}, then: asterisk -rx "pjsip reload"
  3. Set ASTERISK_CHANNEL_TEMPLATE in alarm-core's config to match your trunk,
     e.g. PJSIP/{number}@out-trunk
EOF
