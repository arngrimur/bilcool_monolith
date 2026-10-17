# event_ledger

- Consumes events from SQS and stores them in **DynamoDB** (no Postgres, no migrations).
- Packages: `config`, `domain`, `inbox`, `persistance/dynamodb` and `web`. No CQRS layer.
- Lambda entrypoints: `http, sqs`.
- REST API with Gin and Prometheus metrics at `/metrics`. Swagger is regenerated with `task swagger`.
- Tests use **Ginkgo + Gomega**, not testify.

## Environment variables

| Variable | Description |
|---|---|
| `DYNAMO_TABLE_NAME` | DynamoDB table name (required) |
| `API_PORT` | Listen address, default `:8080` |
