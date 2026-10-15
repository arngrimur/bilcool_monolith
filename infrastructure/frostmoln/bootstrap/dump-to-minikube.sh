#!/usr/bin/env bash
# Copies each service's schema from the Frostmoln Postgres into the minikube Postgres (pod postgres-0, database
# `bilcool`, one schema per service, the same layout as Frostmoln). Connects to Frostmoln as each service role
# using the URLs in db-credentials.env (written by bootstrap.sh).
#
# pg_dump and pg_restore run inside the postgres-0 pod, so the pod needs network access to the Frostmoln load
# balancer and no Postgres client is needed on the host. The dump is a file in the pod's /tmp, removed afterwards.
#
# WARNING: the target schema is DROPPED and recreated for every service in $SERVICES. Anything in minikube
# (seed users, test bookings) is replaced by the Frostmoln data.
#
# Usage:
#   ./dump-to-minikube.sh                      # all services
#   SERVICES="bookings" ./dump-to-minikube.sh  # one service
#   KEEP_OUTBOX=1 ./dump-to-minikube.sh        # leave unsent outbox rows unsent (default: mark them emitted)
#   RESTART=0 ./dump-to-minikube.sh            # do not restart the deployments afterwards
set -euo pipefail

CREDS="${CREDS:-$(dirname "$0")/db-credentials.env}"
SERVICES="${SERVICES:-bookings authentication journal}"
POD="${POD:-postgres-0}"
DB="${DB:-bilcool}"

# shellcheck disable=SC1090
source "$CREDS"

pod_sh() { kubectl exec -i "$POD" -- sh -c "$1"; }
pod_psql() { kubectl exec -i "$POD" -- psql -X -q -v ON_ERROR_STOP=1 -U postgres -d "$DB" "$@"; }

kubectl get pod "$POD" >/dev/null

for s in $SERVICES; do
  var="URL_${s^^}"
  [[ -n "${!var:-}" ]] || { echo "$CREDS has no $var" >&2; exit 1; }
  file="/tmp/$s.dump"

  echo "== $s: dump from Frostmoln =="
  # The URL goes in on stdin so it does not show up in a process list.
  printf '%s\n' "${!var}" | pod_sh "read -r SRC && umask 077 && pg_dump \"\$SRC\" --schema=$s --no-owner --no-privileges -Fc -f $file"
  pod_sh "test -s $file"

  echo "== $s: replace schema in minikube $DB =="
  pod_psql -c "DROP SCHEMA IF EXISTS $s CASCADE" -c "CREATE SCHEMA $s AUTHORIZATION $s"
  # The schema already exists (owned by the service role), so leave its CREATE out of the restore and
  # restore as that role: every table then belongs to it, like after the migrations.
  pod_sh "pg_restore -l $file | grep -v -E '^[0-9]+; [0-9]+ [0-9]+ SCHEMA - ' >$file.list;
          pg_restore -U postgres -d $DB --role=$s --no-owner --exit-on-error -L $file.list $file"

  if [[ -z "${KEEP_OUTBOX:-}" ]] && [[ "$(pod_psql -At -c "SELECT to_regclass('$s.outbox') IS NOT NULL")" == t ]]; then
    # Otherwise the poller would publish rows Frostmoln never relayed to the local SNS.
    pod_psql -c "UPDATE $s.outbox SET emitted_at = now() WHERE emitted_at IS NULL"
  fi

  echo "tables: $(pod_psql -At -c "SELECT count(*) FROM information_schema.tables WHERE table_schema = '$s' AND table_type = 'BASE TABLE'")"
  pod_sh "rm -f $file $file.list"
  echo
done

if [[ "${RESTART:-1}" == 1 ]]; then
  for s in $SERVICES; do kubectl rollout restart "deploy/$s"; done
fi
echo "Done."
