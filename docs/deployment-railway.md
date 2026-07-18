# Railway and Cloudflare deployment

## Railway

1. Create a Railway project, add a PostgreSQL service, and deploy this repository
   as a service. `railway.json` selects the root `Dockerfile` and `/readyz`.
2. Set `DATABASE_URL` from the PostgreSQL service and copy the non-secret values
   from `.env.production.example`. Set `OPENTUNNEL_BASE_URL=https://opts.ink`,
   `OPENTUNNEL_PUBLIC_HOST=opts.ink`, and `OPENTUNNEL_SECURE_COOKIES=true`.
3. Generate a Railway domain first and verify `/healthz` and `/readyz`. The
   process runs migrations during startup, so deploy only one migration-capable
   version at a time.
4. In the service networking settings, add both `opts.ink` and `*.opts.ink` as
   custom domains. Copy Railway's current CNAME and ACME validation targets;
   do not infer or reuse values from another environment.

Railway must route WebSocket upgrades without a path rewrite. The service listens
on Railway's injected `PORT` through `OPENTUNNEL_HTTP_ADDR`; set it to `:${PORT}`
if Railway does not map port 8080 automatically.

## Cloudflare DNS

Create the records Railway requests:

- `@` CNAME to the Railway application target.
- `*` CNAME to the Railway application target.
- Railway's `_acme-challenge` CNAME for the wildcard certificate.

Keep the ACME challenge record **DNS only**. Start the application CNAME records
as DNS only until Railway shows valid certificates. They may then be proxied
through Cloudflare. Set Cloudflare SSL/TLS mode to **Full (strict)** after
Railway's certificates are active (use **Full** only during initial diagnosis).
Never use Flexible mode: it breaks secure-cookie and origin-security assumptions.

Cloudflare must allow WebSockets. Do not cache `/api/*`, `/tunnel`, `/healthz`,
or `/readyz`; dashboard fingerprinted assets may be cached. Forward the original
`Host` header so the server can resolve `<slug>.opts.ink`.

## Verification

```sh
curl -fsS https://opts.ink/healthz
curl -fsS https://opts.ink/readyz
curl -i https://unused-test-slug.opts.ink/
```

The unused wildcard hostname should reach OpenTunnel and return `503 Tunnel
Offline`, proving wildcard DNS, TLS, and host routing. Then connect the CLI to a
test domain and send a request through that hostname.

## Rollback

Roll back the application image in Railway. Database migrations in this MVP are
forward-only; restore a tested PostgreSQL backup before rolling back across an
incompatible schema change.
