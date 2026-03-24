# Outbox

A PostgreSQL WAL-based implementation of the [transactional outbox pattern](https://microservices.io/patterns/data/transactional-outbox.html) using logical replication.

## How it works

Instead of polling an outbox table, this library tails the PostgreSQL Write-Ahead Log (WAL) via logical replication. When rows change in the tables you care about, your registered `Action` handlers are called with the affected table information.

The library:

1. Creates (or reuses) a PostgreSQL **publication** for the tables you specify
2. Creates (or reuses) a persistent **replication slot**
3. Starts a background goroutine that streams WAL changes and dispatches them to your handlers

> **Current limitation:** Only `INSERT` operations are handled. `UPDATE`, `DELETE`, and `TRUNCATE` events are received but ignored.

## Prerequisites

Your PostgreSQL instance must have logical replication enabled:

```sql
ALTER SYSTEM SET wal_level = logical;
```

A server restart is required after changing `wal_level`.

## Usage

### 1. Implement the `Action` interface

```go
type myInsertHandler struct{}

func (h myInsertHandler) Execute(table outbox.Table) {
    fmt.Printf("INSERT on %s.%s\n", table.SchemaName, table.TableName)
}
```

### 2. Register actions

```go
actions := outbox.NewActions()
actions.Add(outbox.ActionInsert, myInsertHandler{})
```

### 3. Create a publication

Use `CreatePublication` to create a new publication (idempotent — safe to call if it already exists).

> **Note:** The internal `publication` struct is currently unexported. Until a public constructor is added, `CreatePublication` can only be instantiated from within the `outbox` package itself (e.g. in your own factory function placed in the same package, or via a constructor that will be added to this package).

The intended usage once a constructor is available:

```go
pub := outbox.NewCreatePublication("my_publication", "mydb", []string{"orders", "payments"}, actions)
```

In the meantime, callers in the same module can construct it directly:

```go
pub := outbox.CreatePublication{
    // unexported publication fields set here — only valid inside package outbox
}
```

### 4. Start the outbox

```go
connURL, _ := url.Parse("postgres://user:pass@localhost:5432/mydb")

o, err := outbox.NewOutbox(ctx, connURL, outbox.PgOutputPlugin, pub)
if err != nil {
    log.Fatal(err)
}

stopCh, err := o.StartReplication()
if err != nil {
    log.Fatal(err)
}

// To stop replication:
close(stopCh)
```

`NewOutbox` is a singleton — calling it multiple times returns the same instance.

## Output plugins

| Constant | Plugin | Notes |
|---|---|---|
| `PgOutputPlugin` | `pgoutput` | Built-in, recommended |
| `W2JoutputPlugin` | `wal2json` | Requires the wal2json extension; WAL data is logged but not decoded |

## Integration tests

Tests require a running PostgreSQL instance and are gated behind the `integration` build tag:

```bash
go test -tags integration ./pkg/outbox/...
```