# BetterGuests Server

Go API for Better Guests.

## Developer command

The repository has one developer command:

```sh
go run ./cmd/dev setup
go run ./cmd/dev migrate
go run ./cmd/dev migration-status
go run ./cmd/dev test
go run ./cmd/dev run
```

`setup` creates an ignored `.env` from `.env.example` only when `.env` does
not already exist. It never overwrites credentials. Go does not automatically
load `.env`; `cmd/dev` exports it before running the API or migrations. Tests
use only the disposable local database described below.

## One-command tests

With Docker Desktop running, execute:

```sh
go run ./cmd/dev test
```

This command creates a disposable PostgreSQL container, bootstraps the local
database roles, applies all embedded Goose migrations, and runs every Go test.
It then removes the container and its data, including after a failed test.

The test command never reads `.env`. It hard-overrides `DATABASE_URL`,
`MIGRATION_DATABASE_URL`, `TEST_DATABASE_URL`, and
`TEST_MIGRATION_DATABASE_URL` with loopback-only URLs for the container, so a
Railway or production URL in your shell or `.env` cannot be used by tests.

## Database roles

- `betterguests_app` is the API runtime role. It cannot create schemas or
  tables, own objects, change RLS, truncate tables, or assume the migration
  role.
- `betterguests_migrator` owns the `app` schema and applies reviewed schema
  migrations. The migration command refuses any other database role.

The API verifies `current_user` at startup and refuses to start unless it is
connected as `betterguests_app`.

## Bootstrap and migrations

`db/bootstrap/001_roles_and_schema.sql` is an administrator-only,
once-per-database bootstrap. It is grouped with the migration files for easy
discovery, but it is deliberately excluded from Goose because it creates the
migrator role that must already exist before Goose can connect.

Normal versioned migrations live in `db/migrations` and are compiled into the
migration executable. Follow `db/migrations/README.md` for the required RLS,
grant, constraint, and indexing pattern.

The integration tests intentionally create and drop a fixture table and execute
hostile SQL. Run them through `go run ./cmd/dev test`; its URLs are fixed to the
disposable local container and cannot inherit production credentials.

## Environment

- `DATABASE_URL`: runtime connection URL for `betterguests_app`
- `MIGRATION_DATABASE_URL`: migration URL for `betterguests_migrator`
- `DB_EXPECTED_ROLE=betterguests_app`
- `DB_MAX_CONNS=5` initially; total across replicas must remain below the role's
  20-connection limit
- `FIREBASE_PROJECT_ID`: Firebase project that issues the client ID tokens
- `FIREBASE_CREDENTIALS_JSON`: Firebase Admin service-account JSON, either raw
  or base64-encoded. Raw JSON may be wrapped in single quotes and span several
  lines; double quotes also work but must escape every JSON quote. The API
  verifies each bearer token with it and maps the Firebase UID to a stable
  `app.accounts` row on first use.
- `PORT`: HTTP port

## Tests

Go tests stay beside the packages they test using the standard `_test.go`
naming convention. This gives tests direct package access and keeps ownership
clear. The database suite uses two disposable accounts to test cross-account
access, identity spoofing, SQL injection, role escalation, DDL, RLS disabling,
truncation, migration-table access, and pooled session-state leakage.
