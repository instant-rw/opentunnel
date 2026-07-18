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
browser and CLI target the same control-plane origin and that cookies are allowed.
System clocks must be reasonably synchronized.

## Tunnel repeatedly reconnects

Check proxies for WebSocket idle timeouts, compare heartbeat and grace settings,
and verify the local target is listening on `127.0.0.1:<port>`. A second CLI
using the same domain is rejected until the first session disconnects.

## Dashboard is missing or stale

Build the root Dockerfile; a plain local Go build embeds only the development
fallback asset unless `internal/webassets/out` was populated before compilation.
Purge Cloudflare cache for HTML, but keep fingerprinted `_next/static` assets
cacheable. API and tunnel paths must never be cached.

## Installer checksum or PATH failures

Installers require a published `v*` GitHub release and `checksums.txt`. Set
`OPENTUNNEL_VERSION=vX.Y.Z` to pin a release and `OPENTUNNEL_INSTALL_DIR` to a
writable directory. Open a new PowerShell session after the Windows installer
updates the user PATH.
