# BilCool UI

Single-page app for BilCool (React 19, TypeScript, Vite 8 with `@vitejs/plugin-react`, Tailwind). It talks to the backend only through relative
`/api/v1/...` paths, so the same build works locally, in Kubernetes and behind CloudFront.

## Scripts

Run these in `ui/bilcool-ui/` (or use `task ui:build`, `task ui:test`, `task ui:dev` from the repo root):

| Script | What it does |
|---|---|
| `npm run dev` | Vite dev server on http://localhost:3000 |
| `npm run build` | `tsc -b && vite build` |
| `npm run preview` | Serve the production build locally |
| `npm test` | Unit tests (`vitest run`); `npm run test:watch` to watch |
| `npm run lint` | ESLint (no warnings allowed) |

## API proxy (dev server)

`vite.config.ts` proxies the API paths to the services. Override the targets with environment variables:

| Path | Variable | Default |
|---|---|---|
| `/api/v1/users` | `AUTH_SERVICE_URL` | `http://localhost:8082` |
| `/api/v1/bookings` | `BOOK_SERVICE_URL` | `http://localhost:8081` |
| `/api/v1/events` | `EVENTS_SERVICE_URL` | `http://localhost:8083` |
| `/api/v1/journal` | `JOURNAL_SERVICE_URL` | `http://localhost:8084` |

Passkeys need the WebAuthn origin to match: locally the authentication service uses `WEBAUTHN_RP_ID=localhost` and
`WEBAUTHN_RP_ORIGINS=http://localhost:3000`.

## Deployment

- **Production (AWS):** CI (`ui-deploy`, on push to `main`) syncs the build to the S3 bucket and invalidates CloudFront. CloudFront serves the SPA
  (a function rewrites non-file paths to `/index.html`) and routes `/api/v1/...` to the Lambda Function URLs. Infrastructure:
  `infrastructure/production/terraform/modules/frontend`.
- **Kubernetes / Docker:** the `Dockerfile` builds the app (Node 22) and serves it with nginx on port 80; `nginx.conf` proxies the same API
  paths to the services. The Helm chart template is `infrastructure/helm/bilcool/templates/ui.yaml`.

## Source layout

`src/` has `api` (REST clients), `components`, `hooks`, `i18n`, `lib` and `test`. PWA assets are in `public/` (`manifest.webmanifest`);
`scripts/generate-icons.py` generates the icons.
