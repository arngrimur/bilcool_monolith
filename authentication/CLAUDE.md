# authentication

- Follows the full service layout and CQRS split (see `.claude/rules/go-conventions.md`). Extra package: `internal/pkg/mail/{brevo,ses}`. `application.New` also takes the mail sender, WebAuthn and the JWT secret.
- Lambda entrypoints: `http, outbox, migrate`.
- **JWT**: issued on login and validated by middleware on this service's protected routes. `JWT_SECRET` is read only by this service.
- **WebAuthn/Passkeys**: `go-webauthn`; begin/verify/complete flow.
- **Email**: sent through **Brevo** (`internal/pkg/mail/brevo`, wired in both the long-running service and the Lambda). An SES sender exists in `internal/pkg/mail/ses` but is not wired in or selectable by config.
- **Roles**: role-based access control stored in the `roles` table. Admin routes use a group with JWT and admin middleware.
- Publishes `user.created`, which is the only way `bookings.users` gets filled.

## Environment variables

| Variable | Description |
|---|---|
| `DATABASE_URL` | Postgres connection string. Production: `postgres://authentication:<pw>@<host>:5432/bilcool?sslmode=require` |
| `OUTBOX_MODE` | `polling` or `replication` (default `replication` when unset) |
| `JWT_SECRET` | JWT signing secret (required) |
| `FROM_EMAIL` | Sender address (required) |
| `MAIL_SENDER` | `brevo` (default) or `log` (development only: login code is logged, not mailed) |
| `BREVO_API_KEY` | Brevo transactional email API key (required unless `MAIL_SENDER=log`) |
| `WEBAUTHN_DISPLAY_NAME` | WebAuthn relying party display name (required) |
| `WEBAUTHN_RP_ID` | WebAuthn RP ID (required, e.g. `localhost` or `bilcool.areskiftet44.se`) |
| `WEBAUTHN_RP_ORIGINS` | Allowed origins, comma separated (required) |
