# Troubleshooting

## Server does not become ready

Check `DATABASE_URL`, PostgreSQL TLS settings, connection limits, and migration
errors in server logs. `/healthz` may be 200 while `/readyz` is 503 when the
process is alive but PostgreSQL is unavailable.

## Wildcard hostname does not connect

Confirm both the wildcard application CNAME and Railway ACME CNAME exist. The
ACME record must be DNS only. Verify Railway has issued the wildcard certificate,
Cloudflare uses Full (strict), WebSockets are enabled, and the origin receives
the original `Host`. `curl -i https://unused.opts.ink/` should return `503 Tunnel
Offline`, not a Cloudflare or Railway 404.

## CLI login remains pending

Use the newest device URL and code; codes expire after ten minutes. Ensure the
browser opens `OPENTUNNEL_FRONTEND_URL` (dashboard) while the CLI talks to the
API origin, and that cookies are allowed for the configured cookie domain.
System clocks must be reasonably synchronized.

## Dashboard cannot call the API (CORS / cookies)

Confirm `OPENTUNNEL_CORS_ORIGINS` includes the exact dashboard origin, cookies
use `OPENTUNNEL_COOKIE_DOMAIN` that covers both hosts when split (e.g.
`.opts.ink`), and the SPA was built with the correct `VITE_API_URL`. Locally,
`http://localhost:3000` → `http://localhost:8080` should work with
`OPENTUNNEL_SECURE_COOKIES=false`.

## Tunnel repeatedly reconnects

Check proxies for WebSocket idle timeouts, compare heartbeat and grace settings,
and verify the local target is listening on `127.0.0.1:<port>`. A second CLI
using the same domain is rejected until the first session disconnects.

## Dashboard is missing or stale

Rebuild and redeploy the frontend image (`frontend/Dockerfile`). Purge Cloudflare
cache for HTML; keep fingerprinted static assets cacheable. API and tunnel paths
must never be cached.

## Installer checksum or PATH failures

Installers require a published `v*` GitHub release and `checksums.txt`. Set
`OPENTUNNEL_VERSION=vX.Y.Z` to pin a release and `OPENTUNNEL_INSTALL_DIR` to a
writable directory. Open a new PowerShell session after the Windows installer
updates the user PATH. Use `opentunnel update` to refresh an existing install.
