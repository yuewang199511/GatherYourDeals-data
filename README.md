# GatherYourDeals-data

This is the data service for the GatherYourDeals project. It provides a server for storing and querying purchase data, with user authentication via OAuth2.

## Quick Start

### Prerequisites

- Go 1.23 or later
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

Creates the database and prompts for an admin username and password.

### Run

```bash
./gatheryourdeals serve
```

The server starts on the port configured in `config.yaml` (default: 8080). It will refuse to start if no admin account exists.

## Configuration

The server reads from `config.yaml` in the current directory by default. Use `--config` to specify a different path.

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

On first startup, clients from `config.yaml` are seeded into the database. After that, clients are managed via the admin API and persist across restarts.

## CLI Commands

```
gatheryourdeals init                    # Create database and admin account
gatheryourdeals serve                   # Start the HTTP server
gatheryourdeals admin reset-password    # Reset a user's password
```

### Global flags

```
--config <path>    Path to config file (default: config.yaml)
--db <path>        Override database path from config
```

## Admin Recovery

If the admin forgets their password, reset it directly on the host machine:

```bash
./gatheryourdeals admin reset-password
```

This connects directly to the database — the server does not need to be running.

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
- **OAuth2 authentication** — standard token flow with access and refresh tokens
- **Dynamic client management** — add/revoke OAuth2 clients via admin API without restart
- **SQLite with WAL mode** — lightweight, concurrent reads, no setup required
- **Swappable database** — repository pattern allows switching to PostgreSQL
- **Embedded migrations** — schema managed by goose, compiled into the binary
- **YAML configuration** — server port, database path, OAuth2 clients all configurable
