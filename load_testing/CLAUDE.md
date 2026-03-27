# Testing master

You job is to run higher level testing according to the user specification, which mostly may have variations in running from a remote provider(or several) or locally.

# read scope

Read anywhere within `load_testing/` and all its subfolders (including `integration/` and `locust/`).

# editing scope

Only write within `load_testing/` root. Do not edit files inside `integration/` or `locust/` — those are owned by their respective subagents.

# Local vs Remote

Check `GYD_TARGET_URL` in `load_testing/.env` to determine the test environment:
- If `GYD_TARGET_URL` contains `localhost` or is unset — **local test**
- If `GYD_TARGET_URL` is anything else — **remote test**

If only running integration test, just push and let integration test subagent to check result from repo host

## Local test setup
Before running any tests locally:
1. Restore the snapshot and start the service using the SQLite Snapshot Restore instructions below
2. Verify the service is healthy: `curl -sf http://localhost:8080/health`
3. If the service is already running and healthy, skip restore and startup — do not restart unnecessarily

## Remote test setup
Do not touch local service or snapshot.
Verify the remote service is healthy before running tests: `curl -sf {GYD_TARGET_URL}/health`
If the health check fails, stop and report to the user — do not proceed.

## Remote seeding strategy

When seeding against a remote provider (Railway, Azure), the correct fix for seed timeouts is **reducing workers**, not increasing timeout:

| Symptom | Wrong fix | Right fix |
|---|---|---|
| `ReadTimeout` during seeding | Increase `GYD_SEED_TIMEOUT` | Reduce `GYD_SEED_WORKERS` |
| Seed completes but slow | Increase workers | Acceptable — seed is pre-test only |

**Why:** Remote services have CPU/connection limits. High concurrency (10 workers default) can saturate a single-instance Railway service or exhaust PostgreSQL connections. 3 workers is the validated safe value for Railway hobby/single-instance.

Increasing timeout alone does not fix the root cause (CPU saturation) — it just makes each timed-out request wait longer before failing.

**Configured values for Railway CI:** `GYD_SEED_WORKERS=10`, `GYD_SEED_TIMEOUT=40` (set in workflow env).

The 3-worker value was a temporary conservative fix while parallel runs were causing resource contention. Once concurrency was serialized per provider, 10 workers is safe. 40s covers Railway's bcrypt login spike (~11s avg moderate, up to ~16s under light load) with headroom.

# Test Execution Order

Always run in this order — do not run load tests if integration tests fail:
1. Integration tests first — run integration subagent, wait for pass
2. Load tests second — only start locust subagent after integration tests pass

If integration tests fail, stop and report to the user before proceeding to load tests.

# Tasks

1. Before running tests, verify the seed snapshot matches the requirements in `docs/testing/load_testing.md` — specifically: 1,000 purchase receipts pre-seeded under a single user account. If the snapshot does not match (wrong record count, missing user, schema mismatch), stop and escalate to the user — do not proceed with tests until the user approves a seed rebuild.
2. Receive reports from subagents about test results and fix plans. After each fix cycle, propose additions to the relevant CLAUDE.md (e.g. `docs/CICD/railway/KNOWN_ISSUES.md` for Railway infra patterns, subagent CLAUDE.md for test-specific lessons). Present the proposed additions to the user for approval before writing them.
3. Only let the subagents to perform the fixing of testing codes and also the testing plans in docs/testing.
4. If code needs to be fixed outside of test — business code or service setup — report to master agent.

# escalation rule

In order to make testing robust and fixing solutions repeatable. 
1. Please also ask user if fix needs to have new guardrails to prevent same testing issue happens.
2. Ask the user if there are some fixing rules/strategies should be saved in CLAUDE.md for each subagent for future reuse. In order to recude review time.

# Report Extension

Follow the skeleton in `docs/testing/report_format.md`.

Under `### Extension` include:

**Agents involved:** [list of subagents that ran]
**Service code issue found:** [yes | no — if yes, forwarded to master agent]
**New guardrail needed:** [yes | no — escalate to user if yes]
**Rules to save in CLAUDE.md:** [any fixing strategies worth persisting for future reuse]

---

### SQLite Snapshot Restore and Start (Docker)

All commands below must be run from the **repo root** (not from `load_testing/`).

Always stop the app container before copying a snapshot into place:
```bash
# From repo root:
docker compose stop app
cp load_testing/seed_snapshot.db data/db/gatheryourdeals.db
docker compose up -d
```

Do NOT use `--build` here — no code changed, rebuilding is unnecessary and slow.

If the service is running when you copy, Docker keeps the old file descriptor open and the copy lands on a new inode that the running process never sees — the restore silently has no effect.

Verify the service is healthy after startup:
```bash
curl -sf http://localhost:8080/health
```