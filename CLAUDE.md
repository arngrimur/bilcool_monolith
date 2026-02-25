# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
go build ./...          # Build all packages
go test ./...           # Run all tests
go test ./path/to/pkg   # Run tests in a specific package
go vet ./...            # Run static analysis

docker compose up -d    # Start Postgres and run migrations
docker compose down     # Stop services
```

Migrations are managed with [dbmate](https://github.com/amacneil/dbmate). Migration files live under `internal/pkg/<service>/migrations/`. The `docker-compose.yaml` runs migrations automatically on `up`.

## Architecture

BilCool is a booking application built as a monolith that is architecturally structured for future extraction into microservices communicating via gRPC and events, with a React SPA frontend, targeting Kubernetes.

### Planned services

- **Booking** — manages reservations
- **Auth** — authentication
- **Users** — user management
- **Journals** — activity/audit journals
- **EventLedger** — event sourcing ledger

Each service owns its own PostgreSQL database.

### Package layout

Each service lives under `internal/pkg/<service>/` and follows this internal structure:

```
internal/pkg/<service>/
  internal/pkg/domain/        # Domain types and business logic
  internal/pkg/persistance/   # Persistence interfaces (DbActions)
  internal/pkg/persistance/postgresql/  # Postgres implementations
  migrations/                 # dbmate SQL migrations
```

The nested `internal/` ensures service internals cannot be imported by other services, enforcing service boundary isolation even within the monolith.

### Domain types pattern

Domain types in `domain/` define request/response structs with JSON and validate tags. Persistence implementations in `persistance/postgresql/` take a `driver.QueryerContext` or `driver.ExecerContext` interface (defined in `persistance/db_actions.go`) rather than a concrete `*sql.DB`, keeping the persistence layer decoupled.