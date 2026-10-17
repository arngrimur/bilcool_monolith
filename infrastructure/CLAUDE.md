# infrastructure

| Path | What |
|---|---|
| `helm/bilcool/` | Helm chart (minikube for local dev, later Frostmoln). See `helm/README.md` |
| `frostmoln/terraform` | Frostmoln VPC, Postgres instance and the public load balancer |
| `frostmoln/bootstrap` | `bootstrap.sh` (creates the database, schemas and roles; writes `db-credentials.env`) and `check-tables.sh` (verifies the migrations per schema) |
| `production/terraform` | The AWS stack (Lambdas, Function URLs, CloudFront + S3, SNS/SQS, DynamoDB, EventBridge Scheduler, IAM) plus `modules/database` (Frostmoln connection strings). State is in the S3 bucket `bilcool-terraform-state` |
| `production/bootstrap`, `production/kubernetes`, `production/PLAN.md` | State bucket, legacy k8s, historical plan |

- **Frostmoln Terraform: use the `dev` workspace and `-var-file=environments/dev.tfvars`.**
- Secrets (`*.tfvars`, `.envrc`, `db-credentials.env`, `helm/values-secrets.yaml`) are gitignored. Never read, commit or paste them.
- Runbooks: `frostmoln/README.md`, `production/terraform/README.md`, `../plan.md`.

## Production (AWS)

Lambdas are exposed through Lambda Function URLs behind CloudFront (the SPA comes from S3, `/api/v1/...` paths go to the Function URLs). Events flow over SNS topics and SQS queues. The outbox is published by scheduled Lambdas (`OUTBOX_MODE=polling`). The site is `bilcool.areskiftet44.se`, DNS is edited by hand at Loopia, and mail is sent through Brevo. The three `aws_lambda_invocation` migrate resources re-run when a database URL changes.

## Databases

- **Production:** **one** Frostmoln PostgreSQL database `bilcool`, with **one schema per service** (`bookings`, `authentication`, `journal`). Each service has its own login role whose default `search_path` is `<service>, public`, so every connection, including dbmate and the outbox, lands in the service's own schema.
  - Connections use `sslmode=require` (the certificate is self-signed, so `verify-full` does not work) through a public Layer-4 load balancer; there is no connection pooler, so do not use `pgbouncer=true`.
  - The Lambdas keep few connections (`MaxOpenConns=2`, no idle) because Postgres has no pooler in front.
  - The database, schemas and roles are created by `frostmoln/bootstrap/bootstrap.sh`, not by Terraform or the service migrations.
  - The old Neon setup is retired and its Terraform module and provider are removed.
- **Local (minikube/Helm):** ONE PostgreSQL 18 instance, database `bilcool`, one schema and login role per service (same shape as production), with `wal_level=logical` so `OUTBOX_MODE=replication` stays testable. Created by `helm/bilcool/files/db-init.sh`.

## Minikube services and ports

Services are ClusterIP-reachable by name inside the cluster (`authentication`, `bookings`, `journal`, `event-ledger`, `ui`, all on :8080 except the UI on :80; the UI nginx proxies `/api/v1/*` to them). On the host only what `task minikube:port-forward` exposes is reachable:

| Target | Host port | Notes |
|---|---|---|
| ui | 3000 | nginx image |
| postgres | 54320 | one instance, database `bilcool`, schema + role per service |
| localstack | 4566 | SNS, SQS, DynamoDB |

Helpers: `minikube:build`, `minikube:build-<service>`, `minikube:build-migrate[-<service>]`, `helm:upgrade`, `minikube:port-forward`, `minikube:kill-port-forward`. The Helm chart defaults to `OUTBOX_MODE=polling`.
