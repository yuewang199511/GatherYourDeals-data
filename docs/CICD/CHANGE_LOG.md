# CI/CD Change Log

Design decisions and significant infrastructure changes. Routine report additions are not logged here — see `docs/testing_reports/` for run history.

| Date | PR | Change | Reason |
|---|---|---|---|
| 2026-03-15 | #33 | Added PostgreSQL support to CI pipeline | Production target requires PostgreSQL; SQLite is local-dev only |
| 2026-03-16 | #41 | Added OTel tracing + Honeycomb export to service | Need distributed tracing for load test diagnostics and production observability |
| 2026-03-22 | #51 | Added integration test job to CI | Catch HTTP-level regressions before load testing; faster feedback than load tests |
| 2026-03-23 | #52 | Added seed guardrail check before every test run | Prevent tests from running against wrong DB state; avoids false passes on stale data |
| 2026-03-26 | #72 | Added Azure Container Apps ephemeral load-test environment via Terraform | Validate service on a second cloud provider; ACI swapped for Container Apps to match Railway's scale-to-zero model |
| 2026-03-26 | #83 | Switched Railway load-test DB to PostgreSQL via PgBouncer (session mode) | SQLite cannot scale horizontally; PgBouncer prevents connection exhaustion under stress load |
| 2026-03-26 | #83 | Reverted Railway load-test to 1 replica; serialized runs per provider | 2-replica run caused cross-provider concurrency conflicts and resource contention during seeding |
| 2026-03-26 | #83 | Added `replicas` workflow input; upsert numReplicas in railway.toml | Allows controlled replica scaling experiments without hardcoding values in the workflow |
| 2026-03-28 | #89 | Added pre-merge branch sync check rule to CLAUDE.md | Prevent merging stale branches; enforces `git fetch + git log HEAD..origin/<base>` before every merge |
| 2026-03-28 | #91 | Subagents made read-only; only master agent writes/commits/pushes | Prevent concurrent file mutations from multiple agents; single write path simplifies audit trail |
| 2026-03-28 | #93 | Added multi-agent fix coordination rules | Define conflict detection, one-fix-at-a-time sequencing, and priority order for simultaneous subagent reports |
| 2026-03-28 | — | Switched Azure PostgreSQL to persistent VNet-private instance | Eliminate 8-15 min provisioning wait per CI run; ~$19/mo persistent cost vs repeated wait |
| 2026-03-28 | — | Added Container Apps VNet integration | Container Apps must reach private PostgreSQL via gyd-vnet; subnet-apps delegated to Microsoft.App/environments |
| 2026-03-28 | — | Replaced direct psql DB reset with Container App Job | CI runner cannot reach VNet-private PostgreSQL; job runs inside VNet instead |
| 2026-03-28 | — | Updated Container App CPU/memory to 4.0 vCPU / 8.0 Gi | Approximate Railway's 8 vCPU / 8 GB resource limit (Consumption plan max is 4 vCPU) |
| 2026-03-29 | — | Replaced az acr build with docker build + docker push | ACR Tasks blocked by subscription policy (TasksOperationsNotAllowed); local runner build avoids it |
| 2026-03-29 | — | Made otel-headers Container App secret conditional | Empty OTEL_EXPORTER_OTLP_HEADERS secret caused ContainerAppSecretInvalid; skip secret when unset |
| 2026-03-29 | — | Added pre-provision subnet cleanup + synchronous teardown | ManagedEnvironmentSubnetInUse blocked new runs; --no-wait teardown left subnet occupied; now waits for full deletion |
