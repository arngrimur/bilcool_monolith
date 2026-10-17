# bilcool-ui

- **Stack**: React 19 + TypeScript + Vite 8 (`@vitejs/plugin-react`)
- **Commands**: `task ui:dev` (Vite dev server on http://localhost:3000), `task ui:build`, `task ui:test`. CI runs `npm ci`, build and tests.
- **API**: the base is the relative path `/api/v1`. Vite proxies `/api/v1/{users,bookings,events,journal}` to the services (`AUTH_SERVICE_URL`, `BOOK_SERVICE_URL`, `EVENTS_SERVICE_URL`, `JOURNAL_SERVICE_URL`; defaults are localhost:8082, 8081, 8083, 8084).
- **Production**: the SPA is built to S3 behind CloudFront (CI `ui-deploy` on push to `main`: S3 sync with `--delete` + CloudFront invalidation).
- **Kubernetes**: the Docker image (multi-stage build, nginx on port 80, `nginx.conf` proxies the same API paths).
- **Config**: `vite.config.ts`, `tsconfig.app.json`, `nginx.conf`
