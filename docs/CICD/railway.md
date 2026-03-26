# CI/CD with Railway — Structure, Process, and Gotchas

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
| `integration-tests.yml` | PR → develop | Pre-merge: local build + test (no Railway needed) |
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
| `RAILWAY_TOKEN` | integration-tests (post-merge), load-tests | Project token scoped to load-test env (see token scope section below) |
| `GYD_JWT_SECRET` | integration-tests (pre-merge) | Must be ≥ 32 characters |
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
  - `load-test` (ID: `9000f540-3e22-4bf5-b97f-3e50f8f14b7c`) — CI post-merge + manual load tests

## Token Scope — Critical

**Railway project tokens are environment-scoped.** A token created for `production` cannot access `load-test`, and vice versa.

The `RAILWAY_TOKEN` secret must be created for the **`load-test` environment** specifically:
```
Railway dashboard → Project → Settings → Tokens → New token → select "load-test"
```

**Do not use an account-level (personal) token** unless absolutely necessary — it grants access to all Railway resources. The project token for `load-test` is sufficient for all CI workflows.

**Why this matters:** We tried using Railway Preview Environments (one per PR branch) for pre-merge testing. This was abandoned because preview environments are dynamic and Railway project tokens cannot access them — only account-level tokens can. The solution was to run pre-merge tests locally instead.

## Pre-merge vs Post-merge Integration Tests

### Pre-merge (PR → develop)
- Builds the Go binary directly on the runner (CGO_ENABLED=1)
- Initialises a local SQLite DB with admin credentials
- Starts the server on localhost:8080
- Runs pytest against localhost
- **No Railway token required**
- Fast (~1–2 min)

### Post-merge (push → develop)
- Uses Railway CLI to deploy current develop to the load-test environment
- Resolves the load-test service domain via `railway domain`
- Waits for health check to pass
- Runs the same pytest suite against the Railway URL
- **Requires `RAILWAY_TOKEN`** (load-test scoped)
- Runs are serialised via concurrency group `post-merge-develop` to prevent concurrent merges from racing on the shared load-test environment

## Load Test Workflow

Triggered manually from GitHub Actions UI on the `develop` branch.

**Inputs:**
- `phase`: `moderate` (default) or `stress`

**What it does:**
1. Links Railway CLI to load-test environment
2. Deploys current `develop` branch to load-test
3. Resets the database: `TRUNCATE receipts CASCADE; DELETE FROM users WHERE role != 'admin'`
4. Re-seeds 1,000 receipts via `seed.py`
5. Runs `run_all.sh` (all 4 Locust groups, moderate then stress phase)
6. Uploads results as a GitHub Actions artifact (retained 30 days)

The reset+reseed on every run ensures results are comparable across runs regardless of accumulated write-test data.

## Checklist Before Pushing to a PR Branch

- [ ] `RAILWAY_TOKEN` is set in GitHub secrets and scoped to `load-test`
- [ ] `GYD_JWT_SECRET`, `GYD_ADMIN_USERNAME/PASSWORD`, `GYD_TEST_USERNAME/PASSWORD` are set
- [ ] Docker build passes locally: `docker build -t test .`
- [ ] Unit tests pass: `go test ./...`
- [ ] No lint errors: `golangci-lint run`

## Known Failure Patterns for Future Agents

### 1. `railway link` does not accept a positional project ID
**Wrong:** `railway link "7639518c-..."`
**Right:** `railway link -p "7639518c-..." -e "load-test" -s "GatherYourDeals-data"`

The CLI v3 uses flags, not positional arguments.

### 2. `set -e` exits on failed command substitution
GitHub Actions runs bash with `set -e`. This pattern **silently exits** the script if the command fails:
```bash
OUTPUT=$(failing_command)   # ← script exits here if command fails
if [ $? -eq 0 ]; then ...   # ← never reached
```
**Fix:** Put the command directly in the `if` condition:
```bash
if OUTPUT=$(failing_command 2>&1); then
  # success
fi
```
This is safe under `set -e` because bash treats `if`-condition failures as expected branching, not errors.

### 3. Railway project tokens cannot access preview environments
Railway creates dynamic preview environments per PR branch. Project tokens are environment-scoped and return `Unauthorized` when attempting to access any environment they weren't created for.

**Do not attempt** to use project tokens with preview environments — it will always fail.
**Solution:** Run pre-merge tests locally (no token needed) and reserve Railway for post-merge against a fixed, known environment.

### 4. `railway domain` returns empty when service not linked
If `railway link` succeeded for the environment but not the service, `railway domain` returns "Project does not have any services."

Always specify the service explicitly:
```bash
railway domain -s "GatherYourDeals-data" --json | jq -r '.domain // empty'
```

### 5. Raw Railway GraphQL API requires account-level token
Querying `project { environments { ... } }` via the Railway GraphQL API (`backboard.railway.app/graphql/v2`) requires an account-level personal token, not a project token. Project tokens return `Not Authorized` on this query.

**Use the Railway CLI instead** — it handles auth internally and works with project tokens for supported operations.
