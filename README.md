# GatherYourDeals-data

The data service for the GatherYourDeals project. It provides a server for storing and querying purchase data, with user authentication via JWT.

![GatherYourDeals-data](./GatherYourDeals-data.png)

## Quick Start (without Docker) — SQLite

**Prerequisites:** Go 1.25+, GCC (required by the SQLite driver)

```bash
go mod tidy
go build -o gatheryourdeals ./cmd/gatheryourdeals

export GYD_JWT_SECRET="$(openssl rand -hex 32)"
./gatheryourdeals init      # create database and admin account
./gatheryourdeals serve     # start the server on :8080
```

## Quick Start (without Docker) — PostgreSQL

**Prerequisites:** Go 1.25+, GCC, a reachable PostgreSQL instance

```bash
go mod tidy
go build -o gatheryourdeals ./cmd/gatheryourdeals

export GYD_JWT_SECRET="$(openssl rand -hex 32)"
export GYD_DATABASE_DRIVER=postgres
export DATABASE_URL="postgres://user:password@host:5432/dbname?sslmode=require"

./gatheryourdeals init      # create schema and admin account in PostgreSQL
./gatheryourdeals serve     # start the server on :8080
```

Supported cloud platforms: **Railway**, **Azure Database for PostgreSQL**, **Supabase**, **Neon**.
On Railway, `DATABASE_URL` is injected automatically — just set `GYD_DATABASE_DRIVER=postgres`
in your service's environment variables.

Logs are written to both stdout and rotating files in `./logs/`.

## Quick Start (with Docker)

**Prerequisites:** Docker and Docker Compose

```bash
cp .env.example .env

# Generate a random secret and paste it into .env
openssl rand -hex 32
# Edit .env and set GYD_JWT_SECRET to the generated value

docker compose run --rm app init    # create database and admin account
docker compose up --build           # start the server on :8080

# When done, stop and remove containers (prevents auto-restart on next reboot):
docker compose down
```

Logs are written to stdout (visible via `docker compose logs`) and to rotating files persisted in `./data/logs/` on the host.

> **Note:** Docker Compose treats dollar signs in `.env` values as
> variable interpolation. If your secret contains dollar-sign characters
> (e.g. from `openssl rand -base64`), you will see warnings like
> *"The "mP" variable is not set"* and the secret will be silently
> corrupted. Use `openssl rand -hex 32` instead — hex output only
> contains `0-9` and `a-f`, so it avoids this issue entirely.
> If you must use a secret that contains a dollar sign, escape each one
> by doubling it (e.g. `$$`).

## Configuration

Configuration is loaded in this order — later values override earlier ones:

1. Built-in defaults
2. `config.yaml` (optional — if the file is absent, defaults apply)
3. Environment variables (always win)

This means the service runs without any config file on Railway, Azure, or any container platform — everything can be driven by env vars alone. `config.yaml` remains useful for local development.

### Environment variables

| Variable | Default | Description |
|:---------|:--------|:------------|
| `GYD_JWT_SECRET` | — | **Required.** JWT signing secret, minimum 32 characters. Generate with `openssl rand -hex 32`. |
| `PORT` | `8080` | HTTP listen port. Injected automatically by Railway and Azure Container Apps. |
| `GYD_DATABASE_DRIVER` | `sqlite` | Database backend. Set to `postgres` for cloud deployment. |
| `DATABASE_URL` | — | PostgreSQL connection string. Injected automatically by Railway when a PostgreSQL add-on is provisioned. |
| `GYD_ACCESS_TOKEN_EXP` | `1h` | Access token lifetime. Go duration syntax: `15m`, `1h`, `24h`. |
| `GYD_REFRESH_TOKEN_EXP` | `168h` | Refresh token lifetime. Go duration syntax: `24h`, `168h`, `720h`. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | — | OpenTelemetry export endpoint (e.g. `https://api.honeycomb.io`). Leave blank to disable tracing. |
| `OTEL_EXPORTER_OTLP_HEADERS` | — | OTel headers, e.g. `x-honeycomb-team=<key>`. |
| `OTEL_SERVICE_NAME` | `gatheryourdeals` | Service name shown in traces. |
| `OTEL_RESOURCE_ATTRIBUTES` | — | Comma-separated span attributes, e.g. `service.environment=production,db.type=postgres`. |

See `.env.example` for a full annotated template.

## Logging

All logs (Gin request logs and application logs) go to both stdout and a rotating log file. Log files are named with their creation timestamp, e.g. `gatheryourdeals-2025-04-05-14-30-00.log`. Only the two most recent files are kept.

Configure logging in `config.yaml`:

```yaml
log:
  dir: "logs"          # directory for log files (default: logs)
  max_size_mb: 10      # max file size before rotation (default: 10 MB)
```

| Setup | Log location on host |
|:------|:----|
| Local | `./logs/` |
| Docker | `./data/logs/` (mounted from container's `/data/logs/`) |

## GitHub Actions CI Setup

The integration test workflow requires five repository secrets. Set them at:
**Settings → Secrets and variables → Actions → New repository secret**

| Secret | Description |
|:-------|:------------|
| `GYD_JWT_SECRET` | Random hex string — run `openssl rand -hex 32` to generate |
| `GYD_ADMIN_USERNAME` | Admin account username (used by `./gatheryourdeals init` and admin tests) |
| `GYD_ADMIN_PASSWORD` | Admin account password |
| `GYD_TEST_USERNAME` | Regular test user username (auto-registered by the test suite) |
| `GYD_TEST_PASSWORD` | Regular test user password |

The workflow builds the binary, initialises a fresh SQLite database, starts the server, then runs the full pytest integration suite on every push to `main`/`develop` and on pull requests targeting `main`.

## Development

### Fixing lint/formatting issues

```bash
goimports -w ./...
```

This fixes both import ordering and general formatting. Run it before committing if `golangci-lint` reports `goimports` errors.

**Prerequisites:** `go install golang.org/x/tools/cmd/goimports@latest`

## Documentation

| Document | Description |
|:---------|:------------|
| [OpenAPI Spec](docs/api.yaml) | Full API specification (OpenAPI 3.0) |
| [API Examples](docs/api_examples.md) | curl examples for every endpoint |
| [Connection & Auth](docs/connection_and_auth.md) | Hosting, auth design, access keys |
| [Data Format](docs/data_format.md) | Purchase record format, metadata, ETL process |
| [Service Structure](docs/service_structure.md) | Project layout, CLI commands, design decisions |

## Key Features

- **Single binary** — server and admin CLI in one executable
- **Docker support** — multi-stage build, persistent volumes for database and logs
- **JWT authentication** — stateless access tokens, rotating refresh tokens
- **Role-based access** — admin and user roles enforced on every request
- **Flexible schema** — native fields as columns, user-defined fields as JSON
- **Structured logging** — stdout + rotating log files, Gin and app logs unified
- **SQLite with WAL mode** — lightweight, no setup required, default for local use
- **PostgreSQL support** — set `GYD_DATABASE_DRIVER=postgres` for cloud deployment
- **Embedded migrations** — schema managed by goose, compiled into the binary

# Future plan for deployment

⚠️compare price between railway and AWS
