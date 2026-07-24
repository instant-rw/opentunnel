# Deployment

Production images are built from `backend/Dockerfile` (API/tunnel server) and
`frontend/Dockerfile` (SPA on nginx).

`compose.yaml` runs both against an external `DATABASE_URL` (no bundled Postgres).
Copy env templates first:

- `backend/.env.example` → `backend/.env`
- `frontend/.env.example` → `frontend/.env`

Railway: each service has its own config-as-code file —

- [`backend/railway.json`](../backend/railway.json) — API image + `/readyz`
- [`frontend/railway.json`](../frontend/railway.json) — SPA image

Set `VITE_API_URL` at frontend build time (see
`frontend/.env.production.example`).

See [`docs/deployment-railway.md`](../docs/deployment-railway.md) for wildcard DNS
and TLS setup and [`docs/operations.md`](../docs/operations.md) for backups,
retention, limits, and incident guidance.
