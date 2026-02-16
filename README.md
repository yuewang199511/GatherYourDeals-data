# GatherYourDeals-data

This is the data service for the GatherYourDeals project. It provides a server for storing and querying purchase data, with user authentication via OAuth2.

## Quick Start (without Docker)

### Prerequisites

- Go 1.25.5
- GCC (required by the SQLite driver)

### Build

```bash
go mod tidy
go build -o gatheryourdeals ./cmd/gatheryourdeals
```

### Initialize

```bash
./gatheryourdeals init
```

Creates the database and prompts for an admin username and password. The database file is created at the path specified in `config.yaml` (default: `gatheryourdeals.db` in the current directory).

### Run

```bash
./gatheryourdeals serve
```

The server starts on the port configured in `config.yaml` (default: 8080). It will refuse to start if no admin account exists.

### Admin Recovery

If the admin forgets their password, reset it directly on the host machine:

```bash
./gatheryourdeals admin reset-password
```

This connects directly to the database — the server does not need to be running.

## Quick Start (with Docker)

### Prerequisites

- Docker and Docker Compose

### First-time setup

```bash
# Build the image and initialize the database.
# This prompts for an admin username and password.
docker compose run --rm app init
```

This creates the admin account and database inside a Docker volume (`gyd-data`). You only need to do this once.

### Run

```bash
docker compose up --build
```

The server starts on port 8080. The `--build` flag ensures the image is rebuilt if the code has changed.

### Stop

```bash
docker compose down
```

The database persists in the `gyd-data` volume. Next time you run `docker compose up --build`, everything is still there.

### Admin Recovery (Docker)

```bash
docker compose run --rm app admin reset-password
```

### Custom Configuration

The `config.yaml` at the repo root is baked into the Docker image. To override it without rebuilding, mount your own config:

```yaml
# docker-compose.yml
services:
  app:
    volumes:
      - gyd-data:/data
      - ./my-config.yaml:/data/config.yaml
```

## Configuration

The server reads `config.yaml` from the current directory by default. Override with `--config` flag or `GYD_CONFIG` environment variable.

⚠️ Service will only read clients from database after first initialization

```yaml
server:
  port: "8080"

database:
  path: "gatheryourdeals.db"

oauth2:
  access_token_exp: "1h"
  refresh_token_exp: "168h"

  clients:
    - id: "gatheryourdeals"
      secret: ""
      domain: "http://localhost"
```

Server, database, and token expiry settings are read from `config.yaml` on every startup. Changes take effect on restart.

⚠️ **OAuth2 clients are only read from `config.yaml` on first startup** to seed the database. After that, clients are managed exclusively via the admin API. Editing the `clients` section in `config.yaml` after initialization has no effect.

## CLI Commands

```
gatheryourdeals init                    # Create database and admin account
gatheryourdeals serve                   # Start the HTTP server
gatheryourdeals admin reset-password    # Reset a user's password
```

### Global flags

```
--config <path>    Path to config file (default: config.yaml, or GYD_CONFIG env var)
--db <path>        Override database path from config
```

## API Reference

The full API specification is available in OpenAPI format:

- **[OpenAPI Spec](docs/api.yaml)** — machine-readable, can be loaded into Swagger UI, Postman, or used for client code generation
- **[API Examples](docs/api_examples.md)** — curl examples for every endpoint

### Endpoints Overview

| Endpoint | Method | Auth | Description |
|:---------|:-------|:-----|:------------|
| `/api/v1/users` | POST | client_id | Create a new user |
| `/api/v1/oauth/token` | POST | — | Login or refresh token |
| `/api/v1/oauth/sessions` | DELETE | Bearer | Delete current session |
| `/api/v1/admin/clients` | POST | Admin | Register an OAuth2 client |
| `/api/v1/admin/clients` | GET | Admin | List all clients |
| `/api/v1/admin/clients/:id` | DELETE | Admin | Revoke a client |

## Documentation

- [OpenAPI Spec](docs/api.yaml) — full API specification
- [API Examples](docs/api_examples.md) — curl examples for every endpoint
- [Connection and Authentication](docs/connection_and_auth.md) — hosting, auth design, access keys
- [Data Format](docs/data_format.md) — purchase record format, metadata, ETL process
- [Service Structure](docs/service_structure.md) — project layout, design decisions

## Key Features

- **Single binary** — server and admin CLI in one executable
- **Docker support** — multi-stage build, persistent volume for database
- **OAuth2 authentication** — standard token flow with access and refresh tokens
- **Dynamic client management** — add/revoke OAuth2 clients via admin API without restart
- **SQLite with WAL mode** — lightweight, concurrent reads, no setup required
- **Swappable database** — repository pattern allows switching to PostgreSQL
- **Embedded migrations** — schema managed by goose, compiled into the binary
- **YAML configuration** — server port, database path, OAuth2 clients all configurable
