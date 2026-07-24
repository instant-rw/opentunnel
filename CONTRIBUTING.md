# Contributing to OpenTunnel

Thanks for helping improve OpenTunnel. Bug reports, documentation fixes, tests,
and focused code contributions are welcome.

By submitting a contribution, you license it under the
[PolyForm Noncommercial License 1.0.0](LICENSE) and agree that it may be
distributed under those terms. This project is source-available, not
OSI-approved open source, because commercial use is not permitted.

## Before you start

- Search the existing issues and pull requests before opening a duplicate.
- Open an issue before starting a large feature, protocol change, database
  migration, or breaking API change.
- Keep security vulnerabilities private by following [SECURITY.md](SECURITY.md).
- Keep changes focused. Unrelated refactors make review and rollback harder.

## Development setup

You need Go 1.25 or newer, Bun, Docker, and a reachable PostgreSQL database.

```sh
cp backend/.env.example backend/.env
cp frontend/.env.example frontend/.env
# Set DATABASE_URL in backend/.env, then:
docker compose --env-file frontend/.env up --build
```

See [docs/local-development.md](docs/local-development.md) for separate-process
development and database-backed test instructions.

## Making a change

1. Fork the repository and create a short-lived branch from the default branch.
2. Add or update tests for behavior changes.
3. Update the relevant documentation and contracts.
4. Run formatting and checks:

   ```sh
   make format
   make check
   ```

5. If an OpenAPI or protobuf contract changed, regenerate committed output:

   ```sh
   make generate
   git diff --exit-code
   ```

6. Open a pull request using the repository template.

Do not commit `.env` files, credentials, tunnel tokens, captured private
traffic, database dumps, build output, or dependency directories.

## Code guidelines

- Follow existing package and component conventions.
- Keep Go code formatted with `gofmt`.
- Keep frontend code formatted with Prettier and free of TypeScript errors.
- Put imports at the top of each module.
- Preserve backward compatibility unless the issue explicitly calls for a
  breaking change.
- Never hand-edit generated OpenAPI, protobuf, or frontend API bindings.
- Add migrations for schema changes; do not rewrite migrations that may already
  have been applied.

## Pull requests

A reviewable pull request explains the problem and the chosen solution, links
related issues, lists validation performed, and calls out migrations, contract
changes, security impact, or follow-up work. Screenshots are useful for visible
dashboard changes.

Maintainers may ask for changes, close inactive work, or decline changes that do
not fit the project's direction. Be respectful and follow the
[Code of Conduct](CODE_OF_CONDUCT.md).

## Contact

- Email: [iranzithierry@opts.ink](mailto:iranzithierry@opts.ink)
- GitHub: [@optunnel](https://github.com/optunnel)
