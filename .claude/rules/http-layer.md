---
paths:
  - "**/internal/pkg/web/**"
  - "docs/**"
---

# HTTP layer conventions

- **Framework**: Gin v1.12. All APIs are REST under `/api/v1/...`.
- **Swagger**: annotations in `web/router.go` and domain types; regenerate with `task swagger` (bookings and event_ledger). Check handlers against the real route: the bookings create-booking route is `PUT /api/v1/bookings`, although its swagger annotation says `POST`.
- **Graceful shutdown**: all services listen for `SIGINT`/`SIGTERM` and drain on shutdown (30 seconds in bookings).
- **Route groups** (bookings router): `queryRoutes` (GET), `commandRoutes` (PUT/POST/DELETE), `internalRoutes` (`/swagger/*`); `/ping` is registered on the engine. Authentication uses an admin group with JWT and admin middleware instead.
- **Error handling**: use `NewHttpError(err)` to map domain errors to HTTP status codes (bookings, authentication, event_ledger); never return raw errors. A missing user is `404`.
