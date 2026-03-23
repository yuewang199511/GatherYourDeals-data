# GatherYourDeals — Load & Integration Testing

This directory contains two test suites that share the same virtual environment
and configuration:

- **Integration tests** (`integration/`) — pytest suite that verifies all HTTP
  endpoints against a live server. Fast (\<30 s). Run before load tests.
- **Load tests** (`locust/`) — Locust-based performance suite with four groups
  covering CPU-bound, read-heavy, write, and miscellaneous endpoints.

---

## Prerequisites

1. The server is running and reachable (Docker Compose or local binary).
2. The admin account has been created via `./gatheryourdeals init`.
3. `load_testing/.env` is populated (copy from `.env.example`).

---

## One-Time Setup

```bash
cd load_testing
python3 -m venv .venv
.venv/bin/pip install -r requirements.txt
```

---

## Integration Tests

### Run

```bash
cd load_testing
./run_integration.sh
```

Exits 0 on full success. Any failure prints the failing test name and assertion detail.

### What it tests

All 15 HTTP endpoints with happy-path and negative-path scenarios:

| Group      | Endpoints |
|------------|-----------|
| Health     | `GET /health` |
| Auth       | `POST /api/v1/users`, `/auth/login`, `/auth/refresh`, `/auth/logout`, `GET /auth/me` |
| Receipts   | `POST`, `GET`, `GET /:id`, `DELETE /:id` under `/api/v1/receipts` |
| Users      | `GET /api/v1/users`, `DELETE /api/v1/users/:id` (admin) |
| Meta       | `GET /api/v1/meta`, `POST /api/v1/meta`, `PUT /api/v1/meta/:fieldName` (admin) |

### Run directly with pytest

```bash
cd load_testing
.venv/bin/python3 -m pytest integration/ -v
```

---

## Load Tests

### Seed test data (required before first load test run)

```bash
cd load_testing
.venv/bin/python3 seed.py
```

Save a snapshot so you can restore before each run:

```bash
# Docker Compose (from repo root):
cp data/db/gatheryourdeals.db load_testing/seed_snapshot.db

# Restore before a test run (Docker):
docker compose stop app
cp load_testing/seed_snapshot.db data/db/gatheryourdeals.db
docker compose start app
```

### Run all groups

```bash
cd load_testing
./run_all.sh
```

### Run a single group

```bash
./run_group.sh 2            # Group 2, both phases (moderate then stress)
./run_group.sh 1 stress     # Group 1, stress phase only
./run_group.sh 3 moderate   # Group 3, moderate phase only
```

Groups: `1` = login (CPU-bound), `2` = reads, `3` = writes, `4` = misc.

---

## Environment Variables

All variables can be set in `load_testing/.env` (copy from `.env.example`).

| Variable              | Required | Default                | Description |
|-----------------------|----------|------------------------|-------------|
| `GYD_TEST_USERNAME`   | Yes      | —                      | Regular test account username |
| `GYD_TEST_PASSWORD`   | Yes      | —                      | Regular test account password |
| `GYD_ADMIN_USERNAME`  | Yes      | —                      | Admin account username |
| `GYD_ADMIN_PASSWORD`  | Yes      | —                      | Admin account password |
| `GYD_TARGET_URL`      | No       | `http://localhost:8080`| Base URL of the server |
| `GYD_PLATFORM_NAME`   | No       | `local`                | Label for results (load tests) |
| `GYD_DB_TYPE`         | No       | `sqlite`               | DB label for results (load tests) |
| `GYD_RPS_MODERATE`    | No       | `100`                  | Moderate-phase RPS target |
| `GYD_RPS_STRESS`      | No       | `500`                  | Stress-phase RPS target |
| `GYD_PHASE`           | No       | `moderate`             | Phase when running Locust directly |
| `HONEYCOMB_API_KEY`   | No       | —                      | Send summary events to Honeycomb |
