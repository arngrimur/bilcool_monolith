#!/bin/sh
# Dev Postgres bootstrap: one database "bilcool", a schema + login role per service with search_path = <svc>, public.
# Same logic as infrastructure/frostmoln/bootstrap/bootstrap.sh. Idempotent.
set -eu
until pg_isready -h postgres -U postgres -d bilcool; do echo "waiting for postgres"; sleep 2; done
export PGPASSWORD="$ADMIN_PASSWORD"
psql -X -q -v ON_ERROR_STOP=1 -h postgres -U postgres -d bilcool <<'SQL'
CREATE EXTENSION IF NOT EXISTS "uuid-ossp" SCHEMA public;
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
SQL

setup() { # <service> <password>
  psql -X -q -v ON_ERROR_STOP=1 -v svc="$1" -v pw="$2" -v repl="$REPLICATION_ROLES" -h postgres -U postgres -d bilcool <<'SQL'
SELECT format('CREATE ROLE %I LOGIN PASSWORD %L', :'svc', :'pw') WHERE NOT EXISTS (SELECT FROM pg_roles WHERE rolname = :'svc') \gexec
SELECT format('ALTER ROLE %I PASSWORD %L', :'svc', :'pw') \gexec
SELECT format('CREATE SCHEMA IF NOT EXISTS %I AUTHORIZATION %I', :'svc', :'svc') \gexec
SELECT format('ALTER ROLE %I SET search_path = %I, public', :'svc', :'svc') \gexec
SELECT format('GRANT CONNECT ON DATABASE bilcool TO %I', :'svc') \gexec
-- OUTBOX_MODE=replication: the service creates its publication/slot itself (dev only, postgres.replicationRoles)
SELECT format('ALTER ROLE %I REPLICATION', :'svc') WHERE :'repl' = 'true' \gexec
SELECT format('GRANT CREATE ON DATABASE bilcool TO %I', :'svc') WHERE :'repl' = 'true' \gexec
SELECT format('GRANT CREATE ON DATABASE bilcool TO %I', :'svc') WHERE :'repl' = 'true' \gexec
SQL
}
setup bookings "$PW_BOOKINGS"
setup authentication "$PW_AUTHENTICATION"
setup journal "$PW_JOURNAL"
echo "db-init done"
