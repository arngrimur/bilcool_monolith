#!/usr/bin/env bash
# One-time (idempotent) setup of the Frostmoln Postgres: one database, one schema + login role per service.
#
#   export PGPASSWORD=<pgadmin password from the Frostmoln portal>   # do not paste it into chat
#   DB_HOST=<terraform output db_host> ./bootstrap.sh
#
# Creates database $DB_NAME, the uuid-ossp extension (in public), and for each of bookings, authentication
# and journal: a login role, a schema owned by it, and `search_path = <service>, public` on the role, so the
# services need no schema-aware code. Re-running is safe: existing objects are kept and nothing is dropped.
#
# Role passwords are generated on the first run and written to $OUT (mode 600, gitignored). A re-run reuses
# them; delete $OUT first to rotate. Passwords go to psql on stdin, never on a command line.
# $OUT consists of `export` lines, so `. $OUT` alone puts TF_VAR_db_passwords (and the PW_*/URL_* values)
# into the current shell's environment, ready for terraform. Run `terraform` from that same shell.
set -euo pipefail

: "${PGPASSWORD:?export PGPASSWORD (the pgadmin password) first}"
: "${DB_HOST:?set DB_HOST (terraform output db_host)}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-bilcool}"
ADMIN_USER="${ADMIN_USER:-pgadmin}"
OUT="${OUT:-$(dirname "$0")/db-credentials.env}"
SERVICES=(bookings authentication journal)

export PGSSLMODE=require PGCONNECT_TIMEOUT=10
admin_psql() { # <database>; SQL on stdin
  psql -X -q -v ON_ERROR_STOP=1 -h "$DB_HOST" -p "$DB_PORT" -U "$ADMIN_USER" -d "$1"
}

declare -A PW
if [[ -f "$OUT" ]]; then
  echo "Reusing passwords from $OUT"
  # Upgrade a file written before it exported its variables (keeps the passwords; idempotent, mode is kept).
  sed -i -E 's/^([A-Za-z_][A-Za-z0-9_]*=)/export \1/' "$OUT"
  # shellcheck disable=SC1090
  source "$OUT"
  for s in "${SERVICES[@]}"; do
    var="PW_${s^^}"
    [[ -n "${!var:-}" ]] || { echo "$OUT has no $var" >&2; exit 1; }
    PW[$s]="${!var}"
  done
else
  for s in "${SERVICES[@]}"; do PW[$s]="$(openssl rand -hex 24)"; done # hex: URL-safe
  umask 077
  {
    for s in "${SERVICES[@]}"; do echo "export PW_${s^^}=${PW[$s]}"; done
    for s in "${SERVICES[@]}"; do
      echo "export URL_${s^^}='postgres://${s}:${PW[$s]}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=require'"
    done
    printf "export TF_VAR_db_passwords='{\"bookings\":\"%s\",\"authentication\":\"%s\",\"journal\":\"%s\"}'\n" \
      "${PW[bookings]}" "${PW[authentication]}" "${PW[journal]}"
  } >"$OUT"
  echo "Wrote $OUT (mode 600)"
fi

echo "Creating database $DB_NAME if missing"
admin_psql postgres <<SQL
\set db '$DB_NAME'
SELECT format('CREATE DATABASE %I', :'db') WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = :'db') \gexec
SQL

echo "Database-level setup"
admin_psql "$DB_NAME" <<SQL
\set db '$DB_NAME'
CREATE EXTENSION IF NOT EXISTS "uuid-ossp" SCHEMA public;
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
SELECT format('REVOKE ALL ON DATABASE %I FROM PUBLIC', :'db') \gexec
SQL

for s in "${SERVICES[@]}"; do
  echo "Service $s"
  admin_psql "$DB_NAME" <<SQL
\set svc '$s'
\set pw '${PW[$s]}'
\set db '$DB_NAME'
SELECT format('CREATE ROLE %I LOGIN PASSWORD %L', :'svc', :'pw') WHERE NOT EXISTS (SELECT FROM pg_roles WHERE rolname = :'svc') \gexec
SELECT format('ALTER ROLE %I PASSWORD %L', :'svc', :'pw') \gexec
SELECT format('GRANT %I TO %I', :'svc', current_user) \gexec
SELECT format('CREATE SCHEMA IF NOT EXISTS %I AUTHORIZATION %I', :'svc', :'svc') \gexec
SELECT format('ALTER ROLE %I SET search_path = %I, public', :'svc', :'svc') \gexec
SELECT format('GRANT CONNECT ON DATABASE %I TO %I', :'db', :'svc') \gexec
SQL
done

echo "Verifying as each service role"
for s in "${SERVICES[@]}"; do
  got="$(PGPASSWORD="${PW[$s]}" psql -X -qAt -h "$DB_HOST" -p "$DB_PORT" -U "$s" -d "$DB_NAME" \
    -c "SELECT current_user || ' search_path=' || current_setting('search_path') || ' schema=' || current_schema()")"
  echo "  $got"
  [[ "$got" == "$s search_path=$s, public schema=$s" ]] || { echo "unexpected search_path for $s" >&2; exit 1; }
done
echo "Done. Credentials: $OUT"
