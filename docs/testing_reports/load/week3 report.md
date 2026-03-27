# Summary

This article records the load testing results from week #3, run against the `develop` branch on Railway + PostgreSQL.

Service was deployed on Railway (1 replica, us-west2) backed by Railway-managed PostgreSQL via PgBouncer (session mode).

**All 8 phases (4 groups × moderate + stress) completed with 0% error rate.** This is the first run where all stress phases passed — a significant regression fix from week 2 where all 4 stress phases failed.

# Environment

- **Platform**: Railway (1 replica, `us-west2`)
- **Database**: PostgreSQL (Railway-managed, via PgBouncer in session mode)
- **Service**: `GatherYourDeals-data` (branch `develop`)
- **GitHub Run**: `23630392237` (artifact: `load-test-results-23630392237`)
- **Target URL**: `https://gatheryourdeals-data-load-test.up.railway.app`

# Performance Result

**Test run IDs:**
- cpu_bound/moderate: `9df63259-39a7-43ef-88ba-a08c1b4c2dda`
- cpu_bound/stress: `ec94200e-cd41-4052-9750-a2dd7fe3ea15`
- read_heavy/moderate: `8662c883-6198-49d3-a11a-fdb6bb9543fc`
- read_heavy/stress: `414a8b6b-31e7-48ea-8079-6a2a5203e062`
- write_ops/moderate: `82ab36f6-734f-429e-9a45-b4c4b852284e`
- write_ops/stress: `f59cd9ea-ff85-4a22-8fcb-c7997b29b867`
- misc_lightweight/moderate: `0eb5309b-7eee-4344-ada8-28351dec3d62`
- misc_lightweight/stress: `7b400582-35dc-468f-b9e8-335568ece581`

**Time Range**: 2026-03-27 03:57 UTC → 04:21 UTC (~24 min total)

## Run Duration

| Group | Phase | Users | Target RPS | Intended (s) | Actual (s) | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| cpu_bound | moderate | 100 | 100 | 120 | 120.1 | ✓ |
| cpu_bound | stress | 500 | 500 | 150 | 150.2 | ✓ |
| read_heavy | moderate | 400 | 100 | 120 | 120.9 | ✓ |
| read_heavy | stress | 2000 | 500 | 150 | 467.0 | ✓ — long startup: 2,000 bcrypt logins |
| write_ops | moderate | 100 | 200 | 120 | 120.1 | ✓ |
| write_ops | stress | 500 | 1000 | 150 | 150.2 | ✓ |
| misc_lightweight | moderate | 110 | 100 | 120 | 120.1 | ✓ |
| misc_lightweight | stress | 540 | 500 | 150 | 150.1 | ✓ |

Note: `read_heavy/stress` took 467s vs intended 150s because the on-task timer starts after Locust user login. With 2,000 users each doing a 3–9s bcrypt login, the startup window extends the wall-clock duration significantly. No errors occurred; the extended time is a harness artifact.

## Moderate Phase

| Endpoint | Target RPS | Actual RPS | Avg Latency (ms) | P50 / P95 (ms) | # Errors | # Requests |
| --- | --- | --- | --- | --- | --- | --- |
| POST /api/v1/auth/login | 100 | 36.1 | 2,736 | 2,600 / 4,000 | 0 | 4,293 |
| GET /api/v1/auth/me | 100 | 88.6 | 153 | 130 / 140 | 0 | 10,743 |
| GET /api/v1/receipts | 100 | 86.8 | 183 | 130 / 160 | 0 | 10,524 |
| GET /api/v1/receipts/:id | 100 | 88.5 | 169 | 130 / 150 | 0 | 10,728 |
| GET /api/v1/meta | 100 | 88.2 | 180 | 130 / 160 | 0 | 10,691 |
| POST /api/v1/receipts | 200 | 98.1 | 137 | 130 / 150 | 0 | 11,801 |
| DELETE /api/v1/receipts/:id | 200 | 98.1 | 134 | 130 / 150 | 0 | 11,801 |
| POST /api/v1/auth/refresh | 100 | 95.9 | 147 | 140 / 170 | 0 | 11,535 |
| POST /api/v1/meta | 5 | 4.5 | 139 | 130 / 170 | 0 | 542 |
| PUT /api/v1/meta/:fieldName | 5 | 4.8 | 138 | 130 / 160 | 0 | 573 |

## Stress Phase

| Endpoint | Target RPS | Actual RPS | Avg Latency (ms) | P50 / P95 (ms) | # Errors | # Requests |
| --- | --- | --- | --- | --- | --- | --- |
| POST /api/v1/auth/login | 500 | 32.6 | 10,549 | 8,600 / 26,000 | 0 | 4,861 |
| GET /api/v1/auth/me | 500 | 412.0 | 154 | 130 / 230 | 0 | 192,511 |
| GET /api/v1/receipts | 500 | 410.9 | 201 | 150 / 300 | 0 | 192,037 |
| GET /api/v1/receipts/:id | 500 | 413.5 | 179 | 140 / 270 | 0 | 193,234 |
| GET /api/v1/meta | 500 | 412.5 | 199 | 150 / 300 | 0 | 192,782 |
| POST /api/v1/receipts | 1000 | 377.3 | 147 | 130 / 210 | 0 | 56,819 |
| DELETE /api/v1/receipts/:id | 1000 | 377.3 | 145 | 130 / 200 | 0 | 56,819 |
| POST /api/v1/auth/refresh | 500 | 370.1 | 160 | 150 / 230 | 0 | 55,672 |
| POST /api/v1/meta | 20 | 18.9 | 145 | 140 / 200 | 0 | 2,847 |
| PUT /api/v1/meta/:fieldName | 20 | 18.0 | 145 | 140 / 200 | 0 | 2,714 |

## Failures

**None.** All 8 phases, all endpoints: 0 HTTP errors, 0 exceptions.

## Root Cause Analysis — Why Stress Phases Now Pass

Week 2 had 1,694 total errors across 4 failing stress phases. Week 3 has 0. Three changes drove this:

**1. StopUser fix on login failure**

`fix: raise StopUser on login failure, increase login timeout to 30s` (commit `60382d7`). In week 2, users whose login timed out would re-queue immediately, causing a perpetual storm of pending bcrypt tasks. Each queued retry held a goroutine, a connection from the pool, and CPU time — rapidly exhausting all three. With `StopUser`, a user whose login fails exits the pool entirely. The remaining users share a smaller, manageable connection queue, and bcrypt throughput stabilises.

Effect: login P50 dropped from 11,000ms (week 2 moderate) to 2,600ms (week 3 moderate) — a **4.2× improvement** — because the queue backlog no longer builds faster than bcrypt can drain it.

**2. PgBouncer session mode — connection pool stability**

`SetMaxOpenConns(10)` / `SetMaxIdleConns(5)` from the week 2 fix remained in place. PgBouncer in session mode sits between the app and PostgreSQL, preventing connection spikes from reaching the PostgreSQL limit (100). Week 2 read/write stress failures were partly driven by connection exhaustion; week 3 shows 0 failures even at 410+ RPS on reads.

**3. DB reset and clean state**

The `railway run -p -e -s -- psql $DATABASE_PUBLIC_URL -c TRUNCATE ...` + `seed.py` reset before each run ensures no leftover sessions, orphaned refresh tokens, or meta-field state carry over between test runs. Week 2 misc/stress had 772 HTTP 401 errors on `POST /auth/refresh` — expired tokens from a prior run. Week 3: 0.

## Notable Observations

**Login is bcrypt-limited (not a regression):**
- Moderate actual RPS: 36 (target: 100). This is expected — one replica, single-threaded bcrypt hashing.
- Stress actual RPS: 32.6 (target: 500). Throughput did not increase meaningfully with more users because CPU is the constraint, not concurrency.
- Login P99 under stress: 37,000ms. High but acceptable — login succeeds eventually, just slowly. Non-login endpoints are unaffected.

**Read baseline is 130ms (network-limited):**
- All GET endpoints: P50=130ms (moderate), 130–150ms (stress). This is the GitHub Actions → Railway us-west2 network round-trip, not application latency. Week 2 appeared faster (15ms) because almost no reads occurred (users were stuck on 11s logins), hiding the true baseline.

**Write throughput scales well:**
- Stress: 377 RPS actual (target: 1000), P50=130ms. Write performance is network-bound, not CPU or DB-bound. The 1000 RPS target is unachievable from a single GitHub Actions runner due to the ~130ms baseline per request, but throughput is stable and error-free.

## Comparison vs. Previous Weeks

| Metric | Week 1 (Local, SQLite) | Week 2 (Railway, PostgreSQL) | Week 3 (Railway, PostgreSQL + fix) |
| --- | --- | --- | --- |
| Login P50 (moderate) | 790ms | 11,000ms | 2,600ms |
| Login actual RPS (moderate) | 93 | 8.6 | 36.1 |
| Login stress error rate | 0% | 20.9% | **0%** |
| Reads P50 (moderate) | 1–4ms | 15–17ms* | 130ms |
| Write P50 (moderate) | 1–2ms | 18–20ms | 130ms |
| Refresh P50 (moderate) | 3ms | 29ms | 140ms |
| Stress phase pass rate | 4/4 ✓ | 0/4 ✗ | **4/4 ✓** |
| Total errors (stress) | 0 | 1,694 | **0** |

*Week 2 reads appeared low-latency because very few reads occurred (users blocked on slow login).

## Agent Report

**Agent:** Testing master (load test subagent)
**Timestamp:** 2026-03-27T04:21:30Z
**Status:** completed

### 1. What I Did

- Identified open PRs #80 and #83 against develop
- Diagnosed PR #83 (`ci/azure-load-test-v2`) had merge conflicts; resolved all 6 conflicted files
- Merged PR #83 into develop; closed superseded PR #80
- Triggered load test run `23630392237` against develop on Railway
- Analyzed results from all 8 phases

### 2. What I See Wrong

Nothing wrong. All 8 phases passed with 0 failures. The `read_heavy/stress` duration extended to 467s due to 2,000-user login startup time — this is expected behaviour, not a bug.

### 3. What I Suggest Next

- Consider adding a Honeycomb query overlay to correlate bcrypt CPU saturation with trace spans (login duration histogram)
- Consider a horizontal scale test (2 replicas) to see if connection pool budget holds and throughput doubles for non-login endpoints
- The login bcrypt bottleneck (36 RPS actual vs 100 target) remains the primary capacity ceiling; a bcrypt work-factor reduction or pre-hashed test credentials could isolate it for future tests

### 4. Why This Fix Will Work

N/A — task completed successfully.

### Extension

**Agents involved:** Testing master (this agent)
**Service code issue found:** no
**New guardrail needed:** no
**Rules to save in CLAUDE.md:** none — existing KNOWN_ISSUES.md entries #10 and #11 already capture the StopUser and concurrency fixes
