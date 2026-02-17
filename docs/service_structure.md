# Folder Structure for This Service

```
GatherYourDeals-data/
├── cmd/
│   └── gatheryourdeals/
│       └── main.go                  # Single binary entry point (cobra: serve, init, admin)
├── internal/
│   ├── auth/
│   │   ├── service.go               # register, login, logout, refresh business logic
│   │   ├── password.go              # bcrypt hashing and verification
│   │   └── token.go                 # OAuth2 manager/server setup, Redis-backed token store
│   ├── handler/
│   │   └── auth.go                  # HTTP handlers for /api/v1/auth/* endpoints
│   ├── middleware/
│   │   └── auth.go                  # middleware to validate Bearer tokens on requests
│   ├── model/
│   │   └── user.go                  # User struct, Role type, role constants
│   └── repository/
│       ├── repository.go            # interface definition (UserRepository, etc.)
│       └── sqlite/
│           ├── sqlite.go            # SQLite connection setup, goose migration runner
│           ├── migrations/          # SQL migration files (embedded into binary via go:embed)
│           │   └── 00001_create_users_table.sql
│           └── user.go              # SQLite implementation of UserRepository
├── docs/
│   ├── connection_and_auth.md
│   ├── data_format.md
│   └── service_structure.md
├── go.mod
└── README.md
```

# CLI Commands

Single binary with subcommands:

```
gatheryourdeals init                              # Create database and admin account (interactive)
gatheryourdeals serve                             # Start the HTTP server
gatheryourdeals admin reset-password              # Reset a user's password (interactive)
gatheryourdeals --config /path/to/config.yaml serve  # Use a custom config file
```

Build command:
```
go build -o gatheryourdeals ./cmd/gatheryourdeals
```

# Design Decisions

## Single Binary

The server and admin CLI are subcommands of one binary, following the pattern used by Gitea, Docker, and Kubernetes. The `serve` command starts the HTTP server. The `init` and `admin` commands operate directly on the database for setup and recovery. This simplifies deployment — one file does everything.

## Repository Pattern

`repository/repository.go` defines interfaces for data access. `repository/sqlite/` is one implementation. To swap to PostgreSQL later, add a `repository/postgres/` package that implements the same interfaces. No business logic needs to change.

## Database: SQLite vs PostgreSQL

The default database is SQLite. This is a deliberate choice for self-hosting simplicity — there is no separate database server to run, the entire database is one file that is trivial to back up, and resource usage is minimal (suitable for a Raspberry Pi or cheap VPS).

The tradeoff is that SQLite only supports a single writer at a time. This means you can only run **one app instance**. If you run multiple instances behind a load balancer, concurrent writes will fail with "database is locked" errors.

For most users — small groups sharing deals — a single instance is more than enough. If you need horizontal scaling with multiple app instances, replace SQLite with PostgreSQL by implementing `repository/postgres/`. The repository interface makes this a clean swap with no changes to business logic or handlers.

## Migrations with Goose

Database schema is managed by [goose](https://github.com/pressly/goose). Migration files live in `repository/sqlite/migrations/` as plain SQL files with `-- +goose Up` and `-- +goose Down` annotations. They are embedded into the binary at compile time using `go:embed`, so no extra files need to be deployed.

Goose tracks which migrations have been applied in a `goose_db_version` table. To add a new table, create a new SQL file like `00002_create_purchases_table.sql`.

## Dependency Wiring

Dependencies are created in the command functions and passed explicitly through constructors — no global singletons. The wiring order is: database → repository → service → handler → router.

## Token Storage (Redis)

OAuth2 tokens (access and refresh) are stored in Redis rather than in-memory. This means tokens survive server restarts and can be shared across multiple service instances if you scale horizontally. Redis connection details are configured in `config.yaml` under the `redis` section. The `docker-compose.yml` includes a Redis service that the app depends on.
