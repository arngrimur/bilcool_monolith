---
paths:
  - "**/*_test.go"
  - "testing/**"
---

# Testing conventions

- Integration tests require real infrastructure (databases, AWS services via testcontainers or LocalStack) and are gated with `//go:build integration`. Unit: `go test ./...`; integration: `go test -tags=integration ./...`.
- Mocks are generated with `mockgen` (`go.uber.org/mock`) from `//go:generate` directives. They are gitignored (`**/*_mock.go`); run `go generate ./...` in the module directory before testing. CI does this as part of `task <service>:test`.

## Shared helpers (`testing/` module)

Import path: `github.com/arngrimur/bilcool_monolith/testing/testdb`

- `SuiteDbIntegration`, created with `SetupDatabase(t, embeddedMigrationsFS, dbName)`: starts a PostgreSQL container via testcontainers and runs the dbmate migrations. Clean up with `TearDown(t)`.
- `migrate.go`: the programmatic dbmate runner.
- `testing/aws/local_cloud.go`: the LocalStack helper.
