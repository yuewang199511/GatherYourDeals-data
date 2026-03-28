# CI/CD with Railway — Structure and Process

## Branch Strategy

```
feature/* ──PR──► develop ──PR──► main
```

- `main`: protected — only PRs from `develop` are allowed (`protect-main` check enforces this)
- `develop`: protected — 4 required checks must pass before merge
- `feature/*`: open — PRs target `develop`

## Workflow Overview

| File | Trigger | Purpose |
|---|---|---|
| `integration-tests.yml` | PR → develop | Pre-merge: deploy to Railway load-test + test |
| `integration-tests.yml` | push → develop | Post-merge: deploy to Railway load-test + test |
| `load-tests.yml` | Manual (`workflow_dispatch`) | Full Locust suite against Railway load-test |
| `protect-main.yml` | PR → main | Blocks merge if source branch ≠ develop |
| `test.yml` | PR/push → develop or main | Go unit tests + race detector |
| `build.yml` | PR/push → develop or main | Docker build check |
| `code-quality.yml` | PR/push → develop or main | golangci-lint |
| `security.yml` | PR/push → develop or main | govulncheck |

## Required GitHub Secrets

| Secret | Used by | Notes |
|---|---|---|
| `RAILWAY_TOKEN` | integration-tests, load-tests | Project token scoped to **load-test** env only — no production deploy workflow exists yet, so one token is enough |
| `GYD_ADMIN_USERNAME` | integration-tests, load-tests | Admin account credentials |
| `GYD_ADMIN_PASSWORD` | integration-tests, load-tests | Admin account credentials |
| `GYD_TEST_USERNAME` | integration-tests, load-tests | Test user credentials |
| `GYD_TEST_PASSWORD` | integration-tests, load-tests | Test user credentials |
| `HONEYCOMB_API_KEY` | load-tests | Optional — for Honeycomb event posting |

## Railway Project

- **Project ID**: `7639518c-e0a7-4cd4-8227-6c2575458620`
- **Service ID** (GatherYourDeals-data): `b491569f-4c17-4e1d-a813-b8c5e73f4d6c`
- **Environments**:
  - `production` (ID: `a4461018-585b-44e2-9f94-28b46c7def11`) — live traffic
  - `load-test` (ID: `9000f540-3e22-4bf5-b97f-3e50f8f14b7c`) — CI + manual load tests

## Token Scope

**Railway project tokens are environment-scoped.** A token created for `production` cannot access `load-test`.

The `RAILWAY_TOKEN` secret must be created for the **`load-test` environment** specifically:
```
Railway dashboard → Project → Settings → Tokens → New token → select "load-test"
```

## Integration Tests

Both pre-merge (PR) and post-merge (push to develop) deploy to the same `load-test` Railway environment. Runs are serialised via `concurrency: group: load-test-env` to prevent concurrent deploys racing on the shared environment.

## Load Test Workflow

Triggered manually from GitHub Actions UI.

**Inputs:**

| Input | Default | Notes |
|---|---|---|
| `provider` | `railway` | `railway` or `azure` |
| `group` | `all` | `all` or group number `1`–`4` |
| `phase` | `moderate` | `moderate`, `stress`, or `all` |
| `replicas` | `1` | Number of Railway replicas to deploy |

**What it does:**
1. Injects `numReplicas` into `railway.toml` (upsert — workflow input always wins over any value in the file)
2. Deploys current branch to load-test
3. Resets the database: `TRUNCATE receipts CASCADE; DELETE FROM users WHERE role != 'admin'`
4. Re-seeds 1,000 receipts via `seed.py`
5. Runs `run_all.sh` (all 4 Locust groups)
6. Uploads results as a GitHub Actions artifact (retained 30 days)

## Replica Scaling

Replica count is controlled via the `replicas` workflow input, not the Railway dashboard or `railway.toml` directly. This allows different replica counts per test run without touching infrastructure config.

**How it works:** The deploy step runs a TOML upsert before `railway up`:
```bash
if grep -q "^numReplicas" railway.toml; then
  sed -i "s/^numReplicas = .*/numReplicas = N/" railway.toml
else
  echo "numReplicas = N" >> railway.toml
fi
```

The workflow input always wins — if `numReplicas` is already in `railway.toml`, it gets replaced. If absent, it gets appended to the `[deploy]` section (which is last in the file).

**Connection pool headroom by replica count:**

| Replicas | Connections used | Headroom (max 100) |
|---|---|---|
| 1 | 10 + ~3 CI/goose | 87 |
| 2 | 20 + ~3 CI/goose | 77 |
| 5 | 50 + ~3 CI/goose | 47 |
| 9 | 90 + ~3 CI/goose | 7 (limit) |

`SetMaxOpenConns(10)` per replica. Do not exceed 9 replicas without increasing Railway PostgreSQL's `max_connections`.

## PostgreSQL Connection Architecture

PgBouncer runs in **session mode** between the app and PostgreSQL:

```
app replica → PgBouncer → PostgreSQL
```

In session mode each app connection maps 1:1 to a PostgreSQL server connection for its entire lifetime. The app-side pool cap is still required:

- `SetMaxOpenConns(10)` — max server connections per replica
- `SetMaxIdleConns(5)` — idle connections kept warm

**Pool math (unchanged by PgBouncer session mode):**
```
max_open_per_replica = (pg_max_connections - reserved) / num_replicas
```

With Railway PostgreSQL's `max_connections = 100` and 10 reserved, this supports up to 9 replicas safely.

**Why session mode (not transaction mode):** pgx prepared statements do not survive transaction-mode multiplexing, which would require switching to simple protocol. Session mode avoids this incompatibility.


## Checklist Before Pushing

- [ ] `RAILWAY_TOKEN` is set in GitHub secrets and scoped to `load-test`
- [ ] `GYD_ADMIN_USERNAME/PASSWORD`, `GYD_TEST_USERNAME/PASSWORD` are set
- [ ] Postgres service is deployed in the `load-test` Railway environment
- [ ] Docker build passes locally: `docker build -t test .`
- [ ] Unit tests pass: `go test ./...`
- [ ] No lint errors: `golangci-lint run`
