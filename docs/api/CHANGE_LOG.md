# API Change Log

Design decisions and significant API changes. See `docs/api/api.yaml` for the current spec.

| Date | PR | Change | Reason |
|---|---|---|---|
| 2026-02-28 | #16 | Switched auth from OAuth2 to JWT | Simpler stateless token validation; epoch time used in DB for consistency |
| 2026-02-28 | #13 | Added persistent refresh token storage | Foundation for future Redis-backed session sharing across replicas |
| 2026-02-28 | #18 | Added receipt + meta field CRUD endpoints; flattened receipt format | Flattening removes unnecessary nesting; admin control moved into each handler |
| 2026-03-15 | #33 | All endpoints backed by PostgreSQL in addition to SQLite | Both backends must satisfy identical interface; `$N` placeholders for PostgreSQL, `?` for SQLite |
| 2026-03-15 | #34 | Added page-number-based pagination to all list endpoints | Prevent unbounded result sets under load; consistent cursor model across resources |
| 2026-03-16 | #41 | Added OTel span instrumentation to all HTTP handlers | Enables per-endpoint latency tracing in Honeycomb; `test.run_id` forwarded as span attribute for test correlation |
