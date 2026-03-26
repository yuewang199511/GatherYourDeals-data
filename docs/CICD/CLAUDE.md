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
