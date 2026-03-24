# GatherYourDeals-data Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-03-22

## Active Technologies
- Go 1.25.7 + Gin (HTTP), `database/sql` (DB abstraction), Goose v3 (migrations — not used by this feature), `mattn/go-sqlite3` (SQLite), `pgx/v5` (PostgreSQL) (002-api-list-pagination)
- SQLite (primary) + PostgreSQL (feature 001); both must be updated identically in behaviour (002-api-list-pagination)
- Go 1.25.7 (service); Python 3.11 + Locust 2.32 (load testing) (004-add-otel-honeycomb)
- SQLite (primary) + PostgreSQL; both instrumented via `otelsql` driver wrapper (004-add-otel-honeycomb)
- Python 3.11 + `locustio/locust:2.32.0`, `requests` (HTTP), `concurrent.futures` (stdlib) (005-fix-logout-group-pool)
- N/A — in-process `queue.Queue` only; results written to `load_testing/results/` (existing) (005-fix-logout-group-pool)
- Python 3.11 (integration tests); Go 1.25 (service under test — unchanged) + `pytest>=8.0`, `requests>=2.31`, `python-dotenv==1.0.1` (all in venv) (006-integration-tests)
- SQLite (default local) or PostgreSQL — no changes; tests hit the live DB via HTTP (006-integration-tests)
- Python 3.11 (integration tests); Bash (runner scripts) + pytest ≥ 8.0, requests ≥ 2.31, python-dotenv 1.0.1 — all already in `load_testing/.venv`; no new dependencies required (007-seed-guardrail-tests)
- N/A — guardrail is read-only against the live service (007-seed-guardrail-tests)

- Go 1.25 (as declared in `go.mod`) + Gin (HTTP), Cobra (CLI), Goose v3 (migrations), `database/sql` (DB abstraction); adding `pgx/v5` (PostgreSQL driver, pure Go) (001-add-postgres-support)
- Python 3.11 + Locust 2.32 (headless load testing); Docker Compose; Bash runner scripts; `load_testing/` directory (003-load-testing-env-config)

## Project Structure

```text
src/
tests/
```

## Commands

# Add commands for Go 1.25 (as declared in `go.mod`)

## Code Style

Go 1.25 (as declared in `go.mod`): Follow standard conventions

## Recent Changes
- 007-seed-guardrail-tests: Added Python 3.11 (integration tests); Bash (runner scripts) + pytest ≥ 8.0, requests ≥ 2.31, python-dotenv 1.0.1 — all already in `load_testing/.venv`; no new dependencies required
- 006-integration-tests: Added Python 3.11 (integration tests); Go 1.25 (service under test — unchanged) + `pytest>=8.0`, `requests>=2.31`, `python-dotenv==1.0.1` (all in venv)
- 005-fix-logout-group-pool: Added Python 3.11 + `locustio/locust:2.32.0`, `requests` (HTTP), `concurrent.futures` (stdlib)


<!-- MANUAL ADDITIONS START -->
## Actual Project Structure

```text
cmd/gatheryourdeals/main.go          # Entry point: serve / init / admin subcommands
internal/
  auth/                              # JWT issuance, refresh token lifecycle, password hashing
  config/                            # Config struct (YAML + env var resolution)
  handler/                           # HTTP handlers (auth, user, meta, receipt, router)
  middleware/                        # Bearer token validation
  model/                             # Domain structs (User, Receipt, MetaField)
  repository/
    repository.go                    # Interface definitions (UserRepository, ReceiptRepository, MetaFieldRepository)
    sqlite/                          # SQLite implementations + embedded goose migrations
    postgres/                        # PostgreSQL implementations + embedded goose migrations (feature 001)
docs/                                # OpenAPI spec, API examples, service structure docs
specs/                               # Feature specs, plans, research (speckit workflow)
```

## Key Commands

```bash
go build -o gatheryourdeals ./cmd/gatheryourdeals   # build
go test ./...                                        # run all tests
./gatheryourdeals init                               # create DB + admin account
./gatheryourdeals serve                              # start server on :8080

# PostgreSQL dev (feature 001):
TEST_POSTGRES_DSN="postgres://..." go test ./internal/repository/postgres/...
```

## Architecture Notes

- Repository pattern: all DB access goes through interfaces in `internal/repository/repository.go`
- Both SQLite and PostgreSQL implementations must satisfy the same interfaces
- `main.go openDatabase()` is the single switch point for backend selection
- SQLite uses `?` placeholders; PostgreSQL uses `$1, $2, ...`
- Migrations embedded via `go:embed` and run automatically at startup via goose
- JWT secret loaded from `GYD_JWT_SECRET` env var; server refuses to start without it

## Deployment Intent

- **SQLite** is for local development only — do not treat it as a production target
- **PostgreSQL** is the production and scaled deployment target; horizontal scaling requires it
- **Redis** is the planned future store for refresh tokens/sessions — native TTL eliminates orphaned token buildup, and shared state works across multiple server instances; not yet implemented

## Workflow

After completing every spec implementation (i.e., after running `/speckit.implement`), run `/simplify` to review the changed code for reuse, quality, and efficiency issues and fix them.

## Operational Notes

### SQLite Snapshot Restore (Docker)
Always stop the app container before copying a snapshot into place:
```bash
docker compose stop app
cp snapshot.db /path/to/data.db
docker compose start app
```
If the service is running when you copy, Docker keeps the old file descriptor open and the copy lands on a new inode that the running process never sees — the restore silently has no effect.

## load testing monitoring with few tokens

Please save tokens by following these guidelines when performing load testing:
1. only look at the failure records, you can even only check the records after the experiment finished rather than always follow it

## python running guidelines
1. use venv for activating the environment

## load testing workflow

Always use `docker compose` to run the service for load testing. Never use the local binary.

```bash
# 1. Start the service
docker compose up --build -d

# 2. Seed the database
cd load_testing && python3 seed.py

# 3. Run the full load test suite
./run_all.sh

# 4. Clean up when done
cd .. && docker compose down
```

The guardrail (`run_guardrail.sh`) runs automatically inside `run_all.sh` and `run_group.sh` before any Locust traffic starts. If the seed state is bad, the load test aborts before sending requests.
<!-- MANUAL ADDITIONS END -->
