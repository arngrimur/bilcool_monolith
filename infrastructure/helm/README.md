# Helm chart `bilcool`

One chart, two environments:

| File | Purpose |
|---|---|
| `bilcool/values.yaml` | Defaults (everything optional is off) |
| `values-dev.yaml` | minikube: in-cluster Postgres, LocalStack, seed user, throwaway passwords |
| `values-prod.yaml` | Frostmoln k8s + Frostmoln Postgres + real AWS (**not written yet**, see `/plan.md` and "Prod" below) |
| `values-secrets.yaml` | **Gitignored.** Your secrets. Copy from `values-secrets.example.yaml` |

## Dev quickstart (minikube)

```bash
cp infrastructure/helm/values-secrets.example.yaml infrastructure/helm/values-secrets.yaml   # once
task up            # start minikube, build + load all images, helm upgrade --install -f values-dev.yaml -f values-secrets.yaml
task minikube:port-forward   # UI on :3000, Postgres on :54320, LocalStack on :4566
task down          # helm uninstall (keeps the database volume)
task helm:down:purge # DESTRUCTIVE: also deletes Postgres, secrets and the volume
```

Put your LocalStack auth token in `values-secrets.yaml` (`secrets.localstackAuthToken`, required: LocalStack 2026.x does not start without it).
Dev needs no Brevo account: `mailSender: log` makes the authentication service write the login code to its pod log instead of mailing it.
Log in with the seed user from `values-dev.yaml` (`seed.users`), then read the code:

```bash
kubectl logs deploy/authentication | grep "security token"     # field "token" holds the code
```

To receive real mail instead, set `config.authentication.mailSender: brevo` and `secrets.authentication.brevoApiKey`.

### What gets deployed in dev

- **Postgres**: ONE instance, database `bilcool`, one schema and login role per service (`bookings`, `authentication`, `journal`)
  with `search_path = <service>, public`. This mirrors Frostmoln (`infrastructure/frostmoln/bootstrap/bootstrap.sh`);
  `files/db-init.sh` is the in-cluster port. The Go code never schema-qualifies table names.
- **LocalStack**: SNS topics `bilcool_users`, `bilcool_bookings`, SQS queues `bilcool-monolith-{event_ledger,bookings,journal}` with DLQs,
  created by `files/init-aws.sh`. The DynamoDB table is created by the event_ledger service itself.
- **Migrations**: one dbmate Job per service (`arngrimur/bilcool-<svc>-migrate`), as service role.
- **Seed**: `seed.users` are inserted into authentication, bookings and journal.
- **Services**: authentication, bookings, journal, event-ledger and the UI. Service names are fixed
  (`ui/bilcool-ui/nginx.conf` proxies to them by name), so the chart does not prefix them with the release name; one release per namespace.

### Ordering and hooks

The migrate and seed Jobs are `pre-install,pre-upgrade` hooks, and they read DB URLs from Secrets. Normal resources are installed
after pre-hooks, so the Secrets, the dev Postgres StatefulSet/Service and the `db-init` Job are hooks as well (weights -10, -8, -4/-3;
migrate 0; seed 4/5). Consequences:

- Helm does not wait for the Postgres StatefulSet; `db-init` waits with `pg_isready`.
- Hook resources are **not** removed by `helm uninstall`. Use `task helm:down:purge`.

### OUTBOX_MODE

`config.outboxMode` -> `OUTBOX_MODE` for all services. `polling` (default) or `replication`. Which one wins long term is **undecided**;
both stay supported. Dev Postgres runs with `wal_level=logical` and `postgres.replicationRoles: true` grants `REPLICATION`
to the service roles so `replication` can be tried:

```bash
helm upgrade bilcool ./infrastructure/helm/bilcool -f infrastructure/helm/values-dev.yaml -f infrastructure/helm/values-secrets.yaml \
  --set config.outboxMode=replication
```

Verified in minikube (shared database, schema per service): both modes deliver `user.created` to bookings, journal and the event ledger
(polling about 15 s, replication about 3 s). Notes on `replication`:

- the service role needs `REPLICATION` and `CREATE` on the database (`postgres.replicationRoles: true`, dev only);
- a replication slot has exactly one consumer, so bookings and authentication run one replica with `strategy: Recreate`
  in this mode (the chart does this automatically).

Frostmoln cannot set `wal_level` today, so prod uses `polling`.

## Prod (planned)

`postgres.enabled=false`, `localstack.enabled=false`, `seed.enabled=false`, empty `config.awsEndpointUrl`, explicit
`secrets.<service>.databaseUrl` (Frostmoln, `sslmode=require`) and AWS IAM-user keys. See `/plan.md` and
`infrastructure/frostmoln/README.md`. CI deploys with `HELM_VALUES_SECRETS`.

## Checks

```bash
helm lint infrastructure/helm/bilcool -f infrastructure/helm/values-dev.yaml -f infrastructure/helm/values-secrets.example.yaml
helm template bilcool infrastructure/helm/bilcool -f infrastructure/helm/values-dev.yaml -f infrastructure/helm/values-secrets.example.yaml
```
