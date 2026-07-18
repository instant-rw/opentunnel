# Local development

Requirements: Go 1.24+, Node.js 22+, npm, and Docker.

```sh
cp .env.example .env
npm ci
docker compose up --build
```

The dashboard and API are available at `http://localhost:8080`; PostgreSQL is
published at port 5432. For faster UI iteration, run only PostgreSQL and start
the processes separately:

```sh
docker compose up -d postgres
set -a; . ./.env; set +a
go run ./cmd/server
npm run dev --workspace web
```

The Next development server uses its own origin, while the production image
embeds the static export in the Go binary. Use the Compose image when validating
same-origin cookies and the production asset path.

## Generation and checks

```sh
npm run generate
git diff --exit-code
npm run format:check
npm run lint
npm run check
npm run build
go test -race ./...
go build ./cmd/server ./cmd/opentunnel
OPENTUNNEL_TEST_DATABASE_URL="$DATABASE_URL" go test ./internal/storage
```

`npm run generate` updates the OpenAPI Go/TypeScript clients and protobuf
bindings. Generated output is committed; CI fails on generation drift.
