#!/bin/bash
# Interactive setup assistant for alarm-core on Debian 13. Asks which
# components to install and which channels to enable, then runs
# install-postgres.sh / install-asterisk.sh / install-alarm-core.sh and
# writes /etc/alarm-core/alarm-core.env accordingly.
#
# Usage (run as root on the target host, from this directory):
#   ./setup.sh
set -euo pipefail

if [ "$(id -u)" -ne 0 ]; then
    echo "Run as root." >&2
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Prompts are written to stderr so they still show up even though the
# answer is captured via `VAR=$(ask ...)` (which redirects stdout).
ask() {
    local prompt="$1" default="${2:-}" answer
    if [ -n "$default" ]; then
        printf '%s [%s]: ' "$prompt" "$default" >&2
    else
        printf '%s: ' "$prompt" >&2
    fi
    read -r answer
    echo "${answer:-$default}"
}

ask_secret() {
    local prompt="$1" answer
    printf '%s: ' "$prompt" >&2
    read -r -s answer
    echo >&2
    echo "$answer"
}

ask_yesno() {
    local prompt="$1" default="${2:-y}" answer
    printf '%s [y/n] (default: %s): ' "$prompt" "$default" >&2
    read -r answer
    answer="${answer:-$default}"
    [[ "$answer" =~ ^[Yy] ]]
}

# Defaults for everything, so `set -u` doesn't choke on skipped sections.
DB_NAME="alarmcore"; DB_USER="alarmcore"; DB_PASSWORD=""
AMI_USER="alarmcore"; AMI_PASSWORD=""; AMI_ALLOWED_FROM="127.0.0.1"; ANNOUNCEMENT="alarm-announcement"
ASTERISK_HOST=""; ASTERISK_PORT="5038"; ASTERISK_CHANNEL_TEMPLATE="PJSIP/{number}"
SMSEAGLE_BASE_URL=""; SMSEAGLE_LOGIN=""; SMSEAGLE_PASSWORD=""
SMTP_HOST=""; SMTP_PORT="587"; SMTP_USERNAME=""; SMTP_PASSWORD=""; SMTP_FROM=""
TEAMS_WEBHOOK_URL=""
LISTEN_ADDR=":8080"
PROXY_URL=""; NO_PROXY_HOSTS="localhost,127.0.0.1"
MAIL_RELAY_HOST=""; MAIL_RELAY_PORT="25"; CONFIGURE_SYSTEM_RELAY=false; ADMIN_MAIL=""
TTS_ENABLED="false"; TTS_VOICE="de"; TTS_SPEED="0"; ASTERISK_SOUNDS_DIR="/var/lib/asterisk/sounds/custom"
API_KEY=""; SESSION_SECRET=""; ADMIN_USERNAME="admin"; ADMIN_PASSWORD=""

echo "=== Alarm-Core Setup Assistant ==="
echo "Installs and configures PostgreSQL, Asterisk and the alarm-core service"
echo "on this Debian 13 host. Ctrl+C aborts any time before installation starts."
echo

INSTALL_POSTGRES=false
INSTALL_ASTERISK=false
INSTALL_ALARM_CORE=false
ask_yesno "Install / configure PostgreSQL?" y && INSTALL_POSTGRES=true
ask_yesno "Install / configure Asterisk (voice channel)?" y && INSTALL_ASTERISK=true
ask_yesno "Install the alarm-core service?" y && INSTALL_ALARM_CORE=true

# --- alarm-core binary preflight --------------------------------------------
# The binary is not part of the repository. Check this first - before asking many
# questions and before anything is installed - and offer to build it from source.
BIN_PATH="${SCRIPT_DIR}/alarm-core"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
BUILD_BINARY=false
if $INSTALL_ALARM_CORE && [ ! -f "$BIN_PATH" ]; then
    echo
    echo "The alarm-core binary is not part of the repository (${BIN_PATH} not found)."
    if [ -d "${REPO_ROOT}/cmd/alarm-core" ]; then
        if command -v go >/dev/null 2>&1; then
            echo "Go toolchain found: $(go version)"
        else
            echo "Go is not installed; it would be installed with apt (golang-go)."
        fi
        if ask_yesno "Build alarm-core from source as the first installation step?" y; then
            BUILD_BINARY=true
        fi
    fi
    if ! $BUILD_BINARY; then
        echo "Cannot continue without the binary. Build it first (from the repo root, needs Go):" >&2
        echo "  go build -o deploy/alarm-core ./cmd/alarm-core" >&2
        echo "or build it on another machine (GOOS=linux GOARCH=amd64) and copy it to ${BIN_PATH}." >&2
        echo "Nothing was changed." >&2
        exit 1
    fi
fi
# --- end alarm-core binary preflight ----------------------------------------

echo
echo "--- Internet access ---"
if ask_yesno "Does this host need an HTTP/HTTPS proxy for internet access (apt, webhooks)?" n; then
    PROXY_URL=$(ask "Proxy URL (e.g. http://proxy.example.org:8080, or http://user:pass@proxy.example.org:8080)" "")
    NO_PROXY_HOSTS=$(ask "Hosts/domains to bypass the proxy (comma-separated)" "$NO_PROXY_HOSTS")
fi

echo
echo "--- Mail relay ---"
if ask_yesno "Configure a mail relay / smarthost for outgoing mail?" n; then
    MAIL_RELAY_HOST=$(ask "Mail relay host" "")
    MAIL_RELAY_PORT=$(ask "Mail relay port" "$MAIL_RELAY_PORT")
    if ask_yesno "Also relay system mail (cron jobs, e.g. the Postgres backup script) through it?" y; then
        CONFIGURE_SYSTEM_RELAY=true
        ADMIN_MAIL=$(ask "Admin email address for system mail (root's mail alias)" "")
    fi
fi

if $INSTALL_POSTGRES; then
    echo
    echo "--- PostgreSQL ---"
    DB_NAME=$(ask "Database name" "$DB_NAME")
    DB_USER=$(ask "Database user" "$DB_USER")
    DB_PASSWORD=$(ask_secret "Database password (avoid single quotes)")
fi

if $INSTALL_ASTERISK; then
    echo
    echo "--- Asterisk (AMI) ---"
    AMI_USER=$(ask "AMI username" "$AMI_USER")
    AMI_PASSWORD=$(ask_secret "AMI password")
    AMI_ALLOWED_FROM=$(ask "Allowed AMI source IP (alarm-core's IP, or 127.0.0.1 if same host)" "$AMI_ALLOWED_FROM")
    ANNOUNCEMENT=$(ask "Announcement sound name (static fallback if TTS is off or fails)" "$ANNOUNCEMENT")

    if ask_yesno "Speak the actual alert text via offline TTS (espeak-ng) instead of a fixed announcement?" y; then
        TTS_ENABLED="true"
        TTS_VOICE=$(ask "TTS voice/language (espeak-ng -v)" "$TTS_VOICE")
        TTS_SPEED=$(ask "TTS speed in words/min (0 = engine default)" "$TTS_SPEED")
    fi
fi

if $INSTALL_ALARM_CORE; then
    echo
    echo "--- alarm-core: security ---"
    if ask_yesno "Generate a random API key for /api/* automatically?" y; then
        API_KEY=$(openssl rand -hex 32)
    else
        API_KEY=$(ask_secret "API key (clients send it as Authorization: Bearer <key> or X-API-Key)")
    fi
    SESSION_SECRET=$(openssl rand -hex 32)
    ADMIN_USERNAME=$(ask "Initial UI admin username" "admin")
    ADMIN_PASSWORD=$(ask_secret "Initial UI admin password (change it after first login)")

    echo
    echo "--- alarm-core: notification channels ---"

    if ask_yesno "Enable voice channel (Asterisk)?" "$([ "$INSTALL_ASTERISK" = true ] && echo y || echo n)"; then
        ASTERISK_HOST=$(ask "Asterisk AMI host" "127.0.0.1")
        ASTERISK_PORT=$(ask "Asterisk AMI port" "$ASTERISK_PORT")
        ASTERISK_CHANNEL_TEMPLATE=$(ask "Originate channel template ({number} is replaced)" "PJSIP/{number}@out-trunk")
        if [ -z "$AMI_PASSWORD" ]; then
            AMI_USER=$(ask "AMI username" "$AMI_USER")
            AMI_PASSWORD=$(ask_secret "AMI password")
        fi
    fi

    if ask_yesno "Enable SMS channel (SMSEAGLE)?" n; then
        SMSEAGLE_BASE_URL=$(ask "SMSEAGLE base URL" "http://smseagle.local")
        SMSEAGLE_LOGIN=$(ask "SMSEAGLE login" "")
        SMSEAGLE_PASSWORD=$(ask_secret "SMSEAGLE password")
    fi

    if ask_yesno "Enable email channel (SMTP)?" "$([ -n "$MAIL_RELAY_HOST" ] && echo y || echo n)"; then
        if [ -n "$MAIL_RELAY_HOST" ]; then
            SMTP_PORT="$MAIL_RELAY_PORT"
        fi
        SMTP_HOST=$(ask "SMTP host" "${MAIL_RELAY_HOST:-smtp.example.org}")
        SMTP_PORT=$(ask "SMTP port" "$SMTP_PORT")
        SMTP_USERNAME=$(ask "SMTP username (empty for no auth, typical for an internal relay)" "")
        if [ -n "$SMTP_USERNAME" ]; then
            SMTP_PASSWORD=$(ask_secret "SMTP password")
        fi
        SMTP_FROM=$(ask "From address" "alerts@example.org")
    fi

    if ask_yesno "Enable Microsoft Teams channel?" n; then
        TEAMS_WEBHOOK_URL=$(ask "Teams incoming webhook URL" "")
    fi

    LISTEN_ADDR=$(ask "alarm-core HTTP listen address" "$LISTEN_ADDR")
fi

echo
echo "=== Summary ==="
[ -n "$PROXY_URL" ]        && echo "Proxy:      ${PROXY_URL} (bypass: ${NO_PROXY_HOSTS})"
[ -n "$MAIL_RELAY_HOST" ]  && echo "Mail relay: ${MAIL_RELAY_HOST}:${MAIL_RELAY_PORT}$($CONFIGURE_SYSTEM_RELAY && echo " (+ system mail)")"
$BUILD_BINARY        && echo "Build:      alarm-core is built from source (Go) before anything is installed"
$INSTALL_POSTGRES    && echo "PostgreSQL: db=${DB_NAME} user=${DB_USER}"
$INSTALL_ASTERISK    && echo "Asterisk:   AMI user=${AMI_USER} allowed_from=${AMI_ALLOWED_FROM} tts=${TTS_ENABLED}"
if $INSTALL_ALARM_CORE; then
    echo "alarm-core: listen=${LISTEN_ADDR} api_key=$([ -n "$API_KEY" ] && echo "set (shown after install)" || echo "NONE — unauthenticated!")"
    [ -n "$ASTERISK_HOST" ]      && echo "  voice channel via ${ASTERISK_HOST}:${ASTERISK_PORT} (template: ${ASTERISK_CHANNEL_TEMPLATE})"
    [ -n "$SMSEAGLE_BASE_URL" ]  && echo "  sms channel via ${SMSEAGLE_BASE_URL}"
    [ -n "$SMTP_HOST" ]          && echo "  email channel via ${SMTP_HOST}:${SMTP_PORT}"
    [ -n "$TEAMS_WEBHOOK_URL" ]  && echo "  teams channel enabled"
fi
echo
ask_yesno "Proceed with installation?" y || { echo "Aborted, nothing changed."; exit 0; }

if [ -n "$PROXY_URL" ]; then
    echo
    echo ">>> configuring proxy for apt and this script's subprocesses"
    cat > /etc/apt/apt.conf.d/95alarmcore-proxy <<-EOF
Acquire::http::Proxy "${PROXY_URL}";
Acquire::https::Proxy "${PROXY_URL}";
EOF
    export http_proxy="$PROXY_URL" https_proxy="$PROXY_URL" HTTP_PROXY="$PROXY_URL" HTTPS_PROXY="$PROXY_URL"
    export no_proxy="$NO_PROXY_HOSTS" NO_PROXY="$NO_PROXY_HOSTS"
fi

if $BUILD_BINARY; then
    echo
    echo ">>> building alarm-core from source"
    if ! command -v go >/dev/null 2>&1; then
        DEBIAN_FRONTEND=noninteractive apt-get update -qq
        DEBIAN_FRONTEND=noninteractive apt-get install -y golang-go
    fi
    echo "Go: $(go version)"
    echo "(go.mod may require a newer Go than this Debian release ships; the Go toolchain then downloads"
    echo " the required version automatically - this needs internet access or the proxy configured above)"
    export HOME="${HOME:-/root}"
    if ! (cd "$REPO_ROOT" && CGO_ENABLED=0 GOFLAGS=-buildvcs=false go build -o "$BIN_PATH" ./cmd/alarm-core); then
        echo "Error: building alarm-core failed. Nothing else has been installed yet." >&2
        echo "Check the Go output above (typically: no internet access / proxy, or not enough disk space)." >&2
        exit 1
    fi
    chmod +x "$BIN_PATH"
    echo "Built ${BIN_PATH}"
fi
if [ -n "$MAIL_RELAY_HOST" ] && $CONFIGURE_SYSTEM_RELAY; then
    echo
    echo ">>> configuring system mail relay (postfix satellite)"
    debconf-set-selections <<-EOF
postfix postfix/main_mailer_type select Satellite system
postfix postfix/mailname string $(hostname -f 2>/dev/null || hostname)
postfix postfix/destinations string
postfix postfix/relayhost string [${MAIL_RELAY_HOST}]:${MAIL_RELAY_PORT}
EOF
    DEBIAN_FRONTEND=noninteractive apt-get update -qq
    DEBIAN_FRONTEND=noninteractive apt-get install -y postfix >/dev/null
    postconf -e "relayhost = [${MAIL_RELAY_HOST}]:${MAIL_RELAY_PORT}"
    systemctl restart postfix
    if [ -n "$ADMIN_MAIL" ]; then
        if grep -q '^root:' /etc/aliases 2>/dev/null; then
            sed -i "s/^root:.*/root: ${ADMIN_MAIL}/" /etc/aliases
        else
            echo "root: ${ADMIN_MAIL}" >> /etc/aliases
        fi
        newaliases
    fi
fi

if $INSTALL_POSTGRES; then
    echo
    echo ">>> install-postgres.sh"
    ALARM_DB_NAME="$DB_NAME" ALARM_DB_USER="$DB_USER" ALARM_DB_PASSWORD="$DB_PASSWORD" \
        "${SCRIPT_DIR}/install-postgres.sh"
fi

if $INSTALL_ASTERISK; then
    echo
    echo ">>> install-asterisk.sh"
    ASTERISK_AMI_USER="$AMI_USER" ASTERISK_AMI_PASSWORD="$AMI_PASSWORD" \
        ASTERISK_AMI_ALLOWED_FROM="$AMI_ALLOWED_FROM" ASTERISK_ANNOUNCEMENT="$ANNOUNCEMENT" \
        TTS_ENABLED="$TTS_ENABLED" \
        "${SCRIPT_DIR}/install-asterisk.sh"
fi

if $INSTALL_ALARM_CORE; then
    if [ ! -f "${SCRIPT_DIR}/alarm-core" ]; then
        echo
        echo "Error: ${SCRIPT_DIR}/alarm-core binary not found." >&2
        echo "Build it first (needs Go): go build -o deploy/alarm-core ./cmd/alarm-core" >&2
        exit 1
    fi

    echo
    echo ">>> install-alarm-core.sh"
    "${SCRIPT_DIR}/install-alarm-core.sh"

    echo ">>> writing /etc/alarm-core/alarm-core.env"
    DB_DSN="postgres://${DB_USER}:${DB_PASSWORD:-CHANGE_ME}@localhost:5432/${DB_NAME}"
    cat > /etc/alarm-core/alarm-core.env <<-EOF
ALARM_DB_DSN=${DB_DSN}
ALARM_LISTEN_ADDR=${LISTEN_ADDR}
ALARM_API_KEY=${API_KEY}
ALARM_SESSION_SECRET=${SESSION_SECRET}
ALARM_ADMIN_USERNAME=${ADMIN_USERNAME}
ALARM_ADMIN_PASSWORD=${ADMIN_PASSWORD}

ASTERISK_HOST=${ASTERISK_HOST}
ASTERISK_PORT=${ASTERISK_PORT}
ASTERISK_AMI_USER=${AMI_USER}
ASTERISK_AMI_PASSWORD=${AMI_PASSWORD}
ASTERISK_ANNOUNCEMENT=${ANNOUNCEMENT}
ASTERISK_CHANNEL_TEMPLATE=${ASTERISK_CHANNEL_TEMPLATE}
ASTERISK_SOUNDS_DIR=${ASTERISK_SOUNDS_DIR}
TTS_ENABLED=${TTS_ENABLED}
TTS_VOICE=${TTS_VOICE}
TTS_SPEED=${TTS_SPEED}

SMSEAGLE_BASE_URL=${SMSEAGLE_BASE_URL}
SMSEAGLE_LOGIN=${SMSEAGLE_LOGIN}
SMSEAGLE_PASSWORD=${SMSEAGLE_PASSWORD}

SMTP_HOST=${SMTP_HOST}
SMTP_PORT=${SMTP_PORT}
SMTP_USERNAME=${SMTP_USERNAME}
SMTP_PASSWORD=${SMTP_PASSWORD}
SMTP_FROM=${SMTP_FROM}

TEAMS_WEBHOOK_URL=${TEAMS_WEBHOOK_URL}

HTTP_PROXY=${PROXY_URL}
HTTPS_PROXY=${PROXY_URL}
NO_PROXY=${NO_PROXY_HOSTS}
EOF
    chmod 640 /etc/alarm-core/alarm-core.env
    chown root:alarm-core /etc/alarm-core/alarm-core.env

    echo ">>> starting alarm-core"
    systemctl restart alarm-core
    sleep 1
    systemctl --no-pager --full status alarm-core || true
fi

echo
echo "=== Setup complete ==="
if $INSTALL_ALARM_CORE && [ -n "$API_KEY" ]; then
    echo "API key (give this to clients, it is shown only once here — it's"
    echo "also in /etc/alarm-core/alarm-core.env if you lose it):"
    echo "  ${API_KEY}"
fi
if $INSTALL_ALARM_CORE; then
    echo "UI login: http://<this-host>${LISTEN_ADDR} — username '${ADMIN_USERNAME}', the password you just entered."
fi
if $INSTALL_ASTERISK; then
    echo "Remember: fill in the PJSIP trunk in /etc/asterisk/pjsip.conf, then:"
    echo "  asterisk -rx \"pjsip reload\""
fi
