# Frostmoln database infrastructure

One Frostmoln managed PostgreSQL holds the data of all Postgres-backed services. It replaces the three Neon databases.
It contains **one database `bilcool`** with **one schema per service** (`bookings`, `authentication`, `journal`), and each
service has its own login role. The production AWS stack (Lambdas) connects to it through a public Layer-4 load balancer.

```
infrastructure/frostmoln/
├── terraform/    # VPC, subnet, gateway, Postgres instance, public load balancer
└── bootstrap/    # bootstrap.sh (database, schemas, roles), check-tables.sh (verification)
```

## Prerequisites

- Terraform with the `frostmoln/frostmoln` provider, `psql` and `openssl`.
- AWS credentials for the S3 state backend (`AWS_PROFILE`; sign in with `aws login`).
- A Frostmoln API key (portal: Settings -> API Keys) exported as `FROSTMOLN_API_KEY`. Keep it in
  `terraform/.envrc`, which is gitignored. Never commit it, print it or paste it into a chat.

## Terraform

State lives in S3: bucket `bilcool-terraform-state`, key `frostmoln/terraform.tfstate`, region `eu-north-1`
(the bucket is created by `infrastructure/production/bootstrap`). Resource names and tags use `terraform.workspace`,
so **always use the `dev` workspace**:

```bash
cd infrastructure/frostmoln/terraform
set -a && . ./.envrc && set +a            # FROSTMOLN_API_KEY, AWS_PROFILE
terraform init
terraform workspace select dev            # or: terraform workspace new dev
terraform plan  -var-file=environments/dev.tfvars
terraform apply -var-file=environments/dev.tfvars
```

| File | Contents |
|---|---|
| `main.tf` | Provider, S3 backend, default tags, VPC `bilcool-<workspace>-vpc` (10.0.0.0/16), subnet `bilcool_net` (10.0.1.0/24, zone `falkenberg`), gateway (`public_ip` mode), security group `web` |
| `db.tf` | `frostmoln_postgres_instance.bilcool`: name `bilcool-db`, PostgreSQL 16, `db.gp1.small`, 50 GB, no HA, backups on. It has a private IP only. |
| `db_access.tf` | Public IP, L4 load balancer, listener, pool, member and health monitor that expose the database (see below) |
| `variables.tf` | `env` (declared, unused), `system` |
| `environments/dev.tfvars` | `env`, `system`, `db_allowed_cidrs` |

Outputs: `db_host` (the public address to use in connection strings), `db_port`, `db_admin_username` (`pgadmin`),
`gateway_source_address`, `vpc_id`, `subnet_id`.

### How the database is exposed (`db_access.tf`)

The instance has no public address of its own. A public L4 (TCP) load balancer forwards port 5432 to its private IP. L4
passes TLS straight through to Postgres and keeps the client source IP.

- `frostmoln_public_ip.db` has `prevent_destroy`: releasing the address is irreversible and it is used in connection strings.
- `frostmoln_load_balancer.db`: `type = l4`, `scheme = public`; it depends on the VPC gateway.
- `frostmoln_lb_listener.db`: TCP 5432, named `bilcool-listener`.
- `frostmoln_lb_pool.db` (`source_ip_port`, the only algorithm an L4 balancer accepts), `frostmoln_lb_member.db` (the instance's
  private IP and port) and `frostmoln_lb_health_monitor.db` (TCP).

**Known platform issue: the listener was created by hand.** The platform refused every listener create that Terraform sent
with `allowed_cidrs` (the error is a generic "operation could not be completed"). The listener was therefore created in the
portal (Network -> Load Balancers -> `bilcool-dev-db-lb` -> Add Listener, TCP 5432, no CIDR list) and imported:

```bash
terraform import -var-file=environments/dev.tfvars frostmoln_lb_listener.db <load-balancer-id>/<listener-id>
```

Both IDs are shown in the portal; the load balancer ID is also in `terraform state show frostmoln_load_balancer.db`.
`lifecycle { ignore_changes = [allowed_cidrs] }` stops Terraform from pushing the CIDR list, so **`db_allowed_cidrs` is not
applied today**. What an empty CIDR list means on the platform was not verified. Treat the database as reachable from the
internet and protected by TLS (`sslmode=require`, self-signed certificate, so `verify-full` is not possible), SCRAM
passwords and per-service roles. AWS Lambda has no fixed egress IPs, so an IP allowlist is not possible until the apps
move into the Frostmoln VPC.

## One-time database bootstrap

The provider cannot create databases, schemas or roles, and never exposes the admin password. `bootstrap/bootstrap.sh`
does this over a normal Postgres connection as `pgadmin`:

```bash
export PGPASSWORD=<pgadmin password from the portal>      # do not paste it into a chat
DB_HOST=$(terraform -chdir=../terraform output -raw db_host) ./bootstrap/bootstrap.sh
```

It is idempotent (nothing is dropped). It:

1. creates the database `bilcool` if missing, the `uuid-ossp` extension in `public`, and revokes `CREATE` on `public`;
2. for `bookings`, `authentication` and `journal`: creates a login role, a schema owned by it, and sets
   `ALTER ROLE <service> SET search_path = <service>, public`, then grants `CONNECT`;
3. connects as each service role and checks `search_path` and `current_schema()`.

Because of the role-level `search_path`, the services need no schema-aware code: queries, dbmate migrations
(`schema_migrations`) and the outbox (`outbox`, `outbox_schema_migrations`) all land in the service's own schema.

Optional settings: `DB_PORT` (5432), `DB_NAME` (`bilcool`), `ADMIN_USER` (`pgadmin`), `OUT` (credentials file path).

### `db-credentials.env`

The script generates one random password per role (`openssl rand -hex 24`) and writes them to
`bootstrap/db-credentials.env` (mode 600, gitignored). The file consists of `export` lines: `PW_<SERVICE>`,
`URL_<SERVICE>` (`postgres://<service>:<password>@<host>:5432/bilcool?sslmode=require`) and `TF_VAR_db_passwords`
(JSON, consumed by the production Terraform variable `db_passwords`). Load it with `. bootstrap/db-credentials.env` and run
Terraform from that same shell. A re-run reuses the passwords; delete the file first to rotate them (then re-apply the
production stack so the Lambdas get the new URLs).

### Verify

```bash
bash bootstrap/check-tables.sh
```

Connects as each service role and prints its `search_path`, the schema-qualified tables and the table count, and checks that
the `bookings` role cannot use the `authentication` schema. Run it after the migrate Lambdas have run.

## How production uses this

`infrastructure/production/terraform/modules/database` builds the connection strings from `db_host` (this stack's `db_host`
output) and `db_passwords`. There is no connection pooler, so URLs must not contain `pgbouncer=true`, and the Lambdas keep few
connections (`MaxOpenConns=2`). Migrations are run by the three migrate Lambdas; the outbox tables are created by
`message_broker`'s `CreateTable` on cold start. See `infrastructure/production/terraform/README.md` and the repo-root `plan.md`.

## Caveats

- No `prod` workspace is set up (`environments/prod.tfvars` is empty); only `dev` exists.
- The provider's parameter groups are not applied by the platform, so `wal_level=logical` cannot be configured through Terraform
  (the instance's actual value was not checked). Production therefore runs the outbox in `OUTBOX_MODE=polling`.
- Rotate the `pgadmin` password and the API key if they were ever pasted anywhere they should not be.
