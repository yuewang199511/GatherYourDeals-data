# Agent Guide — CI/CD Debugging

## Rule 1: Check Railway logs before anything else

When any Railway deployment fails, run these two commands first — before reading workflow files, CI logs, or guessing:

```
mcp__railway-mcp-server__list-deployments   → get deployment IDs and statuses
mcp__railway-mcp-server__get-logs (logType: build, deploymentId: ...)   → Docker build errors
mcp__railway-mcp-server__get-logs (logType: deploy, deploymentId: ...)  → runtime/startup errors
```

CI logs show symptoms ("health check failed", exit code 22). Railway logs show the actual cause.

## Rule 2: Check both build AND deploy logs

- **Build failure** → look at `logType: build` (Docker build errors, missing files, compile errors)
- **Health check failure** → look at `logType: deploy` (DB connection errors, missing env vars, crash on startup)

## Known Failure Patterns

### 1. Do not call `railway link` with a project token
Project tokens have project + environment context embedded. `railway link -p ... -e ...` returns `Unauthorized`.

**Right:** Set `RAILWAY_TOKEN` and call `railway up` / `railway domain` directly — no link needed.

### 2. `railway up` respects `.gitignore` — unanchored patterns exclude subdirectories
`railway up` applies `.gitignore` when building the upload archive. An unanchored entry like `gatheryourdeals` matches `cmd/gatheryourdeals/` at any depth, causing `stat /app/cmd/gatheryourdeals: directory not found` in the Docker build.

**Fix:** Anchor to root with a leading `/`: `/gatheryourdeals`

Note: `docker build .` uses `.dockerignore` (not `.gitignore`), so local and CI Docker Build checks pass even when this bug is present.

### 3. `railway domain` returns `{domains: [...]}`, not `{domain: "..."}`
**Wrong:** `jq -r '.domain // empty'`
**Right:** `jq -r '.domains[0] // empty'`

The URL already includes `https://` — do not prepend it again.

### 4. Project tokens cannot access preview environments
Railway project tokens are environment-scoped. They return `Unauthorized` for any environment they weren't created for, including dynamic preview environments.

### 5. `set -e` silently exits on failed command substitution
```bash
OUTPUT=$(failing_command)   # ← exits here under set -e
if [ $? -eq 0 ]; then ...   # ← never reached
```
**Fix:** `if OUTPUT=$(failing_command 2>&1); then` — commands in `if` conditions don't trigger `set -e`.

### 6. Postgres must be deployed in the target environment
The `load-test` environment needs its own Postgres instance. If `postgres.railway.internal` can't resolve, the Postgres service is not deployed in that environment.

**Fix:** Railway dashboard → switch to `load-test` → Postgres service → Deploy.

### 7. Raw Railway GraphQL API requires account-level token
`project { environments { ... } }` via `backboard.railway.app/graphql/v2` returns `Not Authorized` with a project token. Use the Railway CLI instead.

### 8. `DOMAIN=$(railway domain ...)` silently exits under `set -e` when the command fails
GitHub Actions runs bash with `-eo pipefail` by default. Assigning via `VAR=$(cmd)` without an `if` guard triggers `set -e` on failure — the script exits before any empty-check runs.

**Wrong:**
```bash
DOMAIN=$(railway domain -s "GatherYourDeals-data" --json | jq -r '.domains[0] // empty')
if [ -z "$DOMAIN" ]; then ...   # never reached if railway domain fails
```
**Right:** use `if !` to capture both failure and empty result:
```bash
if ! DOMAIN=$(railway domain -s "GatherYourDeals-data" --json 2>&1 | jq -r '.domains[0] // empty'); then
  echo "ERROR: railway domain command failed: $DOMAIN"
  exit 1
fi
if [ -z "$DOMAIN" ]; then ...
```
This was present in both `integration-tests.yml` and `load-tests.yml`.

### 9. `railway run -- sh -c 'psql ... -f /tmp/file.sql'` is fragile — use `-c` instead
`railway run` injects Railway env vars and runs the command **locally** on the CI runner. Writing SQL to a temp file and passing it via `-f` works, but is needlessly fragile (file may not exist if a prior step exited early under `set -e`). Pass the SQL inline with `-c`:

**Wrong:**
```bash
echo "TRUNCATE ..." > /tmp/reset.sql
railway run --service "..." -- sh -c 'psql "$DATABASE_PUBLIC_URL" -f /tmp/reset.sql'
```
**Right:**
```bash
RESET_SQL="TRUNCATE ..."
if ! railway run --service "..." -- psql "$DATABASE_PUBLIC_URL" -c "$RESET_SQL"; then
  echo "ERROR: Database reset failed"
  exit 1
fi
```
The `if !` guard also prevents silent exit under `set -e`. This was present in both `integration-tests.yml` and `load-tests.yml`.

### 10. `railway link` returns "Unauthorized" after CLI version upgrade — not a token problem

After a Railway CLI version upgrade, `railway link "<project-id>"` (positional arg) returns `Unauthorized` instead of a helpful "unrecognized argument" error. This looks like a token issue but is actually a CLI breaking change.

**Diagnosis:** If `railway whoami` succeeds locally but `railway link "<id>"` returns `Unauthorized` in CI, it is the CLI syntax that changed, not the token.

**Fix:** Remove `railway link` entirely. Project tokens already carry project/environment context. Use service-name flags directly:
```bash
railway up --service "GatherYourDeals-data"
railway domain -s "GatherYourDeals-data" --json
railway run --service "GatherYourDeals-data" -- <cmd>
```

### 11. Concurrent load test runs exhaust PostgreSQL connection limit

If two load test workflow runs trigger against the same Railway environment simultaneously, the Go service connection pools across both runs exceed Railway PostgreSQL's connection limit (`FATAL: sorry, too many clients already`).

**Cause:** Per-run-id concurrency groups allow parallel runs; each run opens its own connection pool against the shared PostgreSQL instance.

**Fix:** Use per-provider concurrency group (not per-run-id) so Railway runs serialize:
```yaml
concurrency:
  group: ${{ inputs.provider }}-load-test
  cancel-in-progress: false
```
