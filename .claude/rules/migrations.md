---
paths:
  - "**/internal/migrations/**"
  - "**/Dockerfile.migrate"
---

# Migrations

- Migrations are managed with [dbmate](https://github.com/amacneil/dbmate). Files live under `<service>/internal/migrations/` (bookings, authentication, journal; event_ledger uses DynamoDB and has none).
- Always create a new timestamped migration file; never modify an existing one.
- Keep SQL schema-agnostic: no schema-qualified names, no `SET search_path`. In production each service's role (default `search_path` `<service>, public`) puts it in its own schema.
- `message_broker` has its own migration table `outbox_schema_migrations`.
- How they run: Helm runs one `migrate-<service>` Job per service (pre-install/pre-upgrade hook) from each service's `Dockerfile.migrate` image. In production the migrate Lambdas run on every `main` deploy, and the three `aws_lambda_invocation` Terraform resources re-run when a database URL changes.
