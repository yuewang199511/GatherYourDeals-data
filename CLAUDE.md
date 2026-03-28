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

Please refer to [docs/service_structure.md](docs/service_structure.md)

## Code Style

Go 1.25 (as declared in `go.mod`): Code must pass `gofmt` and `golangci-lint` (config: `.golangci.yml`). Both run in CI via `.github/workflows/code-quality.yml` and are the authoritative style gate.

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


# Agent Role

If you are reading this at the begining, you are the master agent who will be orchestrating the development, test, revision, and report escalation or summary from subagents.

## Write / Commit / Push — Master Agent Only

Only the master agent may write files, run `git commit`, `git push`, or create/merge PRs.

Subagents are **read-only**. They investigate, analyze, and produce fix plans or report content, then return everything to the master agent. The master agent reviews, applies all changes, commits, and pushes.

Please scan the subfolders at most 2 levels for other CLAUDE.md for definition of subagents.

Please also revisit the README.md and verify that every command documented there (1) exists in the codebase or PATH, and (2) runs successfully in the local environment without error. Local deployment is the baseline — if a command works locally, it is considered a valid entrypoint for remote deployment as well.

If you receive any request that needs to start those subagents defined in this repo, let the user know.

# environment prerequisites

At the start of any task, verify the following required CLI tools are installed by running `which <tool>` for each:
- `docker`
- `gh`
- `jq`
- `railway`

If any are missing, stop immediately and tell the user the exact install command. Do not proceed until all four are available.

# Tracing Prerequisites

Before starting any testing workflow that involves Honeycomb, verify all three prerequisites:

1. **Honeycomb MCP** — run `claude mcp list` and confirm `honeycomb` appears and is connected.
   If missing or showing "Needs authentication", tell the user to run:
   ```
   claude mcp remove honeycomb && claude mcp add honeycomb --transport http https://mcp.honeycomb.io/mcp --header "Authorization: Bearer <KEY_ID>:<SECRET_KEY>"
   ```
   Use a **Management API key** (not an ingest key) — ingest keys are write-only and cannot query data.
   Key format: `hcamk_<id>:<secret>` from Honeycomb → Team Settings → API Keys → Query permissions.

   **IMPORTANT — session restart required:** MCP tools are registered once at session start. If the MCP was added or re-added during the current session, the tools will NOT be available until the user starts a new Claude Code session. Do not attempt Honeycomb queries in the same session where the MCP was reconfigured — stop and ask the user to restart.

   **Do NOT delegate Honeycomb queries to subagents** — subagents cannot call MCP tools from the parent session context. All Honeycomb tool calls must be made directly by the master agent.

2. **Ingest key** — root `.env` must contain `OTEL_EXPORTER_OTLP_HEADERS=x-honeycomb-team=<key>` (required for the service to send traces).

3. **Load test API key** — `load_testing/.env` must contain `HONEYCOMB_API_KEY` (required for load test event posting).


If any prerequisite is missing, stop and notify the user immediately — do not proceed with Honeycomb queries.

# Circuit Breaker Rules

To prevent agents from consuming excessive tokens on dead or failing external calls:

- Any external call that fails must be retried at most **3 times**. Failure is defined as:
  - **HTTP**: 4xx or 5xx response code
  - **MCP tool call**: tool returns an error result
  - **Bash/CLI tool**: non-zero exit code
- If all 3 retries fail, stop and report to the user immediately — do not continue
- Non-retryable failures (HTTP 401, 403, 404; MCP auth errors; CLI "not found" errors) must not be retried at all — stop and report immediately
- Do not loop on the same failing endpoint in search of a different result
- If a tool or endpoint is unavailable, treat it as a blocked state and follow the Stop / Blocked rule below

This applies to all agents and subagents.

# Multi-Agent Fix Coordination

When multiple subagents return reports with fix suggestions, the master agent follows these rules before applying anything.

## 1. Conflict detection first

Before writing any fix, scan all pending reports for overlapping file paths. If two agents suggest changes to the same file:
- **Stop immediately**
- Present both suggestions to the user with the conflicting file highlighted
- Wait for explicit resolution before proceeding

Do not attempt to merge or reconcile conflicting suggestions autonomously.

## 2. Apply one fix at a time

Never batch fixes from multiple agents into a single commit. The sequence is:

```
apply fix → commit → wait for CI → verify → apply next fix
```

If CI fails after a fix, stop and resolve before moving on. Do not stack unverified fixes.

## 3. Priority order (when no conflict, just sequencing)

Apply in this order — each layer unblocks the one below:

| Priority | Domain | Agent |
|---|---|---|
| 1 | Service / business logic bugs | `internal/` agent |
| 2 | CI/CD / deployment failures | `docs/CICD/` agent |
| 3 | Test code fixes | `load_testing/` agents |

## 4. Dependent fixes

If agent B's fix assumes agent A's fix is already in place, apply A first and verify before applying B. If the dependency is unclear, ask the user.

## 5. Contradictory root causes

If two agents diagnose the same symptom differently, do not pick one arbitrarily. Present both analyses to the user and wait for a decision.

---

# Autonomy Boundary Map

These rules govern how much agents can act independently versus when they must stop and involve the user.

# IMPORTANT FOR GIT

PLEASE ALWAYS CHECK IF BRANCH IS UP TO DATE WITH TARGET IF YOU HAVE A PR!!!

## Pre-merge branch sync check

Before merging any PR (to `develop` or `main`), run:
```bash
git fetch origin
git log HEAD..origin/<base-branch> --oneline
```
If any commits are listed, the branch is behind — **rebase first, then merge**. Do not merge a stale branch.

## During Execution
Run uninterrupted. Do not pause to ask questions mid-task unless blocked.

## Code Replan
If a fix requires changing code only — not the overall approach or architecture — the agent may proceed autonomously.
The user will review changes after the fact via git history and PR description.

For every code change:
- Each commit message must explain what was changed and why for that specific change
- When creating a PR, write the agent report as the PR description
- If CI fails and further fixes are made, rewrite the PR description to summarize the full fix journey — original issue, what CI caught, what was changed, and why

## Strategy Replan
If a fix requires changing the overall approach — architecture, testing strategy, data model, or any decision that affects direction downstream — the agent must stop immediately.

Send a strategy escalation message containing:
1. What I did
2. What I see is wrong
3. What I suggest trying next
4. Why I think this will fix it

Then wait for explicit user approval before proceeding.

## Stop / Blocked
If the agent cannot proceed — missing credentials, access denied, ambiguous requirements, or a situation outside all defined rules — stop and report to the user immediately.

## Summary

| Situation | Action |
|---|---|
| Normal execution | Run uninterrupted |
| Code-level fix needed | Fix autonomously, commit with reasoning, user reviews after |
| Approach or architecture change needed | Stop, send strategy escalation, wait for user |
| Blocked or missing access | Stop, report to user immediately |

# Agent Report Format

All agents must follow the shared report skeleton defined in `docs/testing/report_format.md`.
Each agent's CLAUDE.md defines its own extension fields.

# auto code optimization after every change

After completing any code changes, the implementing subagent must run `/simplify` on its own changes before reporting back to the master agent. The master agent does not run `/simplify` itself.

# permision

if you need sudo permission, please ask me

# Missing CLI Tools

If a required CLI tool is not installed (e.g. `gh`, `docker`, `jq`), do not silently fall back to an alternative approach. Instead:

1. Stop immediately
2. Tell the user exactly what is missing and the install command, e.g.:
   > `gh` is not installed. Run: `! sudo apt install gh` then `! gh auth login`
3. Wait for the user to install it

Once the user installs the tool in the same session (using `! <command>`), it activates immediately — no restart needed. Resume from where you stopped.


<!-- MANUAL ADDITIONS END -->
