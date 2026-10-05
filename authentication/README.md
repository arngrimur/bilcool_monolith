# Authentication Service

Passwordless authentication using WebAuthn (passkeys) with an email-token fallback for first-time registration.

---

## Environment Variables

| Variable | Required | Description |
|---|---|---|
| `DATABASE_URL` | yes | PostgreSQL connection string. Local: `postgres://postgres:postgres@localhost:54322/authentication?sslmode=disable`. Production: `postgres://authentication:<password>@<host>:5432/bilcool?sslmode=require`; the `authentication` schema is chosen by the role's `search_path`, not by the URL |
| `JWT_SECRET` | yes | Secret key for signing JWTs (HS256) |
| `WEBAUTHN_RP_ID` | yes | WebAuthn Relying Party ID — the domain name of your application |
| `WEBAUTHN_RP_ORIGINS` | yes | Comma-separated list of allowed WebAuthn origins |
| `FROM_EMAIL` | yes | Sender address for the security code emails. It must be verified in Brevo |
| `BREVO_API_KEY` | yes | Brevo transactional email API key |
| `WEBAUTHN_DISPLAY_NAME` | no | Human-readable name shown to users during passkey prompts (defaults to `BilCool`) |
| `OUTBOX_MODE` | no | `replication` (default) or `polling`. Lambda uses `polling` |
| `RELEASE` | no | Any value switches Gin to release mode |

Email is sent through **Brevo** (`internal/pkg/mail/brevo`, used by both `cmd/main.go` and `cmd/lambda/http/main.go`). An AWS SES
sender exists in `internal/pkg/mail/ses` but is not wired in and cannot be selected by configuration.

AWS credentials (used for publishing the outbox events to SNS) are resolved via the standard AWS SDK chain (env vars, `~/.aws/credentials`, IAM role, etc.). See the [AWS SDK configuration guide](https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/).

---

### JWT_SECRET

A random, high-entropy string used to sign and verify JWTs with HS256.

**Generate:**

```bash
openssl rand -base64 32
```

**References:**
- [RFC 7519 — JSON Web Tokens](https://datatracker.ietf.org/doc/html/rfc7519)
- [golang-jwt/jwt library](https://github.com/golang-jwt/jwt)

---

### WEBAUTHN_RP_ID

The Relying Party ID is the effective domain of your application. The browser enforces that passkeys created for an RP ID can only be used on that domain (and its subdomains).

- For local development: `localhost`
- For production: your bare domain, e.g. `bilcool.areskiftet44.se`

Do **not** include a scheme (`https://`) or path.

**References:**
- [WebAuthn spec — Relying Party ID](https://www.w3.org/TR/webauthn-2/#rp-id)
- [go-webauthn library](https://github.com/go-webauthn/webauthn)

---

### WEBAUTHN_RP_ORIGINS

A comma-separated list of origins the authenticator is allowed to respond to. Each origin must be a full `scheme://host:port` string.

- For local development: `http://localhost:3000`
- For production: `https://bilcool.areskiftet44.se`

The origin must match the `Origin` header sent by the browser during the WebAuthn ceremony. Multiple origins are useful when the same backend serves several frontends.

**References:**
- [WebAuthn spec — Origins](https://www.w3.org/TR/webauthn-2/#dom-publickeycredentialrpentity-id)
- [go-webauthn configuration](https://github.com/go-webauthn/webauthn?tab=readme-ov-file#configuration)

---

### WEBAUTHN_DISPLAY_NAME

A human-readable name for the application shown to the user in passkey dialogs (e.g. `BilCool`).

**References:**
- [WebAuthn spec — Relying Party display name](https://www.w3.org/TR/webauthn-2/#dictionary-rp-credential-params)

---

## Flows

### User Creation

Creating a user is an **admin-only** operation (`POST /api/v1/users` requires a valid JWT of a user with the admin role).

```
Client                          Auth Service                    Database
  |                                   |                              |
  |  POST /api/v1/users               |                              |
  |  { username, email }              |                              |
  |---------------------------------->|                              |
  |                                   |  BEGIN TRANSACTION           |
  |                                   |----------------------------->|
  |                                   |  INSERT users                |
  |                                   |----------------------------->|
  |                                   |  INSERT outbox (event)       |
  |                                   |----------------------------->|
  |                                   |  COMMIT                      |
  |                                   |----------------------------->|
  |  201 { user_ref, username, email }|                              |
  |<----------------------------------|                              |
  |                                   |                              |
  |                                   |  SNS publish "users/created" |
  |                                   |  (outbox relay; polled every |
  |                                   |   10 min on Lambda)          |
```

---

### Login — First-time user (no passkey registered)

```
Client                          Auth Service                Database          Email
  |                                   |                         |               |
  |  POST /api/v1/users/login         |                         |               |
  |  { email }                        |                         |               |
  |---------------------------------->|                         |               |
  |                                   |  SELECT user by email   |               |
  |                                   |------------------------>|               |
  |                                   |  SELECT passkeys        |               |
  |                                   |  (none found)           |               |
  |                                   |------------------------>|               |
  |                                   |  Generate 6-digit token |               |
  |                                   |  INSERT security_tokens |               |
  |                                   |------------------------>|               |
  |                                   |  Send token via Brevo   |               |
  |                                   |---------------------------------------->|
  |  200 { next_step: "verify_token" }|                         |               |
  |<----------------------------------|                         |               |
  |                                   |                         |               |
  |  [User reads token from email]    |                         |               |
  |                                   |                         |               |
  |  POST /api/v1/users/login/token   |                         |               |
  |  { email, token }                 |                         |               |
  |---------------------------------->|                         |               |
  |                                   |  Verify & consume token |               |
  |                                   |------------------------>|               |
  |                                   |  BeginRegistration()    |               |
  |                                   |  INSERT webauthn_session|               |
  |                                   |  (type: registration)   |               |
  |                                   |------------------------>|               |
  |  200 { session_id, options }      |                         |               |
  |<----------------------------------|                         |               |
  |                                   |                         |               |
  |  [Browser prompts: create passkey]|                         |               |
  |                                   |                         |               |
  |  POST /api/v1/users/login/complete|                         |               |
  |  { session_id, credential }       |                         |               |
  |---------------------------------->|                         |               |
  |                                   |  SELECT webauthn_session|               |
  |                                   |  DELETE webauthn_session|               |
  |                                   |  CreateCredential()     |               |
  |                                   |  INSERT passkeys        |               |
  |                                   |------------------------>|               |
  |                                   |  Sign JWT               |               |
  |  200 { token: JWT }               |                         |               |
  |<----------------------------------|                         |               |
```

---

### Login — Returning user (passkey exists)

```
Client                          Auth Service                Database
  |                                   |                         |
  |  POST /api/v1/users/login         |                         |
  |  { email }                        |                         |
  |---------------------------------->|                         |
  |                                   |  SELECT user by email   |
  |                                   |------------------------>|
  |                                   |  SELECT passkeys        |
  |                                   |  (found)                |
  |                                   |------------------------>|
  |                                   |  BeginLogin()           |
  |                                   |  INSERT webauthn_session|
  |                                   |  (type: assertion)      |
  |                                   |------------------------>|
  |  200 { next_step: "passkey_       |                         |
  |         assertion",               |                         |
  |         session_id, options }     |                         |
  |<----------------------------------|                         |
  |                                   |                         |
  |  [Browser prompts: use passkey]   |                         |
  |                                   |                         |
  |  POST /api/v1/users/login/complete|                         |
  |  { session_id, credential }       |                         |
  |---------------------------------->|                         |
  |                                   |  SELECT webauthn_session|
  |                                   |  DELETE webauthn_session|
  |                                   |  ValidateLogin()        |
  |                                   |  (verify passkey sig)   |
  |                                   |------------------------>|
  |                                   |  Sign JWT               |
  |  200 { token: JWT }               |                         |
  |<----------------------------------|                         |
```

---

## HTTP Endpoints

Public:

| Method | Path | Description |
|---|---|---|
| `GET` | `/ping` | Health check |
| `GET` | `/api/v1/users/:id` | Get user by ref |
| `POST` | `/api/v1/users/login` | Begin login — returns `next_step` |
| `POST` | `/api/v1/users/login/token` | Verify email token (starts passkey registration) |
| `POST` | `/api/v1/users/login/complete` | Complete login with WebAuthn credential, returns JWT |
| `POST` | `/api/v1/users/login/reset` | Reset a login (see the router) |

Admin only (JWT middleware plus the admin role):

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/users` | Create a new user |
| `GET` | `/api/v1/users` | List users |
| `GET` | `/api/v1/users/deleted` | List deleted users |
| `DELETE` | `/api/v1/users/:id` | Delete user |
| `POST` | `/api/v1/users/:id/restore` | Restore a deleted user |
| `PATCH` | `/api/v1/users/:id` | Update user |
| `PATCH` | `/api/v1/users/:id/role` | Change the user's role |

The routes are defined in `internal/pkg/web/router.go`.

---

## Running and database

- Long-running service (Docker, Helm): `cmd/main.go`. AWS Lambda: `cmd/lambda/{http,outbox,migrate}`. The Lambda connection pool is
  small (2 open, 0 idle connections); locally it is 5/5.
- Migrations (dbmate) are in `internal/migrations/` and run with `Dockerfile.migrate` locally or the `migrate` Lambda in production.
  Keep them schema-agnostic: in production this service owns the `authentication` schema of the shared `bilcool` database, selected by the
  role's `search_path`, so never schema-qualify names or `SET search_path`.
- The `outbox` table is created by `message_broker` (`CreateTable`) at start-up, not by these migrations.
- New users reach the bookings service only through the `user.created` event (outbox -> SNS -> SQS). On Lambda the outbox is relayed every
  10 minutes, so a brand-new user may not be able to book for up to 10 minutes.
