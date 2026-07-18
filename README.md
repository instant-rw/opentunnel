# OpenTunnel

OpenTunnel is a production-minded HTTP/HTTPS reverse-tunnel MVP. A Go process
serves the control plane and embedded dashboard, routes wildcard hosts, and
multiplexes traffic over authenticated WebSockets to the Go CLI. PostgreSQL
stores accounts, domains, sessions, captured requests, and replay history.

## Layout

- `cmd/server`: API, dashboard, public-host router, and tunnel server.
- `cmd/opentunnel`: Go CLI entry point.
- `internal/gen/api`: generated Go control-plane bindings.
- `internal/gen/tunnel/v1`: generated versioned tunnel protocol bindings.
- `api/openapi.yaml`: REST control-plane source of truth.
- `protocol/tunnel.proto`: binary WebSocket frame contract.
- `web`: statically exported Next.js dashboard.
- `migrations`: embedded PostgreSQL migrations.
- `docs`: development, deployment, operations, and security guides.

## Prerequisites

- Go 1.24 or newer
- Node.js 22 or newer and npm
- Docker, if running PostgreSQL locally

## Setup

```sh
cp .env.example .env
npm install
docker compose up -d postgres
```

Run the complete local stack:

```sh
docker compose up --build
```

Open `http://localhost:8080`. Liveness is `/healthz`; readiness is `/readyz`.
For separate-process development, see [local development](docs/local-development.md).

## Contracts and generated code

Regenerate TypeScript, Go, and protobuf bindings:

```sh
npm run generate
```

Generated files are committed, and CI detects drift by regenerating before tests.

## Checks

```sh
npm run format:check
npm run lint
npm run check
npm run build
go test ./cmd/... ./internal/...
go build ./cmd/server ./cmd/opentunnel
```

## Documentation

- [CLI installation and usage](docs/cli.md)
- [Railway and Cloudflare deployment](docs/deployment-railway.md)
- [Operations, backups, limits, and retention](docs/operations.md)
- [Security model](docs/security.md)
- [Troubleshooting](docs/troubleshooting.md)
