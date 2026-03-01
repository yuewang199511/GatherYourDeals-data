# Folder Structure for This Service

```
GatherYourDeals-data/
├── cmd/
│   └── gatheryourdeals/
│       └── main.go                      # Single binary entry point (cobra: serve, init, admin)
├── internal/
│   ├── auth/
│   │   ├── service.go                   # Register, login, password reset business logic
│   │   ├── jwt.go                       # TokenService: JWT issuance, validation, refresh token lifecycle
│   │   └── password.go                  # bcrypt hashing and verification
│   ├── handler/
│   │   ├── auth.go                      # HTTP handlers: register, login, refresh, logout, me
│   │   ├── admin.go                     # HTTP handlers: list users, delete user (admin only)
│   │   └── router.go                    # Route registration
│   ├── middleware/
│   │   └── auth.go                      # Bearer token validation, role enforcement
│   ├── model/
│   │   └── user.go                      # User struct, Role type, role constants
│   └── repository/
│       ├── repository.go                # Interface definitions (UserRepository)
│       └── sqlite/
│           ├── sqlite.go                # SQLite connection, goose migration runner
│           ├── user.go                  # SQLite implementation of UserRepository
│           ├── refresh_token.go         # SQLite implementation of auth.RefreshTokenStore
│           └── migrations/              # SQL migration files (embedded via go:embed)
│               ├── 00001_create_users_table.sql
│               └── 00003_create_refresh_tokens_table.sql
├── docs/
│   ├── api.yaml
│   ├── api_examples.md
│   ├── connection_and_auth.md
│   ├── data_format.md
│   └── service_structure.md
├── config.yaml
├── .env.example
├── go.mod
└── README.md
```

# CLI Commands

Single binary with subcommands:

```
gatheryourdeals init                               # Create database and admin account (interactive)
gatheryourdeals serve                              # Start the HTTP server
gatheryourdeals admin reset-password               # Reset a user's password (interactive)
gatheryourdeals --config /path/to/config.yaml serve   # Use a custom config file
```

Build:
```
go build -o gatheryourdeals ./cmd/gatheryourdeals
```

# Design Decisions

## Single Binary

The server and admin CLI are subcommands of one binary, following the pattern used by Gitea, Docker, and Kubernetes. The `serve` command starts the HTTP server. The `init` and `admin` commands operate directly on the database for setup and recovery. This simplifies deployment — one file does everything.

## Direct JWT Authentication (not OAuth2)

The service is its own authentication provider — it owns the user database and issues its own tokens. OAuth2 was originally used via the Resource Owner Password Credentials (ROPC) grant, but that grant is deprecated in OAuth 2.1 precisely because it is the wrong tool when the server is both the identity provider and the resource server.

The replacement is direct JWT authentication:

- **Login** (`POST /api/v1/sessions`) verifies the password and returns a signed JWT access token plus a refresh token.
- **Access tokens** are stateless JWTs verified by HMAC-SHA256 signature. No database lookup is needed per request. The user's role is embedded in the token claims.
- **Refresh tokens** are stored in the `refresh_tokens` SQLite table for revocation support. They are rotated on every use — the old token is deleted and a new pair is issued.
- **Logout** deletes the refresh token from the database. The access token expires naturally.

This removes the need for Redis, OAuth2 client management, and the associated complexity.

## JWT Signing Secret

The secret is an HMAC-SHA256 signing key loaded from the `GYD_JWT_SECRET` environment variable at startup. It is never stored in `config.yaml` or source control. The server refuses to start if it is missing or shorter than 32 characters. See `docs/connection_and_auth.md` for a full explanation.

## Repository Pattern

`repository/repository.go` defines interfaces for data access. `repository/sqlite/` is one implementation. To swap to PostgreSQL, add a `repository/postgres/` package that implements the same interfaces. No business logic needs to change.

## Database: SQLite vs PostgreSQL

SQLite is the default — no separate database server, one file, trivial to back up, minimal resources (suitable for a Raspberry Pi or cheap VPS).

The tradeoff is that SQLite supports only a single writer at a time, so you can only run one app instance. If you need horizontal scaling, replace SQLite with PostgreSQL by implementing `repository/postgres/`. The repository interface makes this a clean swap with no changes to business logic or handlers.

## Migrations with Goose

Schema is managed by [goose](https://github.com/pressly/goose). Migration files live in `repository/sqlite/migrations/` as plain SQL with `-- +goose Up` / `-- +goose Down` annotations. They are embedded into the binary at compile time via `go:embed`, so no extra files need to be deployed. To add a new table, create a new numbered SQL file.

## Dependency Wiring

Dependencies are created in the command functions and passed explicitly through constructors — no global singletons. The wiring order is: database → repository → service/token-service → handler → router.
