# Deployment

The root `Dockerfile` is the production build, `railway.json` configures Railway,
and `compose.yaml` runs the same image with local PostgreSQL. See
[`docs/deployment-railway.md`](../docs/deployment-railway.md) for wildcard DNS
and TLS setup and [`docs/operations.md`](../docs/operations.md) for backups,
retention, limits, and incident guidance.
