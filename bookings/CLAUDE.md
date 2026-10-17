# bookings

- Follows the full service layout and CQRS split (see `.claude/rules/go-conventions.md`). Extra packages: `internal/pkg/inbox` (SQS consumer) and `internal/pkg/infrastructure/mapbox`.
- Lambda entrypoints: `http, sqs, outbox, migrate`.
- The router has no JWT middleware; `user_ref` comes from the request body.
- The create-booking route is `PUT /api/v1/bookings`, although its swagger annotation says `POST`.
- `bookings.users` is filled only by the `user.created` event from authentication, so a user who signs up and immediately books can get `404 user not found` until the event has been relayed (up to 10 minutes in production).
- Two older mocks under `bookings/` are tracked in git by accident; do not rely on that, regenerate with `go generate ./...`.
- `task bookings_db` opens psql as the bookings role (schema `bookings`).

## Environment variables

| Variable | Description |
|---|---|
| `DATABASE_URL` | Postgres connection string. Production: `postgres://bookings:<pw>@<host>:5432/bilcool?sslmode=require` |
| `OUTBOX_MODE` | `polling` or `replication` (default `replication` when unset) |
| `MAPBOX_ACCESS_TOKEN` | Optional; used to compute distance when a booking ends |
