# OpenTunnel dashboard

This directory contains the OpenTunnel dashboard, a TanStack Start single-page
application built with React, TypeScript, Vite, Tailwind CSS, and shadcn/ui. It
uses the control-plane API for account, domain, tunnel, request inspection, and
replay workflows.

See the [project README](../README.md) for the full-stack setup.

## Local development

Install dependencies and configure the API URL:

```sh
bun install
cp .env.example .env
bun run dev
```

The dashboard runs at `http://localhost:3000` by default.
`VITE_API_URL` must point to the API's `/api/v1` endpoint. The API must allow
the dashboard origin through `OPENTUNNEL_CORS_ORIGINS`.

## Commands

```sh
bun run dev        # start the development server
bun run typecheck  # check TypeScript
bun run test       # run Vitest
bun run lint       # run ESLint
bun run check      # verify formatting
bun run format     # apply formatting
bun run build      # create the production build
```

The root `make check` command runs the required frontend checks together with
the Go test and build steps.

## API types

`src/lib/api.generated.ts` is generated from
`../shared/api/openapi.yaml`. Do not edit it manually.

```sh
bun run generate:api
```

The root `make generate` command regenerates all OpenAPI and protobuf outputs.
Generated files are committed and checked for drift in CI.

## Production

The production image is built by `frontend/Dockerfile` and served by nginx.
Set `VITE_API_URL` at build time; use `.env.production.example` as a reference.

## Contributing and license

Read [CONTRIBUTING.md](../CONTRIBUTING.md) before submitting changes. This
project is distributed under the
[PolyForm Noncommercial License 1.0.0](../LICENSE); commercial use is
prohibited.
