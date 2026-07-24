# Migrations

The server applies ordered `*.up.sql` migrations transactionally at startup and
records completed versions in `schema_migrations`.

Destructive down migrations are provided for development recovery only; the
server never applies them automatically.

Run the PostgreSQL integration test against a disposable database:

```sh
OPENTUNNEL_TEST_DATABASE_URL='postgres://...' go test ./internal/storage
```
