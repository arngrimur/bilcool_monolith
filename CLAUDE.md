# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

More specific guidance loads on demand:
- `<module>/CLAUDE.md` for each service (bookings, authentication, journal, event_ledger, message_broker), `infrastructure/CLAUDE.md`, `ui/bilcool-ui/CLAUDE.md`
- `.claude/rules/` for Go conventions, migrations, testing and the HTTP layer (scoped by path)

## Key conventions

- **Go version**: This project uses **Go 1.26**.
- **Service isolation**: Never import a service's `internal/` packages from another service. Only types under `<service>/pkg/` are public.
- **Schema-agnostic SQL**: In production each service's role puts it in its own schema. Never schema-qualify table names and never `SET search_path` in code or migrations. Tables such as `users`, `positions`, `inbox` and `outbox` exist in several schemas on purpose.
- **Outbox events**: Domain state changes that need to notify other services must go through the outbox table; never call SNS/SQS directly from a command handler.
- **go generate**: Mocks are generated locally and gitignored. Always run `go generate ./...` before `go test`, and after changing an interface.
- **Merging to `main` is a production deploy** (Lambdas + UI).
- **Secrets**: never read, print or commit `.envrc`, `*.tfvars`, `values-secrets.yaml` or `db-credentials.env`.

## Commands

### Primary task runner (Taskfile.yml)

Run `task --list` for every task.

> `task up` needs `infrastructure/helm/values-secrets.yaml` (gitignored; `cp infrastructure/helm/values-secrets.example.yaml ...`). Dev uses `MAIL_SENDER=log`: the login code is written to the authentication pod log (`kubectl logs deploy/authentication | grep "security token"`), no Brevo account needed. See `infrastructure/helm/README.md`.

### Go (run from the relevant module directory)

> `go.work` and `vendor/` are gitignored and do not exist in a fresh clone. Run `task vendor` first.

---

## Architecture

BilCool is a booking application built as a monolith structured for future extraction into microservices. Services talk to each other through events (SNS/SQS); gRPC is a possible future step and is not used today. All HTTP APIs are REST (Gin).

| Target | Used for | How |
|---|---|---|
| **AWS Lambda** | **Production (today)** | `cmd/lambda/*` entrypoints, Terraform in `infrastructure/production/terraform`, CI `lambda-deploy` |
| Kubernetes (Helm, minikube) | Local development | `infrastructure/helm`, `task up` |

`<service>/cmd/main.go` is the long-running variant used by the Docker images and Helm. Lambda entrypoints (`<service>/cmd/lambda/`): bookings `http, sqs, outbox, migrate`; authentication `http, outbox, migrate`; journal `http, sqs, migrate`; event_ledger `http, sqs`.

Services publish events with the **transactional outbox pattern** (details in `message_broker/CLAUDE.md`): a command handler writes domain data and an outbox row in one transaction, a relay publishes it to SNS, and consumers read SQS queues idempotently (an `inbox` table, `ON CONFLICT DO NOTHING`).

Databases: one PostgreSQL database `bilcool` with **one schema and login role per service** (`bookings`, `authentication`, `journal`), locally and in production; event_ledger uses DynamoDB. Details in `infrastructure/CLAUDE.md`.

Config is read from the process environment by each service's `internal/pkg/config` package. Secrets (database URLs, JWT secret, Brevo key, Mapbox token) go through `bilcool-lib/pkg/secrets`: `SECRETS_PROVIDER` picks the store (only `env` today, the default), keys are `bilcool-<service>-<name>` in every store, and the `env` provider reads the old variable names (`DATABASE_URL`, …). There is no `.env` loading in the Go code; Helm and Terraform set the variables. `RELEASE` (any value) switches Gin to release mode. The AWS SDK reads `AWS_REGION`, `AWS_ENDPOINT_URL`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`; locally use `test` credentials and `http://localhost:4566` (LocalStack); production uses `eu-north-1`.
