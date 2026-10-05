# message_broker

Shared library for the [transactional outbox pattern](https://microservices.io/patterns/data/transactional-outbox.html) and for
consuming the resulting events. It is a Go module used by the services, not a running service.

```
pkg/postgres       outbox table: migrations, Insert, FindAllNewEvents, CreateTable
pkg/outbox/poller  polling relay: outbox table -> SNS
pkg/outbox/sns     SNS publisher and topic cache
pkg/domain         replication relay: tails the WAL (logical replication) -> your Action handlers
pkg/inbox          inbox worker; pkg/inbox/sqs is the SQS subscriber
```

## How events flow

1. A command handler writes its domain data and an `outbox` row **in the same transaction** (`postgres.Insert`).
2. A relay publishes the unsent rows to SNS. There are two relays; the service picks one with `OUTBOX_MODE`:
   - **`polling`** (`pkg/outbox/poller`): `Poll` reads up to 100 rows with `emitted_at IS NULL` (`FOR UPDATE SKIP LOCKED`), publishes them to an SNS
     topic in a batch and marks them emitted, all in one transaction. `RunLoop(ctx, interval)` repeats this in a long-running process.
     On AWS Lambda the outbox Lambdas are triggered by EventBridge Scheduler (every 10 minutes in production), so delivery
     can take up to 10 minutes. Used in production and as the Helm default.
   - **`replication`** (`pkg/domain`): tails the PostgreSQL WAL through a publication and a replication slot and calls your registered
     `Action` handlers. Needs `wal_level=logical`. Used by docker compose, and it is the default when `OUTBOX_MODE` is unset.
3. Consumers (`pkg/inbox`) read SQS queues subscribed to the topics and are idempotent (an `inbox` table).

## The outbox table and its schema

`postgres.OutboxTableName` is `outbox`, with its own migration history in the table `outbox_schema_migrations` (separate from the service's
`schema_migrations`). Columns: `id`, `event_id` (unique), `type`, `correlation_id`, `producer`, `emitted_at` (null until published),
`created_at`, `payload` (jsonb). The migrations are embedded in `pkg/postgres/migrations/`.

`postgres.CreateTable(url)` applies them with dbmate and is idempotent. The bookings, authentication and journal services call it at start-up
(for the HTTP Lambdas: on every cold start).

**All SQL uses the unqualified table name `outbox`. There is no schema-aware code.** The schema comes from the connection's
`search_path`. In production all Postgres services share one database and each service's login role has
`search_path = <service>, public` (set by `infrastructure/frostmoln/bootstrap/bootstrap.sh`), so each service gets its own `outbox`,
`outbox_schema_migrations` and `schema_migrations`. Do not schema-qualify names here.

## Polling relay

```go
poller := poller.New(db, publisher, "bilcool_users") // *sql.DB, sns.Publisher, topic name
err := poller.Poll(ctx)                              // one pass; returns an error if the SNS publish or the commit fails
// or, in a long-running process:
poller.RunLoop(ctx, 10*time.Second)
```

## Replication relay

The relay needs `wal_level=logical` (a server restart is needed after `ALTER SYSTEM SET wal_level = logical`). It creates (or reuses) a
publication for the given tables and a persistent replication slot, then streams changes to your handlers. Only `INSERT` operations
are acted on; `UPDATE`, `DELETE` and `TRUNCATE` are received but ignored.

```go
// 1. Implement domain.Action
type myInsertHandler struct{}

func (h myInsertHandler) Execute(ctx context.Context, table domain.Table) error {
    fmt.Printf("INSERT on %s.%s\n", table.SchemaName, table.TableName)
    return nil
}

// 2. Create an idempotent publication with its actions
pub := domain.NewCreatePublications(
    "my_publication", "mydb", []string{"outbox"},
    map[domain.ActionName]domain.Action{domain.ActionInsert: myInsertHandler{}},
)

// 3. Start the outbox
connURL, _ := url.Parse("postgres://user:pass@localhost:5432/mydb?sslmode=disable")
o, err := domain.NewOutbox(ctx, connURL, domain.PgOutputPlugin, pub)
if err != nil {
    log.Fatal(err)
}
stopCh, err := o.StartReplication(ctx)
if err != nil {
    log.Fatal(err)
}
// To stop replication: close(stopCh)
```

The publication DDL is `CREATE PUBLICATION ... FOR TABLE "outbox"`, which resolves through the connection's `search_path`.

| Constant | Plugin | Notes |
|---|---|---|
| `PgOutputPlugin` | `pgoutput` | Built-in, recommended |
| `W2JoutputPlugin` | `wal2json` | Requires the wal2json extension; WAL data is logged but not decoded |

## Tests

Mocks are generated (`go generate ./...`) and gitignored. Integration tests need Docker (testcontainers via `testing/testdb`) and are gated
behind the `integration` build tag:

```bash
go generate ./...
go test ./...                    # unit tests
go test -tags=integration ./...  # integration tests (replication, SNS publisher, SQS subscriber)
```
