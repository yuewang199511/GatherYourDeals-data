# Summary

This article records the load testing results from week #1, recorded by Locust.

Service was set up as a local SQLite-backed Gin service running in Docker Compose.

All 8 phases (4 groups × moderate + stress) completed with **0% error rate**.

# Environment

- **Platform**: local (WSL2)
- **Database**: SQLite
- **Service**: Docker Compose (`gatheryourdeals-data-app`)
- **Results directory**: `load_testing/results/20260322_225848/`

# Performance Result

**Test run IDs** (cpu_bound group; others in `*_context.json` files):
- cpu_bound/moderate: `17646cd5-6780-438a-8020-4c49c96542f4`
- cpu_bound/stress: `e1433f04-7f49-404b-84c4-04e4e4da9bdd`

**Time Range**: 2026-03-23 05:58 UTC → 06:18 UTC (~20 min total)

## Run Duration

| Group | Phase | Users | Intended (s) | Actual (s) | Notes |
| --- | --- | --- | --- | --- | --- |
| cpu_bound | moderate | 100 | 120 | 120.1 | ✓ |
| cpu_bound | stress | 500 | 150 | 150.2 | ✓ |
| read_heavy | moderate | 400 | 120 | 120.2 | ✓ |
| read_heavy | stress | 2000 | 150 | 150.9 | ✓ |
| write_ops | moderate | 100 | 120 | 120.1 | ✓ |
| write_ops | stress | 500 | 150 | 150.2 | ✓ |
| misc_lightweight | moderate | 110 | 120 | 120.1 | ✓ |
| misc_lightweight | stress | 540 | 150 | 150.9 | ✓ |

Note: All groups use `constant_throughput(1.0)` — each virtual user targets 1 task/second. Expected RPS ≈ virtual user count per endpoint. When the server is slower than 1s/request, throughput degrades below the user count (e.g. login under stress). Groups run sequentially so there is no cross-group write contention.

## Moderate Phase

| Endpoint | Target RPS | Actual RPS | Avg Latency (ms) | P50 / P95 (ms) | # Errors | # Requests |
| --- | --- | --- | --- | --- | --- | --- |
| POST /api/v1/auth/login | 100 | 93 | 785 | 790 / 1100 | 0 | 11,771 |
| GET /api/v1/auth/me | 100 | 97 | 3 | 1 / 2 | 0 | 12,106 |
| GET /api/v1/receipts | 100 | 97 | 4 | 1 / 4 | 0 | 12,142 |
| GET /api/v1/receipts/:id | 100 | 97 | 3 | 1 / 2 | 0 | 12,057 |
| GET /api/v1/meta | 100 | 97 | 3 | 1 / 2 | 0 | 12,145 |
| POST /api/v1/receipts | 200 | 99 | 3 | 2 / 11 | 0 | 12,314 |
| DELETE /api/v1/receipts/:id | 200 | 99 | 3 | 1 / 9 | 0 | 12,314 |
| POST /api/v1/auth/refresh | 100 | 97 | 3 | 2 / 5 | 0 | 12,257 |
| POST /api/v1/meta | 5 | 5 | 2 | 2 / 3 | 0 | 605 |
| PUT /api/v1/meta/:fieldName | 5 | 5 | 3 | 2 / 8 | 0 | 603 |

## Stress Phase

| Endpoint | Target RPS | Actual RPS | Avg Latency (ms) | P50 / P95 (ms) | # Errors | # Requests |
| --- | --- | --- | --- | --- | --- | --- |
| POST /api/v1/auth/login | 500 | 87 | 4027 | 4600 / 5900 | 0 | 13,435 |
| GET /api/v1/auth/me | 500 | 307 | 74 | 4 / 480 | 0 | 37,951 |
| GET /api/v1/receipts | 500 | 308 | 494 | 110 / 2200 | 0 | 38,041 |
| GET /api/v1/receipts/:id | 500 | 305 | 322 | 48 / 1600 | 0 | 37,759 |
| GET /api/v1/meta | 500 | 303 | 419 | 59 / 1900 | 0 | 37,468 |
| POST /api/v1/receipts | 1000 | 372 | 3 | 2 / 10 | 0 | 58,187 |
| DELETE /api/v1/receipts/:id | 1000 | 372 | 3 | 2 / 7 | 0 | 58,187 |
| POST /api/v1/auth/refresh | 500 | 364 | 4 | 2 / 18 | 0 | 57,778 |
| POST /api/v1/meta | 20 | 18 | 3 | 2 / 12 | 0 | 2,787 |
| PUT /api/v1/meta/:fieldName | 20 | 18 | 3 | 2 / 13 | 0 | 2,887 |



