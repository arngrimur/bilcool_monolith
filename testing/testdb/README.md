# testdb

Test helper that starts a throwaway PostgreSQL in Docker and runs a service's migrations against it. It is part of the shared `testing`
module (`github.com/arngrimur/bilcool_monolith/testing`), used by the integration tests of bookings, authentication, journal and
message_broker.

```go
import "github.com/arngrimur/bilcool_monolith/testing/testdb"
```

## Usage

Integration tests are gated by the `integration` build tag and **need Docker** (testcontainers):

```go
//go:build integration

func (s *MySuite) SetupSuite() {
    // migrations.FS is an embed.FS holding the service's dbmate migration files
    s.db = testdb.SetupDatabase(s.T(), migrations.FS, "mydb")
}

func (s *MySuite) TearDownSuite() {
    s.db.TearDown(s.T())
}
```

`SetupDatabase(t, fs, dbName)`:

1. starts a `postgres:18` container (user and password `postgres`) with `wal_level=logical`, `max_wal_senders=5` and
   `max_replication_slots=5`, so replication-based outbox tests work;
2. creates the database `dbName` on a random local port;
3. applies the migrations in `fs` with dbmate (`NewDBMate(t, WithEmbeddedFs(fs), WithWait())`). Pass an empty `embed.FS{}` to skip migrations.

It returns a `SuiteDbIntegration`:

| Field | Description |
|---|---|
| `Db` | `*sql.DB` connected to the new database |
| `ConnString` | the connection URL (`postgres://postgres:postgres@localhost:<port>/<dbName>?sslmode=disable`) |
| `Ctx`, `CancelFunc` | a context for the test run and its cancel function |
| `PostgresContainer` | the running testcontainers container |

It also has `Exec`, `ExecContext` and `QueryContext` shortcuts. `TearDown(t)` cancels the context, closes `Db` and removes the container.

## Options (for `NewDBMate`)

- `WithEmbeddedFs(fs)`: read migrations from an `embed.FS`.
- `WithProjectRoot(root)`: read migrations from the file system instead; `root` is `GitRoot`, `GoModule` or `TestData` (see `path.go`).
- `WithWait()`: wait for the database to accept connections before migrating.

## Notes

- Tests run against a single schema (`public`); the production layout with one schema per service is created by
  `infrastructure/frostmoln/bootstrap/bootstrap.sh` and is not reproduced here.
- To inspect a test database, use `docker ps` to find the container and connect to its mapped port with the URL above.
- The LocalStack helper for the AWS tests lives next to this package in `testing/aws/local_cloud.go`.
