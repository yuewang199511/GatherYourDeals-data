# GatherYourDeals-data

This is the data service for the GatherYourDeals project. It provides a server for storing and querying purchase data, with user authentication via JWT.

## Quick Start (without Docker)

### Prerequisites

- Go 1.25+
- GCC (required by the SQLite driver)

### Build

```bash
go mod tidy
go build -o gatheryourdeals ./cmd/gatheryourdeals
```

### Initialize

```bash
export GYD_JWT_SECRET="your-secret-at-least-32-chars-long"
./gatheryourdeals init
```

Creates the database and prompts for an admin username and password. The database file is created at the path specified in `config.yaml` (default: `gatheryourdeals.db` in the current directory).

### Run

```bash
./gatheryourdeals serve
```

The server starts on the port configured in `config.yaml` (default: 8080). It will refuse to start if no admin account exists or if `GYD_JWT_SECRET` is not set.

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
# Copy the example env file and set your JWT secret
cp .env.example .env
# Edit .env and set GYD_JWT_SECRET to a random 32+ character string
# You can generate one with: openssl rand -hex 32

# Initialize the database and create the admin account
docker compose run --rm app init
```

### Run

```bash
docker compose up --build
```

The server starts on port 8080. The `--build` flag ensures the image is rebuilt if the code has changed.

### Stop

```bash
docker compose down
```

The database persists in `./data/db/` on the host. Next time you run `docker compose up --build`, everything is still there.

### Admin Recovery (Docker)

```bash
docker compose run --rm app admin reset-password
```

## Configuration

The server reads `config.yaml` from the current directory by default. Override with the `--config` flag.

```yaml
server:
  port: "8080"

database:
  path: "gatheryourdeals.db"

auth:
  access_token_exp: "1h"
  refresh_token_exp: "168h"
```

The JWT signing secret is **not** stored in `config.yaml`. Set it via the environment variable:

```bash
export GYD_JWT_SECRET="your-secret-at-least-32-chars-long"
```

For Docker, set it in your `.env` file (see `.env.example`). The server will refuse to start if the secret is missing or shorter than 32 characters.

## CLI Commands

```
gatheryourdeals init                    # Create database and admin account
gatheryourdeals serve                   # Start the HTTP server
gatheryourdeals admin reset-password    # Reset a user's password
```

### Global flags

```
--config <path>    Path to config file (default: config.yaml)
```

## API Reference

The full API specification is in **[docs/api.yaml](docs/api.yaml)** (OpenAPI 3.0). Load it into any of these tools for an interactive view:

- **Swagger UI** — paste the URL or file into [editor.swagger.io](https://editor.swagger.io)
- **Postman** — Import → Link or file
- **VS Code** — [OpenAPI (Swagger) Editor](https://marketplace.visualstudio.com/items?itemName=42Crunch.vscode-openapi) extension

For quick curl examples, see **[docs/api_examples.md](docs/api_examples.md)**.

## Documentation

- [OpenAPI Spec](docs/api.yaml) — full API specification
- [API Examples](docs/api_examples.md) — curl examples for every endpoint
- [Connection and Authentication](docs/connection_and_auth.md) — hosting, auth design, access keys
- [Data Format](docs/data_format.md) — purchase record format, metadata, ETL process
- [Service Structure](docs/service_structure.md) — project layout, design decisions

## Key Features

- **Single binary** — server and admin CLI in one executable
- **Docker support** — multi-stage build, persistent volume for database
- **JWT authentication** — stateless access tokens, rotating refresh tokens stored in SQLite
- **Role-based access** — admin and user roles enforced on every request
- **SQLite with WAL mode** — lightweight, concurrent reads, no setup required
- **Swappable database** — repository pattern allows switching to PostgreSQL
- **Embedded migrations** — schema managed by goose, compiled into the binary
- **YAML configuration** — server port, database path, token expiry all configurable
