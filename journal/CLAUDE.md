# journal

- Does **not** follow the full service layout: only `config`, `inbox`, `persistance/postgres` (note the spelling, not `postgresql`) and `web`. There is no CQRS layer.
- Lambda entrypoints: `http, sqs, migrate`.
- Tests: testify + mock SQS.

## Environment variables

| Variable | Description |
|---|---|
| `DATABASE_URL` | Postgres connection string. Production: `postgres://journal:<pw>@<host>:5432/bilcool?sslmode=require` |
| `API_PORT` | Listen address, default `:8080` |
