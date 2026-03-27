# CI/CD with Azure — Load Test Environment

## Overview

Azure is used exclusively for **ephemeral load test runs**. Every run provisions fresh infrastructure, runs the tests, then destroys everything. There is no persistent Azure environment.

```
GitHub Actions (workflow_dispatch)
  └── Terraform: create RG + ACR + PostgreSQL
  └── az acr build: build image in ACR
  └── az container create: deploy ACI in the same RG
  └── run_all.sh / run_group.sh
  └── Terraform + az group delete: destroy all resources
```

## Why Container Apps (not ACI or App Service)

The goal is a fair latency and cost comparison with Railway. Railway scales to zero when idle and bills per use — so Azure must use the same model.

| Service | Scale to zero | Billing | Match for Railway |
|---|---|---|---|
| App Service | No (always-on) | Per hour | No |
| ACI | No (runs until deleted) | Per second running | Partial |
| **Container Apps (Consumption)** | **Yes** | **Per vCPU-second used** | **Yes** |

Container Apps on the Consumption plan is the direct equivalent: managed container runtime, scale to zero, no idle cost.

## Infrastructure (Terraform)

Terraform manages the durable infrastructure. The Container App is created separately by the CI runner after the image is pushed (avoids chicken-and-egg with ACR).

| Resource | Type | SKU / Tier | Notes |
|---|---|---|---|
| Resource Group | `azurerm_resource_group` | — | Named `gyd-lt-<run_id>` — deleting it cascades to everything |
| Container Registry | `azurerm_container_registry` | Basic | `admin_enabled = true` — credentials used for Container App image pull |
| PostgreSQL Flexible Server | `azurerm_postgresql_flexible_server` | `B_Standard_B2ms` | PostgreSQL 16, public access, firewall allows all IPs |
| PostgreSQL DB | `azurerm_postgresql_flexible_server_database` | — | Database name: `gatheryourdeals` |
| Container Apps Environment | `azurerm_container_app_environment` | Consumption | Hosts the Container App; destroyed with the RG |

### App Container (Container Apps — CLI-managed)

Created via `az containerapp create` after the image push, in the same resource group so it is destroyed with the RG.

| Parameter | Value |
|---|---|
| CPU | 1.0 vCPU |
| Memory | 2 Gi |
| Ingress | External HTTPS (TLS terminated by platform, proxied to port 8080) |
| Min replicas | 0 (scale to zero) |
| Max replicas | 10 |
| FQDN | `gyd-lt-<run_id>.<hash>.<region>.azurecontainerapps.io` |

## Required GitHub Secrets

| Secret | Description |
|---|---|
| `AZURE_CLIENT_ID` | Service principal client ID (used for OIDC) |
| `AZURE_TENANT_ID` | Azure AD tenant ID |
| `AZURE_SUBSCRIPTION_ID` | Azure subscription ID |
| `AZURE_PG_ADMIN_PASSWORD` | Password for PostgreSQL admin user (`gydadmin`) — min 8 chars, must include uppercase, lowercase, digit |
| `GYD_JWT_SECRET` | JWT signing secret for the app (32+ chars) |
| `GYD_ADMIN_USERNAME` | App admin account username |
| `GYD_ADMIN_PASSWORD` | App admin account password |
| `OTEL_EXPORTER_OTLP_HEADERS` | Optional — Honeycomb ingest key (`x-honeycomb-team=<key>`) |

## One-Time Azure Setup

### 1. Create a Service Principal with Federated Identity (OIDC)

```bash
# Create the app registration
az ad app create --display-name "gyd-github-actions"
APP_ID=$(az ad app list --display-name "gyd-github-actions" --query "[0].appId" -o tsv)

# Create the service principal
az ad sp create --id "$APP_ID"
SP_OBJECT_ID=$(az ad sp show --id "$APP_ID" --query "id" -o tsv)

# Assign Contributor role at subscription scope
SUBSCRIPTION_ID=$(az account show --query id -o tsv)
az role assignment create \
  --assignee "$SP_OBJECT_ID" \
  --role Contributor \
  --scope "/subscriptions/$SUBSCRIPTION_ID"

# Add federated credential for GitHub Actions
az ad app federated-credential create \
  --id "$APP_ID" \
  --parameters '{
    "name": "github-actions-load-test",
    "issuer": "https://token.actions.githubusercontent.com",
    "subject": "repo:yuewang199511/GatherYourDeals-data:ref:refs/heads/develop",
    "audiences": ["api://AzureADTokenAudience"]
  }'

# Also allow any branch (for feature branches triggering the workflow)
az ad app federated-credential create \
  --id "$APP_ID" \
  --parameters '{
    "name": "github-actions-any-branch",
    "issuer": "https://token.actions.githubusercontent.com",
    "subject": "repo:yuewang199511/GatherYourDeals-data:environment:load-test",
    "audiences": ["api://AzureADTokenAudience"]
  }'
```

> **Note:** `workflow_dispatch` triggers use the subject `repo:<owner>/<repo>:ref:refs/heads/<branch>` where the branch is the one selected when dispatching. The simplest approach is to add a federated credential per branch, or use a GitHub environment (recommended).

### Simpler: Use a GitHub Environment for OIDC

1. In GitHub repo → Settings → Environments → New environment → name it `load-test`
2. Add the federated credential with subject `repo:yuewang199511/GatherYourDeals-data:environment:load-test`
3. Add the Azure secrets (`AZURE_CLIENT_ID`, `AZURE_TENANT_ID`, `AZURE_SUBSCRIPTION_ID`, `AZURE_PG_ADMIN_PASSWORD`) to that environment
4. Reference the environment in the workflow job: `environment: load-test`

### 2. Note Down the Values for GitHub Secrets

```bash
echo "AZURE_CLIENT_ID: $APP_ID"
echo "AZURE_TENANT_ID: $(az account show --query tenantId -o tsv)"
echo "AZURE_SUBSCRIPTION_ID: $SUBSCRIPTION_ID"
```

## Teardown

The teardown step runs with `if: always()` — it executes even when tests fail or the job is cancelled.

Two-phase teardown to handle partial failures:
1. `terraform destroy -auto-approve` — removes Terraform-tracked resources cleanly
2. `az group delete --name gyd-lt-<run_id> --yes --no-wait` — backstop that cascades to ACI and any orphans

## Cost Estimate

| Resource | Duration | Approx Cost |
|---|---|---|
| PostgreSQL B_Standard_B2ms | ~45 min | ~$0.05 |
| Container Apps (1 vCPU / 2 Gi, active ~30 min) | ~30 min | ~$0.02 |
| ACR Basic | ~45 min | ~$0.01 |
| **Total per run** | | **~$0.08** |

Container Apps idle time (before/after tests) costs nothing — scale to zero.

## Provider-Specific Docs

- Known issues → `docs/CICD/azure/KNOWN_ISSUES.md`
