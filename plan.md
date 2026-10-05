# Move from Neon to a single Frostmoln Postgres (one DB, one schema per module)

> **Status (2026-10-04): done and in production.** Released as v1.2.0. The Lambdas run against the Frostmoln database, signup, login and
> booking creation work. Still open: tighten database access
> (the load balancer listener has no CIDR allowlist), and decide on a shorter outbox relay interval (10 min today).
> Current documentation lives in `CLAUDE.md`, `infrastructure/frostmoln/README.md` and `infrastructure/production/terraform/README.md`; this
> file is the plan and progress log from the migration.

## Context
Neon runs out of credits. Replace it with the Frostmoln Postgres instance
(`frostmoln_postgres_instance.bilcool` in `infrastructure/frostmoln/terraform/db.tf`, PG16, in-VPC, no
databases/users/outputs defined yet). Today there are 3 Neon databases (bookings, authentication, journal),
each with its own role (`infrastructure/production/terraform/modules/neon/main.tf`). Target: ONE database
`bilcool` with schemas `bookings`, `authentication`, `journal`.

Decisions made with the user:
- Endpoints stay on AWS Lambda (eu-north-1) for now, move to Frostmoln k8s later.
- Domain stays `bilcool.areskiftet44.se`, DNS stays hand-edited at Loopia.
- Mail stays on Brevo (SES not used, to avoid cost).

## Key finding: the app has no schema awareness
All SQL is unqualified, with no `search_path` anywhere. All services use `lib/pq`. Migrations use dbmate's default
`schema_migrations`; message_broker uses `outbox` and `outbox_schema_migrations`. Table names collide across
services (`users`, `positions`, `inbox`, `outbox`).

Approach: **no Go code changes**. Give each service role a default search_path:
`ALTER ROLE bookings SET search_path = bookings, public;`. Every connection, including dbmate, outbox
`CreateTable`, the poller and the replication connection, then lands in that service's schema. Each schema gets its
own `schema_migrations`, `outbox` and `outbox_schema_migrations`. This is more robust than a URL `search_path=`
parameter. `public` stays in the path so `uuid-ossp` functions resolve (the extension is created once, in `public`).

## 1. Make each endpoint use the right schema
1. **One-time DB bootstrap** (admin user, new SQL file or a `task` target, kept out of the service migrations):
   - `CREATE EXTENSION IF NOT EXISTS "uuid-ossp" SCHEMA public;`
   - Per service `svc` in (bookings, authentication, journal): `CREATE ROLE svc LOGIN PASSWORD ...;
     CREATE SCHEMA svc AUTHORIZATION svc; ALTER ROLE svc SET search_path = svc, public;`
   - `REVOKE CREATE ON SCHEMA public FROM PUBLIC;` plus `REVOKE ALL ON DATABASE bilcool FROM PUBLIC;
     GRANT CONNECT ON DATABASE bilcool TO bookings, authentication, journal;`
   - Prefer doing this in Terraform if the frostmoln provider has database/user resources. Otherwise use the
     `cyrilgdn/postgresql` provider or a one-off psql script (see step 2).
2. **Connection strings** (replace `module.neon.*` in `infrastructure/production/terraform/main.tf`, lines ~65-290):
   `postgres://svc:PW@<frostmoln-host>:5432/bilcool?sslmode=require` for both runtime and migrate Lambdas.
   - **Drop `pgbouncer=true`.** lib/pq forwards unknown params to the server, which rejects them.
   - There is no pooler, so keep Lambda connections low: authentication and bookings already use
     `MaxOpenConns=2`. Journal always uses 5 open, 5 idle (`journal/.../setup.go`), so add an
     `AWS_LAMBDA_FUNCTION_NAME` branch like the other two. Set reserved concurrency on the Lambdas so total
     connections stay under the instance's `max_connections` (db.gp1.small).
   - Replace `modules/neon` with a thin `modules/database` (or locals) that outputs the URLs. Remove the neon
     provider and `neon_api_key` from `providers.tf`, `variables.tf` and `terraform.tfvars.example`.
3. **Migrations:** the migrate Lambdas (`*/cmd/lambda/migrate/main.go`) need no code change. They run after bootstrap
   because the schemas must exist first. Keep the `aws_lambda_invocation` ordering with `depends_on` on the bootstrap.
4. **Outbox:** production uses `OUTBOX_MODE=polling`, so the poller's `FROM outbox` resolves via search_path. For the
   later k8s move (replication mode), `CREATE PUBLICATION ... FOR TABLE "outbox"` also resolves via the role's
   search_path. Verify then that `wal_level=logical` is available on the Frostmoln instance, because it is not
   needed on Lambda.
5. **Data move from Neon (decide if needed):** `pg_dump --no-owner` each Neon DB. The dumps contain `public.` qualified
   names and a `SET search_path = ''`, so rewrite `public.` to `<svc>.` (sed) before `psql`. Skip if the data is only test data.
   Include `schema_migrations` so dbmate does not re-run migrations.
6. **Helm/k8s later:** set `postgres.location: frostmoln` and the three `secrets.*.databaseUrl` values. They use the
   same URLs, so no chart changes are needed beyond values.

## 2. Make the Frostmoln DB reachable
Files: `infrastructure/frostmoln/terraform/db.tf`, `main.tf`, `variables.tf`, `environments/*.tfvars`.
1. **First, verify the provider schema** (`terraform providers schema -json`, or the provider docs), since I could not
   confirm these: how the instance gets a public endpoint or IP (the `gateway`/`public_ip` resource, or an instance
   attribute), whether database/user resources exist, how host and credentials are exposed, and whether `ssl` is enforced.
2. Add to `db.tf`: outputs for host and port, and sensitive outputs or variables for the role passwords (set via
   `TF_VAR_*`, never committed; the `.envrc` already holds an API key and is gitignored).
3. Add a `frostmoln_security_group` and `frostmoln_security_group_rule` for TCP 5432 and attach it to the DB (the existing `web`
   group is unattached).
   - **This host:** ingress from this machine's IP `/32` (variable `admin_cidr`).
   - **Lambdas:** AWS Lambda has no fixed egress IPs, so an allowlist is impossible. Interim choice: `0.0.0.0/0` on 5432
     with TLS required, SCRAM, long random passwords and per-service least-privilege roles. Optional hardening
     later: put the Lambdas in a VPC behind a NAT gateway with an Elastic IP and allow only that `/32`
     (about $30+/month). That is moot once the apps move into the Frostmoln VPC.
4. Align versions: dev and Helm use `postgres:18`, Frostmoln is set to 16. Check the schema and migrations on 16, or set
   the instance version to 18 if it is available.
5. Terraform state: the S3 backend (`bilcool-terraform-state`) is already added in `main.tf`. Run `terraform init
   -migrate-state`, and commit the `.gitignore` changes and the staged `.terraform` removals.

### Provider findings (frostmoln provider 0.73.2, read from its schema on 2026-10-03)
- **No database or user resources.** The provider cannot create `bilcool`, the schemas or the roles. The bootstrap
  must be SQL run as the admin user, either by hand or through the `cyrilgdn/postgresql` provider.
- **The admin password is never exposed.** `admin_username` is readable, but "the admin password is never served by
  the read surface". The password comes from the portal or CLI. Do NOT try to read it from Terraform.
- **Public exposure goes through a public L4 load balancer** (written in `infrastructure/frostmoln/terraform/db_access.tf`,
  `terraform validate` passes, not applied): `frostmoln_public_ip` + `frostmoln_load_balancer` (type `l4`, scheme `public`)
  + `frostmoln_lb_listener` (tcp 5432, `allowed_cidrs` allowlist) + pool + member (DB private IP) + tcp health monitor.
  The instance's own `public_ip` stays null. `db_host` output is the address for connection strings.
- **No security-group attachment on the DB resource.** Port access is probably controlled by the platform, so confirm in the portal.
- **Parameter groups are never applied.** `parameter_group_id` is refused at plan. `wal_level=logical` therefore cannot be
  set, so the later k8s move must keep `OUTBOX_MODE=polling`, not replication.
- `extensions` is supported, but `uuid-ossp` needs checking against `frostmoln_database_types`.

## 3. DNS (Loopia, manual)
**Site (`bilcool.areskiftet44.se`)**
- The name stays the same, so the existing Loopia records may need no change. Check that both still exist
  (see `infrastructure/production/terraform/README.md:46-104`): the ACM validation CNAME and the CNAME for
  `bilcool` pointing at the CloudFront distribution (`terraform output`). If the AWS stack is recreated, both change, so copy
  the new values from the outputs.
- Moving to Frostmoln k8s later: replace the CNAME with an A record to the Frostmoln gateway/LB public IP, and issue a
  TLS cert there (cert-manager). Keep `WEBAUTHN_RP_ID=bilcool.areskiftet44.se` and
  `WEBAUTHN_RP_ORIGINS=https://bilcool.areskiftet44.se` unchanged so passkeys keep working.

**Mail (Brevo, sender `security@areskiftet44.se`)**
- In Brevo, add and authenticate the sender domain `areskiftet44.se`. Brevo shows the exact records, which are typically:
  - a TXT verification code,
  - DKIM CNAME or TXT records (selectors like `brevo1._domainkey`),
  - SPF: **one** TXT at the root that includes Brevo's sender (merge it into any existing SPF, because a second SPF record breaks it),
  - DMARC TXT at `_dmarc` (start with `p=none` and an `rua=` address).
- Do not touch any existing MX records. Check whether the apex already has mail before editing SPF.
- Confirm `FROM_EMAIL=security@areskiftet44.se` in the Lambda env, replacing the `noreply@yourdomain.com` placeholder in
  `terraform.tfvars.example`.

## Verification
1. `terraform plan` on the frostmoln stack, then apply. `psql "postgres://bookings:...@<host>/bilcool?sslmode=require" -c 'show search_path'`
   from this host returns `bookings, public`. Repeat for the other roles.
2. Run the migrate Lambdas, or dbmate locally with each URL. Check `\dn`, `\dt bookings.*`, `\dt authentication.*`,
   and that each schema has its own `schema_migrations`, `outbox` and `outbox_schema_migrations`.
3. Negative test: role `bookings` cannot read `authentication.users`.
4. `go generate ./... && go test ./...`, and the integration tests with `-tags=integration` (no code changed, so a
   regression check). Optionally add a testdb integration test that migrates two services into one DB with
   separate search_paths.
5. Deploy and smoke test through CloudFront: login (a Brevo email arrives and passes SPF/DKIM/DMARC; check
   headers or mail-tester), create a booking, and confirm the journal event appears via the outbox poller.
6. Watch Postgres connection counts under load against `max_connections`.
7. Decommission the Neon project only after a successful soak.

## Open items / risks
- Provider capabilities for public access and DB/user management are unverified (step 2.1).
- `0.0.0.0/0` on 5432 is the weakest part of the interim design. It is accepted for cost reasons and temporary.
- Neon data migration needs a user decision on whether the data matters.

## Progress log
- **2026-10-03, step 4 done (code only, not applied):**
  - `journal/.../postgres/setup.go`: Lambda connection limits (2 open, 0 idle), same as bookings and authentication.
  - `infrastructure/production/terraform/modules/database/`: takes the role passwords as `var.passwords` (root
    `var.db_passwords`), URLs without `pgbouncer=true`. The DB, schemas and roles come from
    `infrastructure/frostmoln/bootstrap/bootstrap.sh` (see below), not from Terraform.
    `main.tf` Lambdas now use `module.database`. New vars `db_host` and `db_port`.
  - `module "neon"` was kept at this point so `apply` would not destroy the Neon project before the data was copied (removed later, see the 2026-10-04 entry).
  - **Apply order matters:** (1) apply the frostmoln stack, (2) run `database_bootstrap_sql` as the admin user,
    (3) only then apply the production stack. The migrate Lambdas run during that apply and fail if the schemas
    do not exist. The first production apply needs the bootstrap output, so use
    `terraform apply -target=module.database` first, run the bootstrap, then do the full apply.
- **2026-10-03, Frostmoln DB reachable (dev workspace):** public IP + L4 load balancer + pool/member/health monitor
  applied; the listener was created by hand in the portal and imported, because the platform refused every
  Terraform-created listener that sent `allowed_cidrs` (generic failure). `lifecycle.ignore_changes` keeps Terraform
  from pushing the CIDR list. TCP and TLS 1.3 verified from this host. The listener's effective access with an empty
  CIDR list is UNVERIFIED. The certificate is self-signed, so use `sslmode=require`.
- **Bootstrap:** `infrastructure/frostmoln/bootstrap/bootstrap.sh` (idempotent; needs `PGPASSWORD` + `DB_HOST`) creates
  the DB, extension, schemas, roles and `search_path`, writes credentials to the gitignored `db-credentials.env`
  and verifies each role's `search_path`. Rotate the pgadmin password: it was pasted into a chat.
- **2026-10-03, Neon data skipped** (test data only): the migrate Lambdas create fresh tables in the new schemas.
- **Migrate invocations now re-run on a DB change:** `aws_lambda_invocation.*_migrate` had no `triggers`, so Terraform would
  not have re-run them after the DB switch (CI does, a bare apply did not). Added `triggers = { database = sha256(url) }`.
- **Production stack** (`infrastructure/production/terraform`, `default` workspace, 99 resources, live on Neon): initialised
  locally. Applying needs a gitignored `terraform.tfvars` with `db_host`, `db_passwords` (from `bootstrap/db-credentials.env`)
  and the existing jwt_secret, brevo_api_key, mapbox_access_token, neon_api_key and so on.
- **2026-10-04, Neon removed (PR #80):** `module "neon"`, `modules/neon/`, the neon provider and the helm `postgres.location: neon` option are gone. The leftover `neon_api_key` variable was removed afterwards.
