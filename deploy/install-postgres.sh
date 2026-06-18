#!/bin/bash
# Installs PostgreSQL on Debian 13, creates the alarm-core database/role and
# applies the schema migrations. Idempotent: safe to re-run.
#
# Usage:
#   ALARM_DB_PASSWORD=secret ./install-postgres.sh
#
# Optional env vars:
#   ALARM_DB_NAME       (default: alarmcore)
#   ALARM_DB_USER       (default: alarmcore)
#   ALARM_BACKUP_DIR    (default: /var/backups/alarmcore)
set -euo pipefail

if [ "$(id -u)" -ne 0 ]; then
    echo "Run as root." >&2
    exit 1
fi

DB_NAME="${ALARM_DB_NAME:-alarmcore}"
DB_USER="${ALARM_DB_USER:-alarmcore}"
DB_PASSWORD="${ALARM_DB_PASSWORD:?Set ALARM_DB_PASSWORD before running this script}"
BACKUP_DIR="${ALARM_BACKUP_DIR:-/var/backups/alarmcore}"
MIGRATIONS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../internal/db/migrations" && pwd)"

echo "[1/4] Installing PostgreSQL ..."
apt-get update -qq
DEBIAN_FRONTEND=noninteractive apt-get install -y postgresql >/dev/null

echo "  Ensuring the cluster is running ..."
systemctl enable --now postgresql >/dev/null
for i in $(seq 1 10); do
    sudo -u postgres pg_isready -q && break
    sleep 1
done
sudo -u postgres pg_isready -q || { echo "PostgreSQL did not come up in time." >&2; exit 1; }

echo "[2/4] Creating database and role ..."
sudo -u postgres psql -v ON_ERROR_STOP=1 -q <<-SQL
    DO \$\$
    BEGIN
        IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = '${DB_USER}') THEN
            CREATE ROLE ${DB_USER} LOGIN PASSWORD '${DB_PASSWORD}';
        ELSE
            ALTER ROLE ${DB_USER} PASSWORD '${DB_PASSWORD}';
        END IF;
    END
    \$\$;
    SELECT 'CREATE DATABASE ${DB_NAME} OWNER ${DB_USER}'
    WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = '${DB_NAME}')\gexec
SQL

echo "[3/4] Applying schema migrations ..."
psql_run() {
    PGPASSWORD="${DB_PASSWORD}" psql -h localhost -U "${DB_USER}" -d "${DB_NAME}" -v ON_ERROR_STOP=1 "$@"
}
psql_run -q -c "CREATE TABLE IF NOT EXISTS schema_migrations (filename TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW())"
for f in "${MIGRATIONS_DIR}"/*.sql; do
    name=$(basename "$f")
    ALREADY_APPLIED=$(psql_run -tAc "SELECT EXISTS (SELECT FROM schema_migrations WHERE filename = '${name}')")
    if [ "$ALREADY_APPLIED" = "t" ]; then
        echo "  -> ${name} (already applied, skipping)"
        continue
    fi
    echo "  -> ${name}"
    psql_run -f "$f"
    psql_run -q -c "INSERT INTO schema_migrations (filename) VALUES ('${name}')"
done

echo "[4/4] Setting up daily backup (pg_dump -> ${BACKUP_DIR}, 14 days retention) ..."
mkdir -p "${BACKUP_DIR}"
chown postgres:postgres "${BACKUP_DIR}"
cat > /usr/local/bin/alarmcore-backup.sh <<-EOF
#!/bin/bash
set -euo pipefail
ts=\$(date +%Y%m%d_%H%M%S)
PGPASSWORD='${DB_PASSWORD}' pg_dump -h localhost -U "${DB_USER}" -d "${DB_NAME}" \\
    -F c -f "${BACKUP_DIR}/${DB_NAME}_\${ts}.dump"
find "${BACKUP_DIR}" -name '${DB_NAME}_*.dump' -mtime +14 -delete
EOF
chmod 755 /usr/local/bin/alarmcore-backup.sh
cat > /etc/cron.d/alarmcore-backup <<-EOF
0 3 * * * postgres /usr/local/bin/alarmcore-backup.sh
EOF

echo "Done. DSN: postgres://${DB_USER}:***@localhost:5432/${DB_NAME}"
echo "Backups: daily at 03:00 -> ${BACKUP_DIR} (14 day retention)"
