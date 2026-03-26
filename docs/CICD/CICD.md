# CI/CD Overview

## Branch Strategy

```
feature/* ──PR──► develop ──PR──► main
```

## Workflow Triggers

| File | Trigger | Purpose |
|---|---|---|
| `test.yml` | PR/push → develop, main | Go unit tests + race detector |
| `build.yml` | PR/push → develop, main | Docker build check |
| `code-quality.yml` | PR/push → develop, main | golangci-lint |
| `security.yml` | PR/push → develop, main | govulncheck |
| `integration-tests.yml` | PR/push → develop | Deploy to Railway load-test + run integration tests |
| `load-tests.yml` | Manual (`workflow_dispatch`) | Full Locust load test suite — Railway or Azure |
| `protect-main.yml` | PR → main | Blocks merge if source branch ≠ develop |

## Load Test Providers

The `load-tests.yml` workflow supports two providers via `inputs.provider`:

| Provider | Infrastructure | Teardown |
|---|---|---|
| `railway` | Persistent Railway load-test environment | None (environment stays up) |
| `azure` | Ephemeral Terraform-managed resources | Auto-destroyed after each run |

## Provider-Specific Docs

- Railway → `docs/CICD/railway/` (if present)
- Azure → `docs/CICD/azure/azure.md`
