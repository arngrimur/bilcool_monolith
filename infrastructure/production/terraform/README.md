# BilCool — Production Terraform

Provisions the AWS production stack: 12 Lambda functions (plus their Function URLs), SNS/SQS, DynamoDB, EventBridge Scheduler,
IAM, and an S3 + CloudFront + ACM frontend. The Lambdas run **outside a VPC** (no NAT Gateway) and use the Frostmoln
PostgreSQL database through its public load balancer.

| Module | What it creates |
|---|---|
| `database` | Connection strings for the Frostmoln Postgres: one database `bilcool`, one schema and role per service |
| `lambda` | A Lambda function (and optionally a Function URL), used for every function below |
| `messaging` | SNS topics (`bilcool_users`, `bilcool_bookings`), SQS queues, subscriptions, DLQs |
| `dynamodb` | The event ledger table |
| `scheduler` | EventBridge Scheduler (`rate(10 minutes)`) that runs the outbox Lambdas |
| `frontend` | S3 bucket, CloudFront distribution (SPA + `/api` routed to the Function URLs), ACM certificate |
| `iam` | Lambda roles and the GitHub deploy role |
| `networking` | Empty stub (the Lambdas are not in a VPC) |

Lambdas: bookings `http, sqs, outbox, migrate`; authentication `http, outbox, migrate`; journal `http, sqs, migrate`;
event_ledger `http, sqs`. State lives in the S3 bucket `bilcool-terraform-state` (key `production/terraform.tfstate`); this
stack uses the `default` workspace.

## Prerequisites

- Terraform >= 1.9 and AWS credentials with admin access to the target account (`aws login`).
- The two S3 buckets from the bootstrap step below.
- The Frostmoln database: apply `infrastructure/frostmoln/terraform` and run `infrastructure/frostmoln/bootstrap/bootstrap.sh`
  once (see `infrastructure/frostmoln/README.md`). That produces `db_host` and the role passwords.

## Bootstrap (run once, before everything else)

The main config stores its state in S3 and deploys Lambda functions from S3. Both buckets must exist before `terraform init`
can run, so they are created by a separate config in `../bootstrap/`:

```bash
cd ../bootstrap
terraform init   # uses local state — intentional
terraform apply
cd ../terraform
```

This creates `bilcool-terraform-state` (remote backend, versioned, encrypted) and `bilcool-lambda-artifacts` (where CI
uploads the Lambda ZIPs). The bootstrap state file is gitignored; keep it, never commit it.

## Variables

| Variable | Notes |
|---|---|
| `aws_region` (`eu-north-1`), `environment` (`production`) | defaults |
| `db_host` | **required.** `terraform output db_host` of `infrastructure/frostmoln/terraform` |
| `db_port` | default `5432` |
| `db_passwords` | **required, sensitive.** `map(string)` with `bookings`, `authentication`, `journal`; produced by `bootstrap.sh` |
| `jwt_secret`, `brevo_api_key`, `mapbox_access_token` | **required, sensitive** |
| `from_email`, `webauthn_rp_id`, `webauthn_rp_origins` | **required** |
| `webauthn_display_name` (`BilCool`), `domain_name` (`bilcool.areskiftet44.se`), `frontend_bucket_name` (`bilcool-frontend`) | defaults |
| `lambda_artifacts_bucket` | **required.** The bucket CI uploads to (`bilcool-lambda-artifacts` per `.github/workflows/build.yml`) |

Put **non-secret** values in `terraform.tfvars` (gitignored; start from `terraform.tfvars.example`). Supply **secrets** as
`TF_VAR_*` environment variables, and never write them into a file. A value in `terraform.tfvars` **overrides** the matching
`TF_VAR_*` variable, so do not put a dummy value there. Load the database passwords from the bootstrap output:

```bash
. ../../frostmoln/bootstrap/db-credentials.env        # exports TF_VAR_db_passwords
export TF_VAR_jwt_secret=... TF_VAR_brevo_api_key=... TF_VAR_mapbox_access_token=...
```

## First-time setup

```bash
cp terraform.tfvars.example terraform.tfvars   # fill in the non-secret values
terraform init
terraform plan
terraform apply
```

Check the plan before applying: Lambda environment variables (the secrets) should not change unless you meant to change them.

## Database and migrations

- Every Lambda gets `DATABASE_URL` from `module.database`:
  `postgres://<service>:<password>@<db_host>:5432/bilcool?sslmode=require`. The schema is chosen by the role's `search_path`.
  There is no pooler, so no `pgbouncer=true`.
- Migrations run in the three `*-migrate` Lambdas (dbmate, table `schema_migrations` in each service's schema).
  `aws_lambda_invocation.*_migrate` invokes them, and re-runs them whenever a database URL changes
  (`triggers = { database = sha256(url) }`). CI invokes them after every deploy as well.
- The `outbox` table is not part of the service migrations: the bookings and authentication HTTP Lambdas create it on cold start
  (`message_broker` `CreateTable`).
- The schemas and roles must exist before the migrate Lambdas run (`bootstrap.sh`); Terraform does not create them.

## Custom domain (bilcool.areskiftet44.se)

The domain is served by CloudFront (`modules/frontend`; the certificate lives in `us-east-1` as CloudFront requires). DNS for
`areskiftet44.se` is hosted at Loopia and edited by hand. It is configured in two steps because the certificate needs DNS
validation first.

### Step 1 — Certificate validation

Run `terraform apply`. It waits for the ACM certificate to validate. In a second terminal:

```bash
terraform output acm_validation_records
```

Add the CNAME it prints under `areskiftet44.se` in [Loopia](https://www.loopia.se/loopiakundzon/) (name relative to the zone):

| Type  | Name (relative)         | Value                          | TTL  |
|-------|-------------------------|--------------------------------|------|
| CNAME | `_abc123def456.bilcool` | `_xyz789.acm-validations.aws.` | 3600 |

ACM usually validates within a few minutes and the apply then completes.

### Step 2 — Point the domain at CloudFront

```bash
terraform output cloudfront_domain_name
```

Add a second CNAME in Loopia:

| Type  | Name (relative) | Value                       | TTL  |
|-------|-----------------|-----------------------------|------|
| CNAME | `bilcool`       | `<id>.cloudfront.net`       | 3600 |

(`custom_domain.tf` in this directory is empty; everything domain related is in `modules/frontend`.)

Email is sent through Brevo from `FROM_EMAIL`. The sender domain's DKIM/SPF/DMARC records are managed by hand at Loopia and
are not part of this Terraform.

## Subsequent deployments

Code changes do not need Terraform. Pushing to `main` runs the `lambda-deploy` job (upload ZIPs to `bilcool-lambda-artifacts`,
`update-function-code` for all 12 functions, invoke the three migrate Lambdas) and the `ui-deploy` job (sync the SPA to S3 and
invalidate CloudFront). Run `terraform apply` only for infrastructure changes.

## Outputs

`bookings_function_url`, `authentication_function_url`, `event_ledger_function_url`, `journal_function_url`, the four
`*_http_lambda_arn`, `github_deploy_role_arn`, `cloudfront_domain_name`, `cloudfront_distribution_id` (GitHub secret
`CLOUDFRONT_DISTRIBUTION_ID`), `frontend_bucket_name`, `acm_validation_records`, and the sensitive
`bookings_migrate_url`, `authentication_migrate_url`, `journal_migrate_url`.

## Neon

The Neon module, provider, project and `neon_api_key` variable were removed.
