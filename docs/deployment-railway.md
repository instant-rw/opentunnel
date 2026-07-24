# Railway and Cloudflare deployment

## Railway

1. Create a Railway project and add a PostgreSQL service (or use an external DB).
2. Deploy the **API** service from this repository. Set the service config-as-code
   path to `backend/railway.json` (repo root as the working directory so
   `backend/Dockerfile` can copy `shared/`). Healthcheck is `/readyz`.
3. Set `DATABASE_URL` and copy non-secret values from
   `backend/.env.production.example`. Typical production values:

   - `OPENTUNNEL_BASE_URL=https://opts.ink`
   - `OPENTUNNEL_FRONTEND_URL=https://app.opts.ink`
   - `OPENTUNNEL_PUBLIC_HOST=opts.ink`
   - `OPENTUNNEL_COOKIE_DOMAIN=.opts.ink`
   - `OPENTUNNEL_CORS_ORIGINS=https://app.opts.ink`
   - `OPENTUNNEL_SECURE_COOKIES=true`

4. Deploy the **frontend** as a second service. Set config-as-code to
   `frontend/railway.json`. Set build-time `VITE_API_URL=https://opts.ink/api/v1`
   (see `frontend/.env.production.example`). Attach the dashboard hostname
   (e.g. `app.opts.ink`).
5. Generate a Railway domain for the API first and verify `/healthz` and
   `/readyz`. Migrations run at API startup, so deploy only one migration-capable
   version at a time.
6. In API networking settings, add both `opts.ink` and `*.opts.ink` as custom
   domains. Copy Railway's current CNAME and ACME validation targets; do not
   infer or reuse values from another environment.

Railway must route WebSocket upgrades without a path rewrite. The service listens
on Railway's injected `PORT` through `OPENTUNNEL_HTTP_ADDR`; set it to `:${PORT}`
if Railway does not map port 8080 automatically.

## Cloudflare DNS

Create the records Railway requests:

- `@` CNAME to the Railway API target (control plane + tunnel hosts).
- `*` CNAME to the Railway API target.
- `app` (or your dashboard host) CNAME to the frontend service target.
- Railway's `_acme-challenge` CNAME for the wildcard certificate.

Keep the ACME challenge record **DNS only**. Start the application CNAME records
as DNS only until Railway shows valid certificates. They may then be proxied
through Cloudflare. Set Cloudflare SSL/TLS mode to **Full (strict)** after
Railway's certificates are active (use **Full** only during initial diagnosis).
Never use Flexible mode: it breaks secure-cookie and origin-security assumptions.

Cloudflare must allow WebSockets. Do not cache `/api/*`, `/tunnel`, `/healthz`,
or `/readyz`. Forward the original `Host` header so the server can resolve
`<slug>.opts.ink`.

## Verification

```sh
curl -fsS https://opts.ink/healthz
curl -fsS https://opts.ink/readyz
curl -i https://unused-test-slug.opts.ink/
```

The unused wildcard hostname should reach OpenTunnel and return `503 Tunnel
Offline`, proving wildcard DNS, TLS, and host routing. Then open the dashboard,
connect the CLI to a test domain, and send a request through that hostname.

## Rollback

Roll back the API and frontend deployments in Railway. Database migrations in this
MVP are forward-only; restore a tested PostgreSQL backup before rolling back
across an incompatible schema change.
