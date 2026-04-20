# Production Setup Plan — Issue #41

## Goals

1. Services run as AWS Lambda functions
2. AWS managed services (SNS, SQS, DynamoDB) replace LocalStack
3. PostgreSQL databases migrate to [Neon](https://neon.tech) serverless Postgres
4. All infrastructure codified in Terraform

---

## Critical Architecture Challenge: Outbox Cannot Run on Lambda

The current outbox pattern uses **PostgreSQL logical replication** (`pglogrepl` + `pgoutput`), requiring a persistent long-lived streaming connection. Lambda is short-lived and stateless — this is fundamentally incompatible.

**Solution: polling-based outbox alongside the existing replication-based outbox, switchable via `OUTBOX_MODE` env var.**

- `OUTBOX_MODE=replication` (default) — existing behaviour, used in Docker Compose / Kubernetes
- `OUTBOX_MODE=polling` — new polling-based outbox, used in Lambda and any deployment that cannot sustain a persistent replication connection

Both modes are kept in the codebase. The existing `message_broker/pkg/domain.Outbox` and `StartReplication()` are untouched.

---

## Lambda Decomposition per Service

Each long-running service splits into separate Lambda functions:

| Service | Lambda Function | Trigger |
|---|---|---|
| bookings | `bookings-http` | API Gateway HTTP API |
| bookings | `bookings-sqs-consumer` | SQS event source mapping |
| bookings | `bookings-outbox-dispatcher` | EventBridge Scheduler (every 15s) |
| authentication | `authentication-http` | API Gateway HTTP API |
| authentication | `authentication-outbox-dispatcher` | EventBridge Scheduler (every 15s) |
| event_ledger | `event-ledger-http` | API Gateway HTTP API |
| event_ledger | `event-ledger-sqs-consumer` | SQS event source mapping |
| journal | `journal-http` | API Gateway HTTP API |
| journal | `journal-sqs-consumer` | SQS event source mapping |

---

## Connection Pooling & Neon

### Problems

- **Cold starts** create new DB connections. At high Lambda concurrency this exhausts Postgres connection limits.
- **Warm containers** reuse the existing `*sql.DB` — pool must be initialized in `main()` before `lambda.Start()` so it survives across invocations.
- **Neon auto-suspends** compute after 5 minutes of inactivity — pool connections go stale; `ConnMaxLifetime` must stay below 5 minutes.
- **Neon connection pooler** (`-pooler.neon.tech`, transaction mode) sits in front of the DB and absorbs Lambda-scale connection churn.

### Changes to `SetupPostgresDatabase()` in each service

Pool settings are added to the existing public function. No `init()` functions — all setup is called explicitly from `main()`:

```go
func SetupPostgresDatabase() *sql.DB {
    psqlDb, err := sql.Open("postgres", config.DatabaseUrl())
    // ...existing ping-retry loop...
    psqlDb.SetMaxOpenConns(5)
    psqlDb.SetMaxIdleConns(5)
    psqlDb.SetConnMaxLifetime(4 * time.Minute)
    psqlDb.SetConnMaxIdleTime(2 * time.Minute)
    return psqlDb
}
```

### Neon connection string format

```
postgres://user:pass@ep-xxx-pooler.eu-north-1.aws.neon.tech/dbname?sslmode=require&pool_mode=transaction
```

Transaction-mode pooling is compatible with the current codebase — all queries use `QueryContext`/`ExecContext` with no session-scoped state.

---

## Code Changes

### 1. New migration: outbox status columns (bookings & authentication)

New timestamped dbmate migration in each service's `internal/migrations/`:

```sql
ALTER TABLE outbox ADD COLUMN processed_at TIMESTAMPTZ;
ALTER TABLE outbox ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'pending';
CREATE INDEX idx_outbox_status ON outbox(status) WHERE status = 'pending';
```

### 2. New `message_broker/pkg/outbox/poller/` package

Replaces `StartReplication()` when `OUTBOX_MODE=polling`:

```go
type Poller struct { ... }

// Poll runs a single cycle — used by Lambda (one invocation = one poll)
func (p *Poller) Poll(ctx context.Context) error

// RunLoop runs Poll on a ticker — used by long-running server with OUTBOX_MODE=polling
func (p *Poller) RunLoop(ctx context.Context, interval time.Duration)
```

`Poll` implementation:
1. `BEGIN` transaction
2. `SELECT id, payload FROM outbox WHERE status='pending' LIMIT 100 FOR UPDATE SKIP LOCKED`
3. Publish each record to SNS
4. `UPDATE outbox SET status='processed', processed_at=NOW() WHERE id IN (...)`
5. `COMMIT`

### 3. `OUTBOX_MODE` config in each service

```go
// Each service's config/config.go
func OutboxMode() string {
    if mode, ok := os.LookupEnv("OUTBOX_MODE"); ok {
        return mode
    }
    return "replication"
}
```

### 4. Switch in each service's `cmd/main.go`

```go
switch config.OutboxMode() {
case "polling":
    poller := outboxpoller.New(psqlDb, snsPublisher)
    go poller.RunLoop(ctx, 15*time.Second)
default: // "replication"
    outbox, _ := soutbox.NewOutbox(ctx, dbUrl, soutbox.PgOutputPlugin, ...)
    closer, _ := outbox.StartReplication(ctx)
    defer close(closer)
}
```

### 5. Expose `Engine()` on each `HttpRouter`

Lambda entrypoints need access to the `*gin.Engine` for `ginadapter`. Add:

```go
func (h *HttpRouter) Engine() *gin.Engine { return h.router }
```

### 6. New Lambda entrypoints

Each service gets a `cmd/lambda/` subtree alongside the existing `cmd/main.go`.

**HTTP Lambda** (example: bookings):

```go
// bookings/cmd/lambda/http/main.go
func main() {
    db := postgresql.SetupPostgresDatabase()
    repo := postgresql.NewBookingsRepository(db)
    app := application.New(repo)
    ginLambda := ginadapter.New(web.NewRouter(app.GetBookingsHandler, app.UpdateBookingsHandler).Engine())

    lambda.Start(func(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
        return ginLambda.ProxyWithContext(ctx, req)
    })
}
```

**SQS Lambda** (example: bookings):

```go
// bookings/cmd/lambda/sqs/main.go
func main() {
    db := postgresql.SetupPostgresDatabase()
    repo := postgresql.NewBookingsRepository(db)
    handler := bookinginbox.NewEventHandler(nil, repo) // messages come from event, not subscriber

    lambda.Start(func(ctx context.Context, event events.SQSEvent) (events.SQSBatchResponse, error) {
        // process event.Records, return batch item failures
    })
}
```

**Outbox Lambda** (example: bookings):

```go
// bookings/cmd/lambda/outbox/main.go
func main() {
    db := postgresql.SetupPostgresDatabase()
    awsCfg, _ := awsconfig.LoadDefaultConfig(context.Background())
    publisher := sns.NewPublisher(awsCfg)
    poller := outboxpoller.New(db, publisher)

    lambda.Start(func(ctx context.Context) error {
        return poller.Poll(ctx)
    })
}
```

No `init()` functions anywhere — all wiring happens in `main()`.

### 7. New Go dependencies (per service `go.mod`)

```
github.com/aws/aws-lambda-go v1.47+
github.com/awslabs/aws-lambda-go-api-proxy v0.16+
```

### 8. New Taskfile Lambda build targets

```yaml
bookings:lambda-http-build:
  cmds:
    - cd bookings && GOOS=linux GOARCH=arm64 go build -o bootstrap ./cmd/lambda/http
    - zip -j dist/bookings-http.zip bookings/bootstrap
```

One target per Lambda function, outputting ZIP artifacts to `dist/`.

---

## What Does NOT Change

- Domain types, application layer, repository interfaces
- Gin handler implementations
- SQS subscriber and SNS publisher packages
- DynamoDB client
- Authentication JWT/WebAuthn logic
- All existing tests
- `cmd/main.go` entrypoints for Docker Compose / Kubernetes deployments

---

## Terraform Structure

Location: `infrastructure/production/terraform/`

```
terraform/
├── providers.tf               # AWS + Neon providers
├── variables.tf               # Region, environment, Neon API key, etc.
├── outputs.tf                 # API Gateway URLs, Lambda ARNs
├── main.tf                    # Root: calls all modules
├── terraform.tfvars.example
└── modules/
    ├── iam/                   # Lambda execution roles + least-privilege policies
    ├── neon/                  # Neon projects + 3 databases (bookings, authentication, journal)
    ├── messaging/             # SNS topics + SQS queues + DLQs + SNS→SQS subscriptions
    ├── dynamodb/              # event-ledger DynamoDB table
    ├── lambda/                # Reusable module: function + log group + IAM attachment
    ├── api_gateway/           # HTTP API v2 + routes + Lambda integrations + stage
    └── scheduler/             # EventBridge Scheduler for outbox dispatcher Lambdas
```

### Key Terraform resources

| Resource | Purpose |
|---|---|
| `aws_lambda_function` ×9 | One per Lambda function, `arm64` / `provided.al2023` |
| `aws_apigatewayv2_api` | Single HTTP API routing to 4 HTTP Lambdas |
| `aws_lambda_event_source_mapping` ×3 | SQS → Lambda for bookings, event-ledger, journal consumers |
| `aws_scheduler_schedule` ×2 | EventBridge every 15s for bookings + authentication outbox |
| `aws_sns_topic` ×2 | `bilcool_bookings`, `bilcool_users` |
| `aws_sqs_queue` ×6 | 3 service queues + 3 DLQs |
| `aws_sns_topic_subscription` ×3 | SNS → SQS fan-out |
| `aws_dynamodb_table` | event-ledger |
| `neon_project` + `neon_database` + `neon_role` ×3 | bookings, authentication, journal databases |
| `aws_iam_role` + policies | Least-privilege execution role per service group |
| `aws_s3_bucket` + `aws_s3_object` | Lambda ZIP artifact storage |
| `aws_cloudwatch_log_group` ×9 | One per Lambda |

**Neon Terraform provider:** `kislerdm/neon`

---

## Implementation Order

1. **Migrations** — add outbox `status`/`processed_at` columns (non-breaking, backward-compatible)
2. **Outbox poller package** — `message_broker/pkg/outbox/poller/` with tests
3. **`OUTBOX_MODE` config** — add to each service's `config.go`
4. **Switch in `cmd/main.go`** — replication vs polling based on env var
5. **DB setup** — add pool settings to `SetupPostgresDatabase()` in each service
6. **Expose `Engine()`** — on each `HttpRouter`
7. **Lambda entrypoints** — `cmd/lambda/` per service (http + sqs + outbox as applicable)
8. **Go module updates** — add `aws-lambda-go` + `aws-lambda-go-api-proxy` deps
9. **Taskfile Lambda build targets**
10. **Terraform** — providers, modules, root config (can run in parallel with 1–9)
11. **CI additions** — build Lambda ZIPs, `terraform plan` on PR, `terraform apply` on merge to main

---

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Polling outbox introduces up to 15s event latency vs near-realtime WAL streaming | Acceptable; tune EventBridge schedule frequency down to 5s if needed |
| Neon transaction-mode pooling incompatible with `SET SESSION` / cross-TX prepared statements | Current codebase uses plain `QueryContext`/`ExecContext` — compatible |
| Lambda cold starts add ~200–500ms latency | Use provisioned concurrency on authentication Lambda if latency is critical |
| SQS event source mapping delivers batches; `ProcessMessages` already accepts slices | No change needed; set `BatchSize` in Terraform |
| Neon auto-suspend on inactive compute | Use paid Neon plan for production; `ConnMaxLifetime < 5min` as safety net |
