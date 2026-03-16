# Load Testing Set #1 (Revised)

## Load Test Profiles — Grouped Endpoints

Endpoints are grouped by resource profile to reduce total test time while preserving clean, attributable data. Each group runs its endpoints concurrently. Groups run sequentially.

## Test Approach

For each group, run two phases:

1. **Moderate (baseline):** Fixed RPS for 2 minutes to capture stable performance
2. **Stress (find ceiling):** Ramp up over 1 minute, hold for 1.5 minutes. Find where P99 degrades or errors appear

**Time per group:** ~4.5 min (2 min moderate + 2.5 min stress)  
**Total test time:** ~18 min for all 4 groups per platform

---

## Group 1: CPU-Bound (Solo)

Runs alone — bcrypt password hashing saturates CPU and would skew all other endpoints.

### POST /auth/login

| Phase    | Target RPS | Duration | Load Profile                            |
| -------- | ---------- | -------- | --------------------------------------- |
| Moderate | 100        | 2 min    | Fixed                                   |
| Stress   | 500        | 2.5 min  | Ramp from 100 → 500 over 1 min, hold 1.5 min |

**Watch for:** CPU hitting 100%, all bcrypt operations queuing

---

## Group 2: Read-Heavy (Concurrent)

All read endpoints grouped together. These hit SQLite reads and don't compete for the write lock. Run all four simultaneously.

### GET /receipts (list)

| Phase    | Target RPS | Duration | Load Profile                            |
| -------- | ---------- | -------- | --------------------------------------- |
| Moderate | 100        | 2 min    | Fixed                                   |
| Stress   | 500        | 2.5 min  | Ramp from 100 → 500 over 1 min, hold 1.5 min |

Retrieves page size 50.

### GET /receipts/:id

| Phase    | Target RPS | Duration | Load Profile                            |
| -------- | ---------- | -------- | --------------------------------------- |
| Moderate | 100        | 2 min    | Fixed                                   |
| Stress   | 500        | 2.5 min  | Ramp from 100 → 500 over 1 min, hold 1.5 min |

### GET /auth/me

| Phase    | Target RPS | Duration | Load Profile                            |
| -------- | ---------- | -------- | --------------------------------------- |
| Moderate | 100        | 2 min    | Fixed                                   |
| Stress   | 500        | 2.5 min  | Ramp from 100 → 500 over 1 min, hold 1.5 min |

### GET /meta

| Phase    | Target RPS | Duration | Load Profile                            |
| -------- | ---------- | -------- | --------------------------------------- |
| Moderate | 100        | 2 min    | Fixed                                   |
| Stress   | 500        | 2.5 min  | Ramp from 100 → 500 over 1 min, hold 1.5 min |



---

## Group 3: Write Operations (Create-Then-Delete Cycle)

Each cycle creates a receipt (POST), grabs the ID from the response, then deletes it (DELETE). This is self-sustaining — no pre-seeding or queue exhaustion issues — and tests real write lock contention with mixed create/delete pressure.

Each cycle = 2 write operations, so cycle RPS is set to produce equivalent total write pressure to the original targets (original combined: 25 moderate, 90 stress).

### Workflow: POST /receipts → DELETE /receipts/:id

| Phase    | Cycles/sec | Effective writes/sec | Duration | Load Profile                            |
| -------- | ---------- | -------------------- | -------- | --------------------------------------- |
| Moderate | 100        | 200                  | 2 min    | Fixed                                   |
| Stress   | 500        | 1000                 | 2.5 min  | Ramp from 100 → 500 over 1 min, hold 1.5 min |

**Locust stats:** POST /receipts and DELETE /receipts/:id are tracked as separate entries — you get independent P95, P99, error rate, and RPS for each despite running as a workflow.

**Watch for:** P99 spike from write lock contention. Biggest expected improvement when migrating to Postgres. DELETE latency will include dependency on the preceding POST completing, so factor that in when reading results.

---

## Group 4: Lightweight Misc (Concurrent)

Low-resource endpoints with minimal overlap. Run all four simultaneously.

**Note on POST /auth/logout:** Each successful logout invalidates the token. Use a setup phase to pre-generate a pool of tokens (login N times, push tokens into a shared thread-safe queue). Each logout worker consumes one token from the pool. At 40 RPS stress over 2.5 min, pre-generate ~6,000 tokens to avoid exhaustion.

### POST /auth/refresh

| Phase    | Target RPS | Duration | Load Profile                            |
| -------- | ---------- | -------- | --------------------------------------- |
| Moderate | 100        | 2 min    | Fixed                                   |
| Stress   | 500        | 2.5 min  | Ramp from 100 → 500 over 1 min, hold 1.5 min |

### POST /auth/logout

Kept lower — each logout consumes a pre-generated token from the pool (7,000 tokens pre-filled).

| Phase    | Target RPS | Duration | Load Profile                            |
| -------- | ---------- | -------- | --------------------------------------- |
| Moderate | 10         | 2 min    | Fixed                                   |
| Stress   | 40         | 2.5 min  | Ramp from 10 → 40 over 1 min, hold 1.5 min |

### POST /meta

Kept lower — creates unique field names; high RPS would exhaust the namespace.

| Phase    | Target RPS | Duration | Load Profile                            |
| -------- | ---------- | -------- | --------------------------------------- |
| Moderate | 5          | 2 min    | Fixed                                   |
| Stress   | 20         | 2.5 min  | Ramp from 5 → 20 over 1 min, hold 1.5 min |

### PUT /meta/:fieldName

| Phase    | Target RPS | Duration | Load Profile                            |
| -------- | ---------- | -------- | --------------------------------------- |
| Moderate | 5          | 2 min    | Fixed                                   |
| Stress   | 20         | 2.5 min  | Ramp from 5 → 20 over 1 min, hold 1.5 min |

**Combined moderate RPS:** 120 (refresh 100 + logout 10 + meta ×2 5+5)
**Combined stress RPS:** 580

---

## Time Estimates Per Platform

| Scope              | Groups | Time per group | Total    |
| ------------------ | ------ | -------------- | -------- |
| All groups         | 4      | ~4.5 min       | ~18 min  |

---

## Test Setup

| Parameter          | Value                                           |
| ------------------ | ----------------------------------------------- |
| Database records   | 1,000 purchase receipts pre-seeded              |
| Test user          | Single user account, all records under this user |
| Virtual users/workers | Multiple workers sharing one user account     |
| Test runner        | Locust (headless) via GitHub Actions            |
| Results format     | JSON + CSV per group per phase                  |

---

## Metrics to Capture (Per Endpoint, Per Phase)

### Performance (from Locust)

| Metric                 | Description                              |
| ---------------------- | ---------------------------------------- |
| P95 response time (ms) | 95th percentile latency                  |
| P99 response time (ms) | 99th percentile latency                  |
| Avg response time (ms) | Mean latency                             |
| Actual RPS             | Achieved requests per second vs target   |
| Error rate (%)         | Percentage of failed requests            |
| Total requests         | Total requests sent over test duration   |

### Infrastructure (from platform monitoring)

| Metric                | Source                              |
| --------------------- | ----------------------------------- |
| CPU utilization (%)   | Azure Monitor / Railway dashboard   |
| Memory utilization (%)| Azure Monitor / Railway dashboard   |
| Network ingress (bytes)| Azure Monitor / Railway dashboard  |
| Network egress (bytes)| Azure Monitor / Railway dashboard   |

### Cost (per platform)

| Metric          | Source                                                                  |
| --------------- | ----------------------------------------------------------------------- |
| Compute cost    | Azure Cost Management (hourly) / Railway usage dashboard (per-minute)   |
| Bandwidth cost  | Azure Cost Management / Railway usage dashboard                         |
| Database cost   | Azure Cost Management / Railway usage dashboard                         |
| Total test cost | Sum of above for the test time window                                   |

---

## Test Context (recorded in Locust JSON output)

| Field                      | Description                                             |
| -------------------------- | ------------------------------------------------------- |
| Test start/end timestamps  | Exact time window for correlating with platform metrics  |
| Test duration (seconds)    | Total run time                                          |
| Target RPS                 | From config                                             |
| Group under test           | Which group and endpoints this run targets              |
| Number of virtual users    | Locust worker count                                     |
| Platform name and region   | Railway / Azure / AWS + region                          |
| Database type              | SQLite (Round 1) or PostgreSQL (Round 2)                |

---

## Analysis Tips

| Signal                                  | Meaning                                                                                                  |
| --------------------------------------- | -------------------------------------------------------------------------------------------------------- |
| P99 spike on POST/DELETE receipts       | SQLite write lock contention — multiple writers queuing for the single write lock                         |
| P99 spike on POST /auth/login           | Bcrypt CPU saturation — password hashing consuming all available CPU                                     |
| Large gap between P95 and P99           | Request queuing — server near capacity, some requests waiting in line                                    |
| Error rate climbing with RPS            | Approaching service capacity ceiling — the RPS where errors start is your practical limit                |
| CPU at 100% while RPS plateaus          | Compute-bound — need larger instance or code optimization                                                |
| GET /receipts response time growing     | Payload too large without pagination — 1,000 records serialized per request. **Need pagination first!!!!!** |
| Group 2 reads degrading each other      | SQLite reader contention or CPU saturation from combined read load                                       |
| DELETE latency much higher than POST in Group 3 | Expected — DELETE depends on POST completing first. Compare POST latency in Group 3 vs solo to measure lock contention overhead |