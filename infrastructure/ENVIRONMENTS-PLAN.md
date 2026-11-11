# Plan: three environments, all as infrastructure as code

> **Status: proposal for review, revision 2 (2026-11-11). Nothing is implemented.**
> Revision 2 applies your answers: two independent production environments, Terraform everywhere, LocalStack in Docker,
> LavinMQ on Frostmoln, provider interfaces selected by environment variables, secrets only in the secrets services with
> the same keys in both environments, DNS unchanged, state in S3, small pull requests.

## 1. Target

| | **local** | **aws** (production 1, as today) | **frostmoln** (production 2) |
|---|---|---|---|
| Runtime | minikube, Helm chart | AWS Lambda + Function URLs | Frostmoln Kubernetes, Helm chart |
| Database | Postgres 18 in Docker beside minikube | its own Frostmoln Postgres (the existing instance, public L4 load balancer) | its own, new Frostmoln Postgres (private VPC address) |
| Messaging | LocalStack SNS/SQS in Docker (LavinMQ in Docker as a switch) | AWS SNS/SQS | LavinMQ (AMQP 0-9-1) |
| Secrets | LocalStack Secrets Manager | AWS Secrets Manager | Frostmoln secrets service |
| Event ledger store | LocalStack DynamoDB | DynamoDB | Postgres schema `event_ledger` (assumed, see 6.2) |
| UI | nginx pod | S3 + CloudFront | nginx pod behind Traefik + cert-manager |
| Hostname | `localhost:3000` | `bilcool.areskiftet44.se` | a second hostname (see 6.1) |
| Terraform roots | `stacks/local` | `stacks/aws/database`, `stacks/aws/app` | `stacks/frostmoln/platform`, `stacks/frostmoln/app` |

`aws` and `frostmoln` share nothing at runtime: separate databases, brokers, secrets, users and hostnames. The only common
things are the git repository, the container images and the S3 bucket that holds Terraform state.

## 2. Provider interfaces (Go)

Each service picks its concrete implementation at start-up from an environment variable. Unset means today's behaviour,
so existing deployments keep working while the pull requests land.

| Variable | Values | Default | Interface lives in |
|---|---|---|---|
| `MESSAGING_PROVIDER` | `sns-sqs`, `amqp` | `sns-sqs` | `message_broker/pkg/messaging` (`Publisher`, `Subscriber`) |
| `SECRETS_PROVIDER` | `env`, `aws`, `frostmoln` | `env` | `bilcool-lib/pkg/secrets` (`Provider.Get(ctx, key)`) |
| `EVENT_STORE` | `dynamodb`, `postgres` | `dynamodb` | `event_ledger` (the existing `EventStore` / `EventQuerier`) |

| | local | aws | frostmoln |
|---|---|---|---|
| `MESSAGING_PROVIDER` | `sns-sqs` (or `amqp`) | `sns-sqs` | `amqp` |
| `SECRETS_PROVIDER` | `aws` (LocalStack) | `aws` | `frostmoln` |
| `EVENT_STORE` | `dynamodb` | `dynamodb` | `postgres` |

- **Messaging.** The outbox relay and the inbox worker depend only on the interfaces. `sns-sqs` wraps today's code.
  `amqp` maps a topic to a fanout exchange and a service queue to a durable queue with a dead-letter queue, which is the
  same fan-out shape as SNS -> SQS. Topic and queue names come from the environment; the defaults are today's names.
  The Lambda SQS entrypoints stay as they are (Lambda is handed the messages by AWS).
- **Secrets.** `env` reads process environment variables (unit tests, and the fallback). `aws` reads Secrets Manager.
  `frostmoln` reads the Frostmoln secrets API using the pod's workload identity token. Values are fetched once at
  start-up and cached in memory; they are never logged.

### Secret keys (identical in both environments)

| Key | Used by |
|---|---|
| `bilcool-bookings-database-url` | bookings (runtime and migrate) |
| `bilcool-authentication-database-url` | authentication (runtime and migrate) |
| `bilcool-journal-database-url` | journal (runtime and migrate) |
| `bilcool-event-ledger-database-url` | event_ledger (only where `EVENT_STORE=postgres`) |
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
│   ├── messaging-amqp/        # exchanges, queues, DLQs, bindings (cloudamqp/lavinmq provider)
│   ├── secrets-aws/           # Secrets Manager entries with the keys above
│   ├── secrets-frostmoln/     # frostmoln_secret entries with the same keys
│   ├── dynamodb/, aws-lambda/, aws-iam/, aws-frontend/, aws-scheduler/
│   ├── frostmoln-network/, frostmoln-postgres/, frostmoln-k8s/
│   └── bilcool-release/       # helm_release of the chart
├── stacks/
│   ├── local/                 # Docker: Postgres, LocalStack, (LavinMQ); bootstrap, messaging, secrets, Helm release
│   ├── aws/database/          # today's frostmoln/terraform  (state key unchanged)
│   ├── aws/app/               # today's production/terraform (state key unchanged)
│   ├── frostmoln/platform/    # VPC, Postgres, cluster, LavinMQ, secrets
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
6. `event_ledger`: factory on `EVENT_STORE`, Postgres store, migrations and a migrate image.
7. `frostmoln` secrets implementation (after the API check in 6.4).

### B. Terraform restructure (no-op)
8. Move the modules to `infrastructure/modules` with `moved` blocks.
9. Move the two roots to `stacks/aws/database` and `stacks/aws/app`; update `build.yml`, `Taskfile.yml`, READMEs, `CLAUDE.md`.
10. Delete dead material: `production/kubernetes`, the stale `dump-to-local.sh` README section, the unused `env` variable.

### C. Local
11. `stacks/local`: Postgres in Docker + `modules/db-bootstrap`; remove Postgres and `db-init.sh` from the chart.
12. LocalStack in Docker + `modules/messaging-sns-sqs`; remove LocalStack and `init-aws.sh` from the chart.
13. Secrets in LocalStack Secrets Manager (`SECRETS_PROVIDER=aws`); remove `values-secrets.yaml` and the chart's secret values.
14. Helm release from Terraform; `task up` / `task down` / `task down:purge` call the stack; fix `dump-to-minikube.sh` and `task bookings_db`.
15. LavinMQ in Docker + `modules/messaging-amqp`, switched on with `-var messaging=amqp`.

### D. aws: secrets and remaining gaps
16. `modules/secrets-aws` and the IAM permission. Lambdas still use their current variables. No-op for behaviour.
17. Lambdas switch to `SECRETS_PROVIDER=aws`; secret variables and sensitive Terraform variables are removed; rotate every secret.
18. `modules/db-bootstrap` on the existing database (import the roles and schemas); delete `bootstrap.sh` and `db-credentials.env`.
19. CI: a real `terraform plan` on pull requests with a read-only role.

### E. frostmoln
20. `stacks/frostmoln/platform`: VPC, Postgres, `db-bootstrap` with the fourth schema, `modules/secrets-frostmoln`.
21. Cluster, node pool, workload identity and the per-service secret policies.
22. LavinMQ and `modules/messaging-amqp`.
23. `stacks/frostmoln/app`: Traefik, cert-manager, the `bilcool` release with `values-frostmoln.yaml`; UI image on `nginx-unprivileged`.
24. CI: deploy job for `frostmoln`, independent of the `aws` deploy (see 6.5).

Groups A and B can start immediately and in parallel. C needs A.3-A.5. D needs A.3-A.4 and B. E needs A, and B for the modules.

## 6. Still open

1. **Hostname for `frostmoln`.** It needs its own name (passkeys are bound to the hostname), added by hand at Loopia.
   Proposal: `bilcool-fm.areskiftet44.se`. Please name it.
2. **Event ledger on Frostmoln.** You did not answer this one. The plan assumes a Postgres store behind `EVENT_STORE`,
   in the same interface style as messaging and secrets. Say so if you want something else.
3. **LavinMQ: managed or self-hosted?** I could not find a LavinMQ or broker resource in what is publicly indexed of the
   Frostmoln Terraform provider, so I cannot confirm it is offered as a managed service. Pull request 22 starts by reading
   the provider schema. If there is no managed resource, LavinMQ runs in the cluster as a Helm release, still from
   Terraform. The Go code is the same either way (plain AMQP 0-9-1).
4. **Frostmoln secrets API from a pod.** What I found: `frostmoln_secret` supports `secret_value_wo`; a pod can call the
   secrets lookup API with its projected service-account token; access policies match secrets by UUID. The exact endpoint
   and whether lookup by name works must be confirmed before pull request 7.
5. **When does CI deploy to which environment?** Proposal: `aws` stays as today (push to `main`); `frostmoln` deploys on
   tags behind the existing `DEPLOY_FROSTMOLN_K8S` variable, plus a manual trigger for either.
6. **Bootstrapping a private database.** Terraform on your machine cannot reach a Postgres that only has a VPC address.
   Proposal: a public L4 listener restricted to your IP while bootstrapping, if the `allowed_cidrs` bug is fixed;
   otherwise a one-off Job in the cluster.

## 7. Risks

- **Pull requests 8 and 9 move the code of the live stacks.** `moved` blocks and a no-change plan are the safeguard. CI never applies Terraform.
- **Pull request 17 changes how production reads every secret.** Mitigation: 16 lands first, and the `env` provider
  remains as a one-variable rollback until the rotation.
- **Cold starts get slower** by one Secrets Manager call per secret (cached for the lifetime of the Lambda instance).
- **`password_wo` on the Postgres role** depends on the `cyrilgdn/postgresql` provider version; if it is missing, the role
  passwords are set by a small job that reads the secrets service instead.
- **The cluster kubeconfig is a sensitive value in state** according to the provider's guide. `stacks/frostmoln/app`
  will authenticate with the `fm` exec plugin rather than read it, but the value still sits in the platform state.
  This is the one known exception to section 3 and needs a closer look in pull request 21.
- **Delivery semantics differ between SQS and AMQP.** Consumers are already idempotent (`inbox`); the tests in pull
  request 5 cover redelivery and dead-lettering.
- **Frostmoln cannot set `wal_level`**, so `frostmoln` runs the outbox in `polling` mode with a short in-cluster interval.
- **Merging any pull request to `main` redeploys the Lambdas and the UI.**
- **The existing Frostmoln workspace is called `dev` but is production 1.** Left as is; renaming would recreate resources.

## 8. What I could not check

- The full Frostmoln provider schema and API reference (only search snippets were reachable from this session).
- No `terraform`, `helm` or `minikube` in this session, so every plan and local run is yours until the session environment is extended.
- `frostmoln/terraform/environments/dev.tfvars` and `prod.tfvars` are tracked by git although `*.tfvars` is treated as
  secret. I did not open them; please confirm they hold nothing sensitive.
