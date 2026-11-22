---
paths:
  - "**/*.go"
---

# Go conventions

- **Pointer to primitive**: Never write helper functions (`strPtr`, `intPtr`, `boolPtr`, etc.) to create pointers to primitive values. Declare a named variable and take its address: `s := "value"; req.Field = &s`.
- **Persistence interface**: Repository implementations accept `DbActions`, not `*sql.DB`. When adding new repository methods keep this pattern. Pass `*sql.Tx` for transactional operations.

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
- **Domain types** in `domain/` define request/response structs with `json` and `validate` struct tags.
- **CQRS split** (bookings and authentication): new use cases go into `application/commands/` (writes) or `application/queries/` (reads), not directly in `application.go`. The `App` interface composes `Commands` and `Queries`; the web layer receives `Commands` and `Queries` (not `App`).

  ```go
  type App interface {
      Commands
      Queries
  }

  // bookings: New(repo); authentication also takes the mail sender, WebAuthn and the JWT secret
  func New(repo domain.BookingsRepository) *Application { ... }
  ```
- **Error mapping**: Add new domain error types to the `NewHttpError` mapper in the `web` package; do not add HTTP logic to the domain or application layer.

## Service package layout

bookings and authentication follow the full layout (see either service's tree). The nested `internal/` ensures service internals cannot be imported by other services. journal and event_ledger differ; see their `CLAUDE.md`.
