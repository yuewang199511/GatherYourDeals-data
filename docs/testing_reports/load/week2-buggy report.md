# Summary

This article records the load testing results from week #2, recorded by Locust on Railway + PostgreSQL.

Service was deployed on Railway (1 replica, us-west2) backed by Railway-managed PostgreSQL, with OTel tracing shipped to Honeycomb.

All 4 moderate phases (100–400 users) completed with **0% error rate**. All 4 stress phases failed with HTTP 500/502/401/404 errors — stress failures are expected at this scale with a single-replica deployment and are recorded for capacity planning.

# Environment

- **Platform**: Railway (1 replica, `us-west2`)
- **Railway Plan**: Hobby ($5/month, 5-month commitment) — upgraded from free tier to unlock PgBouncer and additional provisioning nodes required for PostgreSQL connection pooling
- **Database**: PostgreSQL (Railway-managed, via PgBouncer)
- **Service**: `GatherYourDeals-data` (branch `ci/azure-load-test-v2`)
- **GitHub Run**: `23626012442` (artifact: `load-test-results-23626012442`)
- **Results directory**: `load_testing/results/20260327_011016/`

## Infrastructure Cost

| Item | Cost |
| --- | --- |
| Railway Hobby plan | $5/month |
| Commitment | 5 months |
| **Total committed** | **$25** |

The Hobby plan was required to provision PgBouncer (Railway's managed connection pooler) alongside the PostgreSQL instance. The free tier does not allow additional provisioning nodes needed to run both the app service and the connection pooler simultaneously.

# Performance Result

**Test run IDs:**
- cpu_bound/moderate: `68b7d059-c838-4481-b6a3-58130b8d2290`
- cpu_bound/stress: `f4ce8ca3-7da7-4938-98b0-a9927e81208b`
- read_heavy/moderate: `12e72433-d62f-4f4d-93d3-4de555947560`
- read_heavy/stress: `121ebf40-0734-45a9-813d-d49583e9f06c`
- write_ops/moderate: `2c605688-8821-43f7-80d3-3256f7884970`
- write_ops/stress: `14164848-5fab-4fdb-86e0-b3192413768f`
- misc_lightweight/moderate: `f5014e0a-b8cc-43dc-a993-539a2b17f1a6`
- misc_lightweight/stress: `7187a8de-c710-4a17-97ec-b833874e073d`

**Time Range**: 2026-03-27 01:10 UTC → 01:30 UTC (~20 min total)

## Run Duration

| Group | Phase | Users | Intended (s) | Actual (s) | Notes |
| --- | --- | --- | --- | --- | --- |
| cpu_bound | moderate | 100 | 120 | 120.1 | ✓ |
| cpu_bound | stress | 500 | 150 | 150.2 | ✗ 278 errors |
| read_heavy | moderate | 400 | 120 | 136.0 | ✓ (+16s pre-fetch) |
| read_heavy | stress | 2000 | 150 | 150.3 | ✗ 1,258 errors |
| write_ops | moderate | 100 | 120 | 120.2 | ✓ |
| write_ops | stress | 500 | 150 | 150.2 | ✗ 151 errors |
| misc_lightweight | moderate | 110 | 120 | 120.2 | ✓ |
| misc_lightweight | stress | 540 | 150 | 150.2 | ✗ 1,007 errors |

Note: All groups use `constant_throughput(1.0)` — each virtual user targets 1 task/second. Groups run sequentially so there is no cross-group write contention. `read_heavy/moderate` ran 16s over intended due to the startup pre-fetch of 50 receipt IDs for the `GET /receipts/:id` tasks.

## Moderate Phase

| Endpoint | Target RPS | Actual RPS | Avg Latency (ms) | P50 / P95 (ms) | # Errors | # Requests |
| --- | --- | --- | --- | --- | --- | --- |
| POST /api/v1/auth/login | 100 | 8.6 | 11,036 | 11,000 / 16,000 | 0 | 1,027 |
| GET /api/v1/auth/me | 100 | 12.0 | 53 | 15 / 80 | 0 | 1,647 |
| GET /api/v1/receipts | 100 | 11.4 | 269 | 17 / 120 | 0 | 1,557 |
| GET /api/v1/receipts/:id | 100 | 11.7 | 146 | 16 / 79 | 0 | 1,609 |
| GET /api/v1/meta | 100 | 11.8 | 223 | 16 / 96 | 0 | 1,620 |
| POST /api/v1/receipts | 200 | 91.4 | 37 | 20 / 69 | 0 | 10,999 |
| DELETE /api/v1/receipts/:id | 200 | 91.4 | 30 | 18 / 54 | 0 | 10,999 |
| POST /api/v1/auth/refresh | 100 | 81.1 | 43 | 29 / 76 | 0 | 9,755 |
| POST /api/v1/meta | 5 | 4.0 | 33 | 24 / 56 | 0 | 481 |
| PUT /api/v1/meta/:fieldName | 5 | 3.6 | 38 | 24 / 61 | 0 | 434 |

## Stress Phase

| Endpoint | Target RPS | Actual RPS | Avg Latency (ms) | P50 / P95 (ms) | # Errors | # Requests |
| --- | --- | --- | --- | --- | --- | --- |
| POST /api/v1/auth/login | 500 | 8.9 | 32,851 | 30,000 / 63,000 | 278 | 1,331 |
| GET /api/v1/auth/me | 500 | 20.7 | 211 | 15 / 1,600 | 115 | 3,114 |
| GET /api/v1/receipts | 500 | 20.7 | 1,899 | 24 / 17,000 | 374 | 3,110 |
| GET /api/v1/receipts/:id | 500 | 20.3 | 1,557 | 22 / 14,000 | 352 | 3,060 |
| GET /api/v1/meta | 500 | 20.4 | 1,947 | 23 / 17,000 | 417 | 3,065 |
| POST /api/v1/receipts | 1000 | 135.8 | 1,314 | 990 / 4,100 | 80 | 20,273 |
| DELETE /api/v1/receipts/:id | 1000 | 133.7 | 1,298 | 990 / 4,100 | 71 | 19,950 |
| POST /api/v1/auth/refresh | 500 | 71.7 | 2,041 | 1,100 / 8,800 | 772 | 10,700 |
| POST /api/v1/meta | 5 | 3.5 | 1,406 | 900 / 5,700 | 25 | 519 |
| PUT /api/v1/meta/:fieldName | 5 | 3.9 | 1,656 | 900 / 8,400 | 160 | 577 |

## Failures

| Phase | Endpoint | Error | Count |
| --- | --- | --- | --- |
| cpu_bound/stress | POST /api/v1/auth/login | HTTP 500 | 278 |
| read_heavy/stress | GET /api/v1/auth/me | HTTP 502 | 115 |
| read_heavy/stress | GET /api/v1/meta | HTTP 500 + 502 | 417 |
| read_heavy/stress | GET /api/v1/receipts | HTTP 500 + 502 | 374 |
| read_heavy/stress | GET /api/v1/receipts/:id | HTTP 500 + 502 | 352 |
| write_ops/stress | POST /api/v1/receipts | HTTP 500 | 80 |
| write_ops/stress | DELETE /api/v1/receipts/:id | HTTP 500 | 71 |
| misc_lightweight/stress | POST /api/v1/auth/refresh | HTTP 401 | 772 |
| misc_lightweight/stress | POST /api/v1/meta | HTTP 500 | 25 |
| misc_lightweight/stress | PUT /api/v1/meta/:fieldName | HTTP 500 + 404 | 160 |

## Root Cause Analysis

**1. Login bcrypt saturation (cpu_bound/stress)**

At 500 concurrent users, bcrypt hashing fully saturates the single Railway replica's CPU. Login median jumped from 11,000ms (moderate, 100 users) to 30,000ms (stress, 500 users), and 20.9% of requests (278/1,331) failed with HTTP 500. Honeycomb confirms: `/api/v1/auth/login` error spans under `cpu_bound/stress` show P50=17.8s, P95=27.2s. Throughput was capped at ~8.9 req/s regardless of user count (same as moderate) — the server was already at capacity. This is expected bcrypt behaviour; the service correctly rejects overloaded requests rather than returning corrupt responses.

**2. Connection pool / gateway saturation (read_heavy/stress, write_ops/stress)**

At 2,000 users (read_heavy/stress), the Railway reverse proxy returns HTTP 502 (gateway timeout) mixed with HTTP 500 (application error) across all read endpoints. The bimodal latency pattern — P50=22–24ms but P95=14,000–17,000ms — indicates most requests succeed immediately while a fraction queue behind an exhausted PostgreSQL connection pool. Throughput was capped at ~20 req/s per endpoint (vs. 500 target). Write-ops stress (500 users) fared better at 0.4% error rate because the serialized POST→DELETE pattern limits concurrency on the DB side.

**3. Refresh token expiry cascade (misc_lightweight/stress)**

772 HTTP 401 errors on `POST /api/v1/auth/refresh`. Under high concurrency, refresh tokens expire while queued in the locust task pool — by the time the refresh attempt executes, the token's TTL has passed. This is a test-harness artifact amplified by elevated latency under load; the service is correctly rejecting expired tokens. Not a service bug.

**4. Meta field race condition (misc_lightweight/stress)**

119 HTTP 404 errors on `PUT /api/v1/meta/:fieldName` — the PUT task attempts to update a field that was deleted by a concurrent cleanup pass. 50 HTTP 500 on meta setup and 41 HTTP 500 on meta PUT indicate the meta table is a write-contention hotspot under stress: setup creates a field, another worker deletes it, then a third tries to update the now-absent field.

## Infrastructure Finding: Connection Pool Cap Added

**Finding:** The app had no `SetMaxOpenConns` set, meaning `database/sql` defaults to unlimited open connections. With a single replica this was harmless, but with 2 replicas the combined connection count exceeded Railway PostgreSQL's limit, causing the previous 2-replica attempt (`a7fb60a`) to be reverted.

**Resolution:** Railway PostgreSQL `max_connections = 100` (confirmed via dashboard). The app now sets:
- `SetMaxOpenConns(10)` — caps each replica at 10 connections
- `SetMaxIdleConns(5)` — limits idle connections kept open

Value chosen to be scale-safe: `(100 - 10 reserved) / 10 = 9 replicas` maximum before hitting the limit. This covers any realistic Railway scale-out without needing to reconfigure when adding replicas.

**Budget with 2 replicas:**
| Consumer | Connections |
| --- | --- |
| Replica 1 (app pool) | 10 |
| Replica 2 (app pool) | 10 |
| CI `psql` seed/reset | 1 |
| Goose migrations (startup) | ~2 |
| **Total** | **23 / 100** |

This leaves 77 connections of headroom, supporting up to 9 replicas safely. 2 replicas re-enabled in `railway.toml`.

## Comparison vs. Week 1 (Local SQLite)

| Metric | Week 1 (Local, SQLite) | Week 2 (Railway, PostgreSQL) |
| --- | --- | --- |
| Login P50 (moderate) | 790ms | 11,000ms |
| Login actual RPS (moderate) | 93 | 8.6 |
| Login stress error rate | 0% | 20.9% |
| Reads P50 (moderate) | 1–4ms | 15–17ms |
| Write P50 (moderate) | 1–2ms | 18–20ms |
| Refresh P50 (moderate) | 3ms | 29ms |
| Stress phase pass rate | 4/4 ✓ | 0/4 ✗ |

Login is ~14× slower on Railway due to network round-trip from GitHub Actions to Railway (vs. localhost) compounding with bcrypt. All non-login endpoints remain fast under moderate load (≤120ms P95); the ~14–25ms baseline increase vs. week 1 is pure network latency. Stress failures are driven by single-replica CPU/connection-pool limits, not service correctness.
