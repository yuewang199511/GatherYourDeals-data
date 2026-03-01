# GatherYourDeals-data

The data service for the GatherYourDeals project. It provides a server for storing and querying purchase data, with user authentication via JWT.

## Quick Start (without Docker)

**Prerequisites:** Go 1.25+, GCC (required by the SQLite driver)

```bash
go mod tidy
go build -o gatheryourdeals ./cmd/gatheryourdeals

export GYD_JWT_SECRET="$(openssl rand -hex 32)"
./gatheryourdeals init      # create database and admin account
./gatheryourdeals serve     # start the server on :8080
```

## Quick Start (with Docker)

**Prerequisites:** Docker and Docker Compose

```bash
cp .env.example .env

# Generate a random secret and paste it into .env
openssl rand -hex 32
# Edit .env and set GYD_JWT_SECRET to the generated value

docker compose run --rm app init    # create database and admin account
docker compose up --build           # start the server on :8080
```

> **Note:** Docker Compose treats dollar signs in `.env` values as
> variable interpolation. If your secret contains dollar-sign characters
> (e.g. from `openssl rand -base64`), you will see warnings like
> *"The "mP" variable is not set"* and the secret will be silently
> corrupted. Use `openssl rand -hex 32` instead — hex output only
> contains `0-9` and `a-f`, so it avoids this issue entirely.
> If you must use a secret that contains a dollar sign, escape each one
> by doubling it (e.g. `$$`).

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
- **Docker support** — multi-stage build, persistent volume for database
- **JWT authentication** — stateless access tokens, rotating refresh tokens
- **Role-based access** — admin and user roles enforced on every request
- **Flexible schema** — native fields as columns, user-defined fields as JSON
- **SQLite with WAL mode** — lightweight, no setup required
- **Swappable database** — repository pattern allows switching to PostgreSQL
- **Embedded migrations** — schema managed by goose, compiled into the binary
