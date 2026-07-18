# Operations

## Health and logs

`/healthz` is process liveness. `/readyz` performs a PostgreSQL ping and returns
503 until dependencies are usable. `/api/v1/healthz` remains the public API
health endpoint. The server emits JSON logs to stdout; configure the minimum
level with `OPENTUNNEL_LOG_LEVEL=debug|info|warn|error`.

Request logs include method, path, host, status, and duration. They intentionally
exclude authorization, cookies, request bodies, and query strings. Railway
should retain and alert on error logs, readiness failures, elevated 5xx rates,
and restarts.

## Limits and retention

- `OPENTUNNEL_CAPTURE_BODY_BYTES` caps each captured request and response body.
- `OPENTUNNEL_MAX_STORED_REQUESTS` retains the newest requests per domain;
  pruning occurs when a new request is captured.
- `OPENTUNNEL_MAX_IN_FLIGHT`, `OPENTUNNEL_QUEUE_DEPTH`, and
  `OPENTUNNEL_MAX_CHUNK_BYTES` bound tunnel memory and backpressure.
- `OPENTUNNEL_REQUEST_TIMEOUT`, `OPENTUNNEL_HEARTBEAT`, and
  `OPENTUNNEL_HEARTBEAT_GRACE` control request and disconnect detection.

Lower capture and request-retention limits before increasing concurrency. A
captured body can contain application data even though sensitive headers are
redacted.

## PostgreSQL backups

Enable Railway's scheduled PostgreSQL backups and choose retention that matches
the application's data policy. At least weekly, restore the newest backup into a
separate database and run the server integration tests against it.

Manual logical backup and restore:

```sh
pg_dump --format=custom --no-owner "$DATABASE_URL" > opentunnel.dump
createdb opentunnel_restore
pg_restore --no-owner --dbname="$RESTORE_DATABASE_URL" opentunnel.dump
```

Encrypt exports, restrict access, and delete expired copies. A restore contains
users, session/token digests, captured traffic, and replay history. Revoke CLI
tokens and web sessions if a backup leaves controlled storage.

## Capacity and incidents

Scale vertically first: active sessions live in process memory, so multiple
server replicas cannot safely share one domain without a distributed session
registry. During an incident, check `/readyz`, PostgreSQL connections, Railway
memory, WebSocket upgrade status, and Cloudflare origin errors. A disconnected
CLI must produce `503 Tunnel Offline`; it must never fall through to the
dashboard or another tenant.
