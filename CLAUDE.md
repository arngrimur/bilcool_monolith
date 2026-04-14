# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Primary task runner (Taskfile.yml)

```bash
task up                         # Start full stack (Docker Compose + migrations + services)
task down                       # Stop and remove all containers
task test-all                   # Run all unit + integration tests across every module
task swagger                    # Regenerate Swagger/OpenAPI docs
task vendor                     # Re-initialise go.work and vendor all modules
task coverage-all               # Generate merged HTML coverage report in /tmp/coverage/
```

Per-service tasks follow the pattern `<service>:<action>`:

```bash
task bookings:build
task bookings:test
task bookings:integration-test

task authentication:build
task authentication:test
task authentication:integration-test

task message_broker:build
task message_broker:test
task message_broker:integration-test

task event_ledger:build
task event_ledger:test
task event_ledger:integration-test

task ui:build
task ui:test
task ui:dev                     # Vite dev server on :5173
```

### Direct Go commands (run from the relevant module directory)

```bash
go build ./...                  # Build all packages
go test ./...                   # Run unit tests
go test -tags=integration ./... # Run integration tests (require Docker/testcontainers)
go generate ./...               # Regenerate mocks before running tests
go vet ./...                    # Static analysis
```

> **Important**: Always run `go generate ./...` before `go test` — tests depend on generated mocks.

### Database

```bash
docker compose up -d                     # Start all databases and run migrations automatically
docker exec -it bookings-db psql -U postgres -d bookings  # Connect to bookings DB
```

Migrations are managed with [dbmate](https://github.com/amacneil/dbmate). Migration files live under `<service>/internal/migrations/`. `docker-compose.yaml` runs migrations automatically with dedicated `db-migration-*` containers that depend on the database being healthy.

---

## Repository layout

This is a **Go multi-module workspace** (`go.work`). Each service is its own Go module with its own `go.mod`.

```
bilcool_monolith/
├── go.mod                  # Root module (swaggo/swag, Swagger generation)
├── Taskfile.yml            # Build / test orchestration (taskfile.dev)
├── docker-compose.yaml     # Full local stack
├── localstack/             # LocalStack init script (SNS, SQS, DynamoDB setup)
├── docs/                   # Generated Swagger/OpenAPI definitions
│
├── bookings/               # Bookings service (own Go module)
├── authentication/         # Authentication service (own Go module)
├── event_ledger/           # Event Ledger service (own Go module)
├── journal/                # Journal service (own Go module)
├── message_broker/         # Shared messaging library (own Go module)
├── testing/                # Shared test helpers (own Go module)
│
└── ui/bilcool-ui/          # React + TypeScript SPA (Vite)
```

---

## Architecture

BilCool is a booking application built as a monolith structured for future extraction into microservices communicating via gRPC and events, with a React SPA frontend, targeting Kubernetes.

### Services and their databases

| Service | Port (host) | Database | Storage |
|---|---|---|---|
| bookings | 8081 | PostgreSQL :54321 | bookings-db |
| authentication | 8082 | PostgreSQL :54322 | authentication-db |
| journal | — | PostgreSQL :54323 | journal-db |
| event-ledger | 8083 | DynamoDB (LocalStack) | — |
| bilcool-ui (nginx) | 3000 | — | — |

All databases are PostgreSQL 18 with WAL logical replication enabled (`wal_level=logical`).

### Service package layout

Each service follows this internal structure:

```
<service>/
  cmd/main.go                               # Entrypoint: wires dependencies, starts server
  internal/pkg/config/                      # Config loading (env vars)
  internal/pkg/domain/                      # Domain types, business logic, repository interfaces
  internal/pkg/application/                 # Use-case layer
  internal/pkg/application/commands/        # Command handlers (writes)
  internal/pkg/application/queries/         # Query handlers (reads)
  internal/pkg/persistance/postgresql/      # Postgres implementations + DbActions interface
  internal/pkg/event_dispatcher/            # SNS outbox dispatcher
  internal/pkg/web/                         # Gin HTTP router, handlers, Swagger annotations
  internal/migrations/                      # dbmate SQL migration files
  pkg/domain/                               # Public domain types (exported for other modules)
```

The nested `internal/` ensures service internals cannot be imported by other services, enforcing service boundary isolation even within the monolith. Only types under `pkg/` (no `internal`) are intentionally exported.

### Application layer (CQRS pattern)

The `application` package defines an `App` interface composed of `Commands` and `Queries` interfaces. The concrete `Application` struct embeds command and query handler structs:

```go
// application.go — interface composition
type App interface {
    Commands
    Queries
}

// New wires domain → handlers → application
func New(repo domain.BookingsRepository) *Application { ... }
```

Handlers live in `commands/` and `queries/` sub-packages. The web layer receives the `Commands` and `Queries` interfaces (not `App`), keeping routing decoupled from the application layer.

### Domain types pattern

Domain types in `domain/` define request/response structs with `json` and `validate` struct tags. Persistence implementations in `persistance/postgresql/` accept the `DbActions` interface rather than a concrete `*sql.DB`:

```go
// db_actions.go
type DbActions interface {
    QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
    QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
    ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type Transactioner interface {
    BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}
```

This keeps the persistence layer decoupled from the concrete driver and makes it easy to pass `*sql.Tx` for transactional operations.

### Event-driven architecture (outbox pattern)

Services publish events using the **transactional outbox pattern**:

1. **Write**: A command handler writes domain data and an outbox record atomically in one DB transaction.
2. **Replicate**: The `message_broker` module uses PostgreSQL logical replication (`pgoutput`) to tail the outbox table via a replication slot.
3. **Publish**: On commit, the `SnsDispatcher` publishes the event to an SNS topic on LocalStack (or AWS).
4. **Consume**: `journal` and `event_ledger` subscribe to SQS queues that receive messages from the SNS topic and process them.

The `message_broker` module (`message_broker/`) is the shared library that implements this. It is not a running service itself.

```
bookings / authentication
  → [PostgreSQL outbox table]
  → [pg logical replication] → SnsDispatcher → SNS topic
                                                  ↓
                                            SQS queue(s)
                                                  ↓
                                    journal / event_ledger consumers
```

### Authentication details

The `authentication` service implements:
- **JWT** — issued on login, validated as middleware on protected routes
- **WebAuthn/Passkeys** — `go-webauthn` library; begin/verify/complete flow
- **Email** — dual providers: Brevo API and AWS SES (selectable via config)
- **Roles** — role-based access control stored in `roles` table

### Event Ledger details

The `event_ledger` service:
- Consumes events from SQS and stores them in **DynamoDB**
- Exposes a REST API with Gin + Prometheus metrics at `/metrics`
- Uses Ginkgo/Gomega for tests

---

## Testing conventions

### Build tags

Integration tests require real infrastructure (databases, AWS services via testcontainers or LocalStack). They are gated with a build tag:

```go
//go:build integration
```

Run unit tests only: `go test ./...`
Run integration tests: `go test -tags=integration ./...`

### Mock generation

Mocks are generated using `mockgen`. Run `go generate ./...` in a service directory to regenerate. Tests will fail with stale mocks if this step is skipped.

### Shared test helpers (`testing/` module)

The `testing/testdb` package provides:
- `SuiteDbIntegration` — testify suite that spins up a PostgreSQL container via testcontainers and runs dbmate migrations before each test.
- `migrate.go` — programmatic dbmate migration runner.

Import path: `github.com/arngrimur/bilcool_monolith/testing/testdb`

### Test framework by service

| Service | Framework |
|---|---|
| bookings | testify + testcontainers |
| authentication | testify + testcontainers |
| message_broker | testify + testcontainers |
| event_ledger | Ginkgo + Gomega |
| journal | testify + mock SQS |

---

## HTTP layer conventions

- **Framework**: Gin v1.12
- **Swagger**: Annotations in `web/router.go` and domain types; regenerate with `task swagger`
- **Graceful shutdown**: All services listen for `SIGINT`/`SIGTERM` with a 30-second drain
- **Route groups**: `queryRoutes` (GET), `commandRoutes` (PUT/POST/DELETE), `internalRoutes` (`/ping`, `/swagger/*`)
- **Error handling**: Use `NewHttpError(err)` to map domain errors to HTTP status codes; never return raw errors

---

## Environment variables

Variables are loaded from `.env` files per service (via config package) and overridden by the environment. Required variables:

| Variable | Services | Description |
|---|---|---|
| `DATABASE_URL` | all PostgreSQL services | Postgres connection string |
| `AWS_REGION` | all | AWS region (default `eu-north-1`) |
| `AWS_ACCESS_KEY_ID` | all | AWS key (use `test` for LocalStack) |
| `AWS_SECRET_ACCESS_KEY` | all | AWS secret (use `test` for LocalStack) |
| `AWS_ENDPOINT_URL` | all | Override for LocalStack (`http://localhost:4566`) |
| `JWT_SECRET` | authentication | JWT signing secret |
| `FROM_EMAIL` | authentication | Sender address for emails |
| `BREVO_API_KEY` | authentication | Brevo transactional email API key |
| `WEBAUTHN_DISPLAY_NAME` | authentication | WebAuthn relying party display name |
| `WEBAUTHN_RP_ID` | authentication | WebAuthn RP ID (default `localhost`) |
| `WEBAUTHN_RP_ORIGINS` | authentication | Allowed WebAuthn origins |
| `DYNAMO_TABLE_NAME` | event_ledger | DynamoDB table name (default `event-ledger`) |

---

## Frontend (ui/bilcool-ui/)

- **Stack**: React 18 + TypeScript + Vite + SWC
- **Dev server**: `task ui:dev` → http://localhost:5173
- **Production**: Multi-stage Docker build → Nginx serving on :3000
- **Config**: `vite.config.ts`, `tsconfig.app.json`, `nginx.conf`

---

## CI/CD (.github/workflows/)

`build.yml` runs a multi-stage dependency graph:

1. **vendor** — `go work init` + `go work vendor`; result cached across jobs
2. **build-testing** — builds `testing/` helpers
3. **build-message-broker** — build + unit test + integration test
4. **build-bookings** / **build-authentication** / **build-event-ledger** — build + integration test (depend on message-broker)
5. **build-ui** — `npm ci` + `npm run build` + `npm test` (independent)

All Go jobs use [Task](https://taskfile.dev) for orchestration and share the vendored module cache.

---

## Key conventions for AI assistants

- **Service isolation**: Never import a service's `internal/` packages from another service. Only types under `<service>/pkg/` are public.
- **Persistence interface**: Repository implementations accept `DbActions`, not `*sql.DB`. When adding new repository methods keep this pattern.
- **CQRS split**: New use cases go into `commands/` (writes) or `queries/` (reads) sub-packages, not directly in `application.go`.
- **Error mapping**: Add new domain error types to the `NewHttpError` mapper in the `web` package; do not add HTTP logic to the domain or application layer.
- **Migrations**: Always create a new timestamped migration file; never modify an existing one.
- **go generate**: Mocks are checked into source. After changing an interface, run `go generate ./...` and commit the updated mocks.
- **Outbox events**: Domain state changes that need to notify other services must go through the outbox table — never call SNS/SQS directly from a command handler.
