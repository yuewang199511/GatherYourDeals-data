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
│   │   ├── meta.go                      # HTTP handlers: list fields, create field, update description
│   │   ├── receipt.go                   # HTTP handlers: create, list, get, delete receipts
│   │   └── router.go                    # Route registration
│   ├── middleware/
│   │   └── auth.go                      # Bearer token validation, role enforcement
│   ├── model/
│   │   ├── user.go                      # User struct, Role type, role constants
│   │   ├── meta.go                      # MetaField struct
│   │   └── receipt.go                   # Receipt struct, sentinel errors
│   └── repository/
│       ├── repository.go                # Interface definitions (UserRepository, MetaFieldRepository, ReceiptRepository)
│       ├── sqlite/
│       │   ├── sqlite.go                # SQLite connection, goose migration runner
│       │   ├── user.go                  # SQLite implementation of UserRepository
│       │   ├── refresh_token.go         # SQLite implementation of auth.RefreshTokenStore
│       │   ├── meta_field.go            # SQLite implementation of MetaFieldRepository
│       │   ├── receipt.go               # SQLite implementation of ReceiptRepository
│       │   ├── testutil/
│       │   │   └── testutil.go          # In-memory test database helper
│       │   └── migrations/              # SQL migration files (embedded via go:embed)
│       │       ├── 00001_create_users_table.sql
│       │       ├── 00003_create_refresh_tokens_table.sql
│       │       ├── 00004_create_meta_fields_table.sql
│       │       └── 00005_create_receipts_table.sql
│       ├── postgres/
│       │   ├── postgres.go              # PostgreSQL connection, goose migration runner
│       │   ├── user.go                  # PostgreSQL implementation of UserRepository
│       │   ├── refresh_token.go         # PostgreSQL implementation of auth.RefreshTokenStore
│       │   ├── meta_field.go            # PostgreSQL implementation of MetaFieldRepository
│       │   ├── receipt.go               # PostgreSQL implementation of ReceiptRepository
│       │   └── migrations/              # PostgreSQL-compatible SQL files (embedded via go:embed)
│       │       ├── 00001_create_users_table.sql
│       │       ├── 00003_create_refresh_tokens_table.sql
│       │       ├── 00004_create_meta_fields_table.sql
│       │       └── 00005_create_receipts_table.sql
│       └── redis/
│           └── refresh_token.go         # Redis implementation of auth.RefreshTokenStore
├── docs/
│   ├── api.yaml                         # OpenAPI 3.0 specification
│   ├── api_examples.md                  # curl examples for every endpoint
│   ├── connection_and_auth.md           # Hosting, auth design, access keys
│   ├── data_format.md                   # Purchase record format, metadata, ETL process
│   └── service_structure.md             # This file — project layout, design decisions
├── .github/workflows/                   # CI/CD: build, test, code quality, security
├── config.yaml
├── docker-compose.yml
├── docker-compose.postgres.yml          # Optional overlay for local PostgreSQL dev
├── Dockerfile
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

# API Route Summary

## Public
| Method | Path | Description |
|:-------|:-----|:------------|
| POST | `/api/v1/users` | Register a new user |
| POST | `/api/v1/auth/login` | Login |
| POST | `/api/v1/auth/refresh` | Refresh access token |

## Authenticated
| Method | Path | Description |
|:-------|:-----|:------------|
| POST | `/api/v1/auth/logout` | Logout (revoke refresh token) |
| GET | `/api/v1/auth/me` | Current user info |
| GET | `/api/v1/meta` | List all registered fields |
| POST | `/api/v1/meta` | Register a new field |
| PUT | `/api/v1/meta/:fieldName` | Update a field description (admin only) |
| GET | `/api/v1/users` | List all users (admin only) |
| DELETE | `/api/v1/users/:id` | Delete a user (admin only) |
| POST | `/api/v1/receipts` | Create a receipt |
| GET | `/api/v1/receipts` | List own receipts |
| GET | `/api/v1/receipts/:id` | Get a receipt by ID |
| DELETE | `/api/v1/receipts/:id` | Delete a receipt |

Endpoints marked **(admin only)** check the user's role inside the handler and return 403 if the user is not an admin.

# Design Decisions

## Single Binary

The server and admin CLI are subcommands of one binary, following the pattern used by Gitea, Docker, and Kubernetes. The `serve` command starts the HTTP server. The `init` and `admin` commands operate directly on the database for setup and recovery. This simplifies deployment — one file does everything.

## Direct JWT Authentication (not OAuth2)

The service is its own authentication provider — it owns the user database and issues its own tokens. OAuth2 was originally used via the Resource Owner Password Credentials (ROPC) grant, but that grant is deprecated in OAuth 2.1 precisely because it is the wrong tool when the server is both the identity provider and the resource server.

The replacement is direct JWT authentication:

- **Login** (`POST /api/v1/auth/login`) verifies the password and returns a signed JWT access token plus a refresh token.
- **Access tokens** are stateless JWTs verified by HMAC-SHA256 signature. No database lookup is needed per request. The user's role is embedded in the token claims.
- **Refresh tokens** are stored for revocation support and rotated on every use — the old token is deleted and a new pair is issued.
- **Logout** deletes the refresh token. The access token expires naturally.

**Refresh token store selection** (checked at startup in `serveCmd`):

| `REDIS_URL` set? | Store used | Notes |
|:---|:---|:---|
| Yes | `repository/redis/RefreshTokenStore` | Native TTL; supports horizontal scaling |
| No | `repository/sqlite/` or `repository/postgres/` | Falls back to whichever DB driver is configured |

**Redis key layout** (when Redis is used):

| Key | Value | TTL |
|:---|:---|:---|
| `rt:{token}` | `userID` (string) | Token expiry duration (default 7 days) |
| `ut:{userID}` | SET of active token strings | Extended on each new token save |

- `rt:` (refresh token) is the primary lookup: given a token, find its owner in O(1).
- `ut:` (user tokens) is a reverse index used only by `DeleteAllForUser` (triggered when a user is deleted), so all sessions across all devices are revoked atomically.
- Expiry is enforced natively by Redis TTL — no manual cleanup queries needed.

## JWT Signing Secret

The secret is an HMAC-SHA256 signing key loaded from the `GYD_JWT_SECRET` environment variable at startup. It is never stored in `config.yaml` or source control. The server refuses to start if it is missing or shorter than 32 characters. See `docs/connection_and_auth.md` for a full explanation.

## Repository Pattern

`repository/repository.go` defines interfaces for data access. `repository/sqlite/` is one implementation. To swap to PostgreSQL, add a `repository/postgres/` package that implements the same interfaces. No business logic needs to change.

## Database: SQLite vs PostgreSQL

SQLite is the default — no separate database server, one file, trivial to back up, minimal resources (suitable for a Raspberry Pi or cheap VPS).

The tradeoff is that SQLite supports only a single writer at a time, so you can only run one app instance. If you need horizontal scaling, replace SQLite with PostgreSQL by implementing `repository/postgres/`. The repository interface makes this a clean swap with no changes to business logic or handlers.

## Flexible Schema with JSON Extras

Purchase records have fixed columns for the native fields (productName, price, storeName, etc.) and a JSON `extras` column for user-defined fields. This gives you the best of both worlds: efficient SQL queries on common fields, and flexibility for custom data.

Every key in `extras` must be registered in the `meta_fields` table before it can be used. This prevents typos and ensures every field has a description. The meta table is append-only — fields cannot be deleted, because existing receipts may reference them.

## Migrations with Goose

Schema is managed by [goose](https://github.com/pressly/goose). Migration files live in `repository/sqlite/migrations/` as plain SQL with `-- +goose Up` / `-- +goose Down` annotations. They are embedded into the binary at compile time via `go:embed`, so no extra files need to be deployed. To add a new table, create a new numbered SQL file.

## PostgreSQL Connection Pool Sizing

The PostgreSQL driver (`database/sql`) defaults to unlimited open connections. Without a cap, each replica opens as many connections as concurrent requests demand, which exhausts PostgreSQL's `max_connections` limit when running multiple replicas.

The app sets a fixed pool cap in `internal/repository/postgres/postgres.go`:
- `SetMaxOpenConns(10)` — maximum simultaneous connections per replica
- `SetMaxIdleConns(5)` — idle connections kept warm

**Why 10:** With Railway PostgreSQL's `max_connections = 100` and 10 reserved for CI tooling and migrations, this supports up to 9 replicas safely: `(100 - 10) / 10 = 9`. Chosen to be scale-safe without needing reconfiguration as replicas are added.

**The general formula for any deployment:**
```
max_open_per_replica = (pg_max_connections - reserved) / num_replicas
```

**PgBouncer (session mode):** A PgBouncer instance sits between the app and PostgreSQL. The app connects to PgBouncer; PgBouncer manages the actual PostgreSQL connections. In **session mode**, each app connection maps 1:1 to a PostgreSQL server connection for the entire session lifetime — so the app-side pool limits above still apply and must be set correctly. The pool math is unchanged; PgBouncer adds a stable connection endpoint and connection queuing, but does not reduce the total number of server connections. Transaction mode would decouple the math, but session mode is used to preserve compatibility with pgx prepared statements.

## Dependency Wiring

Dependencies are created in the command functions and passed explicitly through constructors — no global singletons. The wiring order is: database → repository → service/token-service → handler → router.
