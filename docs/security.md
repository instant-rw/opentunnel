# Security model

OpenTunnel terminates public TLS at Cloudflare/Railway and carries tunnel frames
over authenticated WSS. Each persistent domain belongs to one user and accepts
at most one active CLI session. Ownership is checked for domains, request logs,
replays, tokens, and tunnel registration.

Web passwords use Argon2id. Web sessions and CLI tokens are stored as digests.
Session cookies are HttpOnly, SameSite=Lax, and Secure in production; mutating
cookie-authenticated requests require a CSRF token. Authentication endpoints are
rate-limited. CLI credentials use the operating-system credential store with a
user-only file fallback.

The proxy removes hop-by-hop headers. Captured authorization, cookie, proxy
authorization, and set-cookie values are redacted before persistence. Bodies are
captured only up to the configured byte limit, but retained traffic can still
contain secrets or personal data.

## Production requirements

- Use Cloudflare Full (strict), secure cookies, and a dedicated least-privilege
  PostgreSQL credential over encrypted transport.
- Restrict Railway project access and rotate database and account credentials.
- Keep dependencies and base images patched; review Bun/npm audits, Go
  vulnerability scans, and container scans during releases.
- Do not expose PostgreSQL publicly. Backups must be encrypted and access-logged.
- Treat tunnel access as equivalent to access to the developer's local service.
  Bind the CLI target to loopback and never forward privileged local endpoints.

## MVP boundaries

Email verification and multi-factor authentication are not enforced. Rate limits
are process-local. Active tunnel coordination is process-local, so horizontal
replicas are unsupported. The service is an HTTP/HTTPS tunnel, not a network
isolation boundary and not a raw TCP/UDP proxy.

Report suspected vulnerabilities privately by following the
[security policy](../SECURITY.md). Revoke affected CLI tokens and web sessions
before sharing logs or database extracts.
