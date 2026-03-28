# Testing Change Log

Design decisions and significant testing strategy changes. Individual run results are in `docs/testing_reports/`.

| Date | PR | Change | Reason |
|---|---|---|---|
| 2026-03-17 | #44 | Added token-pool-driven logout group (Group 5) | Pre-generating tokens avoids polluting login latency during logout stress |
| 2026-03-22 | #51 | Added pytest integration test suite covering all 15 endpoints | Catch HTTP-level regressions fast before running expensive load tests; tests run in CI on every PR |
| 2026-03-23 | #52 | Added seed guardrail script; runs automatically before every load test | Prevent load tests from running against wrong DB state (wrong record count, missing user, schema mismatch) |
| 2026-03-26 | #83 | Adopted Railway + PostgreSQL as the primary remote load-test target | SQLite cannot validate production-scale behaviour; Railway PgBouncer provides realistic connection-pool pressure |
| 2026-03-26 | #83 | Added PgBouncer session-mode connection pooling to load-test environment | Prevents connection exhaustion under stress; `SetMaxOpenConns(10)` / `SetMaxIdleConns(5)` validated as safe headroom — see `docs/testing_reports/load/run2-buggy railway - 20260327_011016.md` |
| 2026-03-26 | manual | Removed Group 5 (logout) from load test suite | Token-pool approach added complexity without proportional signal; logout is covered by integration tests |
| 2026-03-28 | #87 | Renamed load test reports from `weekN` to `runN`; added `-railway` + timestamps to run2–4 | Week numbering implied time-box; run numbering is more accurate. Timestamps enable direct correlation with CI artifacts |
| 2026-03-28 | #91 | Subagents made read-only; master agent owns all writes | Single write path prevents concurrent mutations; subagents return analysis and fix plans only |
