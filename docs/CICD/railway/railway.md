# CI/CD with Railway

For branch strategy, required checks, and workflow triggers — see `docs/CICD/CICD.md`.

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

**Inputs:** `phase`: `moderate` (default) or `stress`

**What it does:**
1. Deploys current branch to load-test
2. Resets the database: `TRUNCATE receipts CASCADE; DELETE FROM users WHERE role != 'admin'`
3. Re-seeds 1,000 receipts via `seed.py`
4. Runs `run_all.sh` (all 4 Locust groups)
5. Uploads results as a GitHub Actions artifact (retained 30 days)

## Checklist Before Pushing

- [ ] `RAILWAY_TOKEN` is set in GitHub secrets and scoped to `load-test`
- [ ] `GYD_ADMIN_USERNAME/PASSWORD`, `GYD_TEST_USERNAME/PASSWORD` are set
- [ ] Postgres service is deployed in the `load-test` Railway environment
- [ ] Docker build passes locally: `docker build -t test .`
- [ ] Unit tests pass: `go test ./...`
- [ ] No lint errors: `golangci-lint run`
