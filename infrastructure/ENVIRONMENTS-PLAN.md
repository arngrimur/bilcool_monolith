# Plan: three environments, all as infrastructure as code

> **Status: proposal for review, revision 4 (2026-11-12). Nothing is implemented. All decisions are made (section 6).**
> Applied so far: Terraform everywhere, LocalStack in Docker, provider interfaces selected by environment variables,
> secrets only in the secrets services with the same keys, DNS unchanged, state in S3, small pull requests.
> Revision 3: only one of `aws` and `frostmoln` exists at a time, on the same URL; Frostmoln's managed LavinMQ, secrets
> API and workload identity are confirmed from `docs.frostmoln.se` (section 8).

## 1. Target

| | **local** | **aws** (production, as today) | **frostmoln** (production, alternative) |
|---|---|---|---|
| Runtime | minikube, Helm chart | AWS Lambda + Function URLs | Frostmoln Kubernetes, Helm chart |
| Database | Postgres 18 in Docker beside minikube | Frostmoln Postgres through the public L4 load balancer | Frostmoln Postgres on its private VPC address |
| Messaging | LocalStack SNS/SQS in Docker (LavinMQ in Docker as a switch) | AWS SNS/SQS | Frostmoln managed LavinMQ (AMQPS, private to the VPC) |
| Secrets | LocalStack Secrets Manager | AWS Secrets Manager | Frostmoln Secrets, read with workload identity |
| Event ledger store | LocalStack DynamoDB | DynamoDB | ScyllaDB Alternator (DynamoDB API) on a Frostmoln compute node |
| UI | nginx pod | S3 + CloudFront | nginx pod behind Traefik + cert-manager |
| Hostname | `localhost:3000` | `bilcool.areskiftet44.se` | `bilcool.areskiftet44.se` |
| Terraform roots | `stacks/local` | `stacks/database` + `stacks/aws` | `stacks/database` + `stacks/frostmoln/platform` + `stacks/frostmoln/app` |

**Only one of `aws` and `frostmoln` exists at a time.** Switching means applying one and destroying the other, then
changing the Loopia record by hand (CNAME to CloudFront for `aws`, A record to the Frostmoln ingress address for
`frostmoln`). They never talk to each other: each has its own broker and its own secrets.

The database is a separate root (`stacks/database`) that both targets use and that **survives a switch**, so users,
bookings and passkeys are kept (passkeys are bound to the hostname, which does not change). Its public load balancer is a
switch: on for `aws`, off for `frostmoln`.

## 2. Provider interfaces (Go)

Each service picks its concrete implementation at start-up from an environment variable. Unset means today's behaviour,
so existing deployments keep working while the pull requests land.

| Variable | Values | Default | Interface lives in |
|---|---|---|---|
| `MESSAGING_PROVIDER` | `sns-sqs`, `amqp` | `sns-sqs` | `message_broker/pkg/messaging` (`Publisher`, `Subscriber`) |
| `SECRETS_PROVIDER` | `env`, `aws`, `frostmoln` | `env` | `bilcool-lib/pkg/secrets` (`Provider.Get(ctx, key)`) |

| | local | aws | frostmoln |
|---|---|---|---|
| `MESSAGING_PROVIDER` | `sns-sqs` (or `amqp`) | `sns-sqs` | `amqp` |
| `SECRETS_PROVIDER` | `aws` (LocalStack) | `aws` | `frostmoln` |

- **Messaging.** The outbox relay and the inbox worker depend only on the interfaces. `sns-sqs` wraps today's code.
  `amqp` maps a topic to a fanout exchange and a service queue to a durable queue with a dead-letter queue, which is the
  same fan-out shape as SNS -> SQS. Topic and queue names come from the environment; the defaults are today's names.
  The Lambda SQS entrypoints stay as they are (Lambda is handed the messages by AWS).
- **Event ledger.** No interface is needed: ScyllaDB Alternator speaks the DynamoDB API, so `event_ledger` keeps its
  DynamoDB client and only gets a different endpoint (`AWS_ENDPOINT_URL_DYNAMODB`) and credentials on Frostmoln.
- **Secrets.** `env` reads process environment variables (unit tests, and the fallback). `aws` reads Secrets Manager.
  `frostmoln` calls `GET https://api.frostmoln.cloud/api/v1/tenants/<tenant>/secrets/lookup?name=<key>` with the
  workload identity token from the file named by `FROSTMOLN_WORKLOAD_CREDENTIALS_FILE` (re-read on every call, since
  the platform renews it about hourly). Values are fetched once at start-up and cached in memory; they are never logged.

### Secret keys (identical in both environments)

| Key | Used by |
|---|---|
| `bilcool-bookings-database-url` | bookings (runtime and migrate) |
| `bilcool-authentication-database-url` | authentication (runtime and migrate) |
| `bilcool-journal-database-url` | journal (runtime and migrate) |
| `bilcool-event-ledger-dynamodb-access-key-id`, `bilcool-event-ledger-dynamodb-secret-access-key` | event_ledger (only on Frostmoln, for Alternator) |
| `bilcool-authentication-jwt-secret` | authentication |
| `bilcool-authentication-brevo-api-key` | authentication |
| `bilcool-bookings-mapbox-access-token` | bookings |
| `bilcool-<service>-amqp-url` | each service (only where `MESSAGING_PROVIDER=amqp`) |

Lowercase letters and hyphens only, so the names are valid in both services. With `SECRETS_PROVIDER=env` the key
`bilcool-<service>-<name>` is read from the variable `<NAME>` in upper snake case (`DATABASE_URL`), which is what the
services read today.

## 3. No secret is exposed

- **Not in git:** unchanged rule. `values-secrets.yaml`, `db-credentials.env` and the secret `*.tfvars` entries disappear
  because nothing needs them any more.
- **Not in Terraform state:** generated secrets (role passwords, JWT secret, broker passwords) are created with
  ephemeral `random_password` and written with write-only arguments (`secret_string_wo` on AWS, `secret_value_wo` on
  `frostmoln_secret`, `password_wo` on the Postgres role). This needs Terraform 1.11 or later (the repository says 1.9 today).
- **Not in Terraform variables:** secrets that come from outside (Brevo key, Mapbox token) are entered once directly in
  the secrets service; Terraform creates only the empty secret and its access policy. The `pgadmin` password and the
  Frostmoln API key are passed as ephemeral variables from your shell.
- **Not in Lambda or pod configuration:** environment variables hold `SECRETS_PROVIDER` and nothing secret. The Lambda
  role gets `secretsmanager:GetSecretValue` on `bilcool-*`; each Frostmoln service account gets a policy for exactly its
  own secrets.
- **Clean-up:** today's state and Lambda configuration already contain the database URLs and the JWT secret, and S3 keeps
  old state versions. Every secret is therefore **rotated once** after the switch (pull request 17).

## 4. Layout

```
infrastructure/
├── modules/
│   ├── db-bootstrap/          # schema + login role per service, search_path, grants (replaces bootstrap.sh and db-init.sh)
│   ├── messaging-sns-sqs/     # topics, queues, DLQs, subscriptions; names as variables
│   ├── messaging-lavinmq/     # frostmoln_messaging_instance, one vhost, permissions (users are created with fm)
│   ├── scylla-alternator/     # compute instance + volume running ScyllaDB with the DynamoDB-compatible API
│   ├── secrets-aws/           # Secrets Manager entries with the keys above
│   ├── secrets-frostmoln/     # frostmoln_secret entries with the same keys
│   ├── dynamodb/, aws-lambda/, aws-iam/, aws-frontend/, aws-scheduler/
│   ├── frostmoln-network/, frostmoln-postgres/, frostmoln-k8s/
│   └── bilcool-release/       # helm_release of the chart
├── stacks/
│   ├── local/                 # Docker: Postgres, LocalStack, (LavinMQ); bootstrap, messaging, secrets, Helm release
│   ├── database/              # today's frostmoln/terraform  (state key unchanged); kept across a switch
│   ├── aws/                   # today's production/terraform (state key unchanged)
│   ├── frostmoln/platform/    # cluster, node pool, LavinMQ, ScyllaDB node, secrets, workload identity bindings
│   └── frostmoln/app/         # Traefik, cert-manager, bilcool release
└── helm/bilcool/              # application workloads only: services, UI, migrations, seed, ingress
```

The chart loses its Postgres StatefulSet, `db-init` Job, LocalStack Deployment and secret values.

## 5. Pull requests

Small and independently mergeable. "No-op" means production behaves exactly as before; for Terraform it means
`terraform plan` reports no changes (you run it; it needs your credentials).

### A. Go interfaces (no infrastructure change)
1. `message_broker`: `Publisher` / `Subscriber` interfaces and a factory on `MESSAGING_PROVIDER`; SNS/SQS implement them. No-op.
2. Topic and queue names from the environment, defaults unchanged. No-op.
3. `bilcool-lib/pkg/secrets`: `Provider` interface, `env` implementation, factory on `SECRETS_PROVIDER`; the four
   `config` packages read secrets through it. No-op.
4. `aws` secrets implementation (integration test against LocalStack).
5. `amqp` messaging implementation (integration test against a LavinMQ container).
6. `event_ledger`: integration test of the existing DynamoDB store against a ScyllaDB Alternator container; fix any incompatibility it shows.
7. `frostmoln` secrets implementation (API in section 8).

### B. Terraform restructure (no-op)
8. Move the modules to `infrastructure/modules` with `moved` blocks.
9. Move the two roots to `stacks/database` and `stacks/aws`; update `build.yml`, `Taskfile.yml`, READMEs, `CLAUDE.md`.
10. Delete dead material: `production/kubernetes`, the stale `dump-to-local.sh` README section, the unused `env` variable.

### C. Local
11. `stacks/local`: Postgres in Docker + `modules/db-bootstrap`; remove Postgres and `db-init.sh` from the chart.
12. LocalStack in Docker + `modules/messaging-sns-sqs`; remove LocalStack and `init-aws.sh` from the chart.
13. Secrets in LocalStack Secrets Manager (`SECRETS_PROVIDER=aws`); remove `values-secrets.yaml` and the chart's secret values.
14. Helm release from Terraform; `task up` / `task down` / `task down:purge` call the stack; fix `dump-to-minikube.sh` and `task bookings_db`.
15. LavinMQ in Docker, switched on with `-var messaging=amqp`. The services declare their own exchanges and queues at start-up.

### D. aws: secrets and remaining gaps
16. `modules/secrets-aws` and the IAM permission. Lambdas still use their current variables. No-op for behaviour.
17. Lambdas switch to `SECRETS_PROVIDER=aws`; secret variables and sensitive Terraform variables are removed; rotate every secret.
18. `modules/db-bootstrap` on the existing database (import the roles and schemas); delete `bootstrap.sh` and `db-credentials.env`.
19. CI: a real `terraform plan` on pull requests with a read-only role.

### E. frostmoln
20. `stacks/database`: public load balancer behind a variable. `modules/secrets-frostmoln` in `stacks/frostmoln/platform`.
21. `stacks/frostmoln/platform`: cluster and node pool with the `workload-identity` add-on, one
    `frostmoln_workload_identity_binding` per service with a policy allowing `secrets:secrets:read`.
22. `modules/messaging-lavinmq`: managed instance (`require_tls` on), vhost `bilcool`, a user per service with
    permission patterns limited to its own exchanges and queues. The users are created by a Taskfile/CI step with `fm`,
    which pipes each generated password straight into Frostmoln Secrets, so it never reaches Terraform state or a log.
22b. `modules/scylla-alternator`: a compute instance with a data volume in the VPC, ScyllaDB installed by cloud-init
    with Alternator and authorization enabled, volume snapshots as backup. No secret in `user_data`.
23. `stacks/frostmoln/app`: Traefik, cert-manager, the `bilcool` release with `values-frostmoln.yaml`; UI image on `nginx-unprivileged`.
24. CI: a repository variable `DEPLOY_TARGET` (`aws` or `frostmoln`) decides which deploy jobs run on push to `main`;
    replaces `DEPLOY_FROSTMOLN_K8S`. Plus a manual trigger.
25. Runbook for switching target: apply the new one, smoke test on the ingress address, change DNS, destroy the old one.

Groups A and B can start immediately and in parallel. C needs pull requests 3 to 5. D needs 3, 4 and B. E needs A and B.

## 6. Decisions

| Question | Decision |
|---|---|
| `aws` and `frostmoln` | Only one exists at a time, on the same URL |
| Database | One common Frostmoln Postgres for both targets; it survives a switch |
| Local tooling | Terraform; Postgres and LocalStack in Docker beside minikube |
| Messaging | One interface; SNS/SQS on AWS, managed LavinMQ on Frostmoln; chosen by `MESSAGING_PROVIDER` |
| Exchanges and queues in LavinMQ | Each service declares its own at start-up |
| LavinMQ user passwords | Created with `fm` and pushed straight into Frostmoln Secrets |
| Secrets | One interface; AWS Secrets Manager or Frostmoln Secrets; chosen by `SECRETS_PROVIDER`; same keys |
| Event ledger on Frostmoln | ScyllaDB Alternator on a compute node |
| DNS | Unchanged, by hand at Loopia |
| Terraform state | S3 |
| CI | `DEPLOY_TARGET` selects the deploy jobs on push to `main` |

## 7. Risks

- **Pull requests 8 and 9 move the code of the live stacks.** `moved` blocks and a no-change plan are the safeguard. CI never applies Terraform.
- **Pull request 17 changes how production reads every secret.** Mitigation: 16 lands first, and the `env` provider
  remains as a one-variable rollback until the rotation.
- **Cold starts get slower** by one Secrets Manager call per secret (cached for the lifetime of the Lambda instance).
- **The cluster kubeconfig is a sensitive attribute in the platform state.** `stacks/frostmoln/app` authenticates through
  `fm` instead of reading it, but the attribute still exists in state. Pull request 21 checks whether the provider can
  leave it out; if not, it is the one documented exception.
- **ScyllaDB is a database you operate yourself:** upgrades, monitoring and restores are yours, and a single node has no
  redundancy. Alternator is compatible with most, not all, of the DynamoDB API, which is why pull request 6 tests it
  before any infrastructure is built. Its sizing needs (dedicated cores, several GB of RAM) set the compute flavor and cost.
- **Event ledger data does not follow a switch of target** (DynamoDB and ScyllaDB are separate stores). The Postgres
  data does. If the ledger history matters, a copy step is added to the switch runbook.
- **A switch of target has downtime** (DNS propagation, and unsent outbox rows are published by the new target's relay).
  In-flight messages in the old broker are drained before it is destroyed.
- **`password_wo` on the Postgres role** depends on the `cyrilgdn/postgresql` provider version; if it is missing, the role
  passwords are set by a small job that reads the secrets service instead.
- **Delivery semantics differ between SQS and AMQP.** Consumers are already idempotent (`inbox`); the tests in pull
  request 5 cover redelivery and dead-lettering.
- **Frostmoln cannot set `wal_level`**, so `frostmoln` runs the outbox in `polling` mode with a short in-cluster interval.
- **Merging any pull request to `main` redeploys the Lambdas and the UI.**
- **The existing Frostmoln workspace is called `dev` but is production.** Left as is; renaming would recreate resources.

## 8. What was checked, and what was not

Confirmed from `docs.frostmoln.se` (2026-11-11):
- **Messaging:** managed LavinMQ, single node, AMQP 0-9-1. Terraform: `frostmoln_messaging_instance` (`type = "lavinmq"`,
  flavors `mq.gp1.*`, `persistence_mode`, `require_tls`), `frostmoln_messaging_vhost`, `frostmoln_messaging_user` and
  permissions. Private address in the VPC only; AMQPS on 5671 with a certificate from Frostmoln's customer CA
  (`frostmoln_ca_bundle` data source). At most 100 vhosts and 100 users per instance.
- **Secrets:** `frostmoln_secret`; lookup by name works through the endpoint in section 2; access is granted by an
  access policy with `secrets:secrets:read`. Frostmoln also offers External Secrets Operator as a cluster add-on, which
  would copy secrets into Kubernetes Secrets; the plan reads the API directly so that pods hold no secret objects.
- **Workload identity:** cluster add-on `workload-identity`, `frostmoln_workload_identity_binding`, service-account
  annotation `frostmoln.cloud/workload-identity: "true"`, token file in `FROSTMOLN_WORKLOAD_CREDENTIALS_FILE`.
- **Databases:** PostgreSQL 14 to 18 are offered (the existing instance is 16; local uses 18). Private instances are
  reachable from your machine with `fm forward`, which is how Terraform will bootstrap a database without a public listener.
- **No managed DynamoDB-like service** in the documentation (compute, databases, caches, messaging, object storage, registry,
  DNS, secrets, Kubernetes).

Not checked:
- Whether the provider has write-only or ephemeral variants for the LavinMQ user password and the kubeconfig.
- No `terraform`, `helm` or `minikube` in this session, so every plan and local run is yours until the session environment is extended.
- `frostmoln/terraform/environments/dev.tfvars` and `prod.tfvars` are tracked by git although `*.tfvars` is treated as
  secret. I did not open them; please confirm they hold nothing sensitive.
