# Local development

Requirements: Go 1.25+, Bun, Docker, and an external PostgreSQL database.

```sh
cp backend/.env.example backend/.env
cp frontend/.env.example frontend/.env
# Set DATABASE_URL in backend/.env to your Postgres instance
docker compose --env-file frontend/.env up --build
```

API: `http://localhost:8080` · Dashboard: `http://localhost:3000`.

For faster iteration, run the API and SPA as separate processes (Postgres must
already be reachable via `DATABASE_URL`):

```sh
set -a; . ./backend/.env; set +a
make backend-dev          # or: go run ./backend/cmd/server
make frontend-dev         # or: cd frontend && bun run dev
```

Local SPA origin is `http://localhost:3000` and talks to the API with credentials
and CSRF. `OPENTUNNEL_CORS_ORIGINS` should include that origin.

## Generation and checks

```sh
make generate
git diff --exit-code
make check
OPENTUNNEL_TEST_DATABASE_URL="$DATABASE_URL" go test ./backend/internal/storage
```

`make generate` updates the OpenAPI Go/TypeScript clients and protobuf bindings.
Generated output is committed; CI fails on generation drift.
