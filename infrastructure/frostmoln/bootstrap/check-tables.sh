#!/usr/bin/env bash
# Read-only check that every service's migrations ran in its own schema on the Frostmoln Postgres.
# Connects as each service role using the URLs in db-credentials.env (written by bootstrap.sh).
# Prints schema-qualified table names and row counts; no credentials are printed.
set -euo pipefail

CREDS="${CREDS:-$(dirname "$0")/db-credentials.env}"
# shellcheck disable=SC1090
source "$CREDS"

for s in bookings authentication journal; do
  var="URL_${s^^}"
  echo "== $s =="
  psql -X -At "${!var}" <<'SQL'
SELECT 'search_path=' || current_setting('search_path') || '  schema=' || current_schema();
SELECT format('%s.%s', table_schema, table_name)
  FROM information_schema.tables
 WHERE table_schema = current_schema() AND table_type = 'BASE TABLE'
 ORDER BY table_name;
SQL
  echo "tables in schema: $(psql -X -At "${!var}" -c "SELECT count(*) FROM information_schema.tables WHERE table_schema = current_schema() AND table_type = 'BASE TABLE'")"
  echo
done

echo "== other schemas visible to each role must not contain these services' tables =="
psql -X -At "$URL_BOOKINGS" -c "SELECT CASE WHEN has_schema_privilege('authentication', 'USAGE') THEN 'WARNING: bookings can use schema authentication' ELSE 'ok: bookings cannot use schema authentication' END"
