# OpenTunnel

[![CI](https://github.com/optunnel/opentunnel/actions/workflows/ci.yml/badge.svg)](https://github.com/optunnel/opentunnel/actions/workflows/ci.yml)
[![License: PolyForm Noncommercial 1.0.0](https://img.shields.io/badge/license-PolyForm%20Noncommercial%201.0.0-blue.svg)](LICENSE)

OpenTunnel is a source-available, production-minded HTTP/HTTPS reverse tunnel.
A Go API server routes wildcard hosts and multiplexes traffic over authenticated
WebSockets to the Go CLI. A separate SPA dashboard talks to the API over CORS.
PostgreSQL stores accounts, domains, sessions, captured requests, and replay
history.

## Layout

- `backend/`: API, public-host router, and tunnel server (`cmd/server`).
- `cli/`: Go CLI (`cmd/opentunnel`), including `opentunnel update`.
- `shared/`: OpenAPI + protobuf contracts, generated Go bindings, tunnel client.
- `frontend/`: TanStack Start SPA dashboard (Bun + Vite).
- `backend/migrations/`: embedded PostgreSQL migrations.
- `docs/`: development, deployment, operations, and security guides.

## Prerequisites

- Go 1.25 or newer
- Bun (frontend)
- Docker, for Compose and images
- An external PostgreSQL database (`DATABASE_URL`)

## Setup

```sh
cp backend/.env.example backend/.env
cp frontend/.env.example frontend/.env
# Point DATABASE_URL in backend/.env at your Postgres instance, then:
docker compose --env-file frontend/.env up --build
```

API: `http://localhost:8080` · Dashboard: `http://localhost:3000`  
Liveness is `/healthz`; readiness is `/readyz`.

For separate-process development, see [local development](docs/local-development.md).

## Contracts and generated code

```sh
make generate
```

Regenerates Go OpenAPI/protobuf bindings and the frontend OpenAPI types.
Generated files are committed; CI fails on drift.

## Checks

```sh
make check
```

Or individually: `make backend-dev`, `make frontend-dev`, `make cli-build`.

## Contributing and support

Contributions are welcome. Read [CONTRIBUTING.md](CONTRIBUTING.md) before
opening an issue or pull request, follow the
[Code of Conduct](CODE_OF_CONDUCT.md), and use [SUPPORT.md](SUPPORT.md) when
asking for help.

Report suspected vulnerabilities privately according to
[SECURITY.md](SECURITY.md).

## Documentation

- [CLI installation and usage](docs/cli.md)
- [Railway and Cloudflare deployment](docs/deployment-railway.md)
- [Operations, backups, limits, and retention](docs/operations.md)
- [Security model](docs/security.md)
- [Troubleshooting](docs/troubleshooting.md)

## License

Copyright 2026 OpenTunnel.

OpenTunnel is distributed under the
[PolyForm Noncommercial License 1.0.0](LICENSE). You may use, modify, and
redistribute it only for noncommercial purposes permitted by that license.
Commercial use is prohibited.

Because of this restriction, OpenTunnel is source-available rather than
OSI-approved open source.

## Contact

- Email: [iranzithierry@opts.ink](mailto:iranzithierry@opts.ink)
- GitHub: [@optunnel](https://github.com/optunnel)
