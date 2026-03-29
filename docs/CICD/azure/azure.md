# CI/CD with Azure — Load Test Environment

## Overview

Azure is used exclusively for **ephemeral load test runs**. PostgreSQL is persistent (always-on) to eliminate the 8-15 min provisioning wait. Everything else is ephemeral and destroyed after each run.

```
GitHub Actions (workflow_dispatch)
  └── Terraform: create ephemeral RG + ACR + Container Apps Environment (VNet-integrated)
  └── az acr build: build image in ACR
  └── az containerapp create: deploy Go service Container App
  └── az containerapp job: reset DB via Container App Job (VNet-private psql)
  └── seed.py: seed test data via HTTP
  └── run_all.sh / run_group.sh: load tests
  └── Terraform destroy + az group delete: destroy ephemeral resources only
       (persistent PostgreSQL in gyd-persistent RG is NOT touched)
```

## Architecture

### Persistent infrastructure (`gyd-persistent` RG — always on)

Provisioned once manually. Not managed by CI. Terraform stack at `infra/azure/persistent/`.

| Resource | Type | Notes |
|---|---|---|
| Resource Group | `gyd-persistent` | Permanent — never destroyed by CI |
| Virtual Network | `gyd-vnet` (10.0.0.0/16) | Hosts all persistent and ephemeral networking |
| Subnet `subnet-pg` | 10.0.1.0/24 | Delegated to PostgreSQL Flexible Server |
| Subnet `subnet-apps` | 10.0.2.0/24 | Delegated to Container Apps Environment |
| PostgreSQL Flexible Server | `gyd-pg-main` B_Standard_B1ms | PostgreSQL 16, VNet-private, always on (~$15/mo) |
| Private DNS Zone | `gyd-pg-main.private.postgres.database.azure.com` | Resolves within gyd-vnet |

### Ephemeral infrastructure (`gyd-lt-<run_id>` RG — per CI run)

Created by Terraform, destroyed at teardown. Named with `run_id` for isolation.

| Resource | Type | SKU / Tier | Notes |
|---|---|---|---|
| Resource Group | `azurerm_resource_group` | — | Named `gyd-lt-<run_id>` — delete cascades to everything |
| Container Registry | `azurerm_container_registry` | Basic | `admin_enabled = true` for image pull credentials |
| Container Apps Environment | `azurerm_container_app_environment` | Consumption | VNet-integrated via `subnet-apps` — can reach private PostgreSQL |

### App Container (CLI-managed, ephemeral)

Created via `az containerapp create` after image push.

| Parameter | Value |
|---|---|
| CPU | 4.0 vCPU (Consumption plan max; Railway allows up to 8 vCPU) |
| Memory | 8.0 Gi |
| Ingress | External HTTPS |
| Min replicas | 0 (scale to zero) |
| Max replicas | 10 |

### DB Reset Job (CLI-managed, ephemeral)

A Container App Job (`postgres:16` image) runs inside the same VNet to execute the TRUNCATE SQL before each test. This replaces direct `psql` from the CI runner, which cannot reach the VNet-private PostgreSQL.

## Why VNet Integration

PostgreSQL Flexible Server with VNet integration cannot enable public access after creation. All connections must originate from within the VNet. The Container Apps Environment is integrated with `subnet-apps`, giving all apps and jobs private access to PostgreSQL via the private DNS hostname.

## Why No PgBouncer

Container Apps only support HTTP/HTTPS ingress — TCP port 5432 cannot be exposed. PgBouncer as a Container App is therefore not viable. The app uses `database/sql`'s built-in connection pool as a per-replica mitigation. A shared PgBouncer is tracked as future work (see KNOWN_ISSUES.md).

## Required GitHub Secrets

| Secret | Description |
|---|---|
| `AZURE_CLIENT_ID` | Service principal client ID |
| `AZURE_CLIENT_SECRET` | Service principal client secret |
| `AZURE_TENANT_ID` | Azure AD tenant ID |
| `AZURE_SUBSCRIPTION_ID` | Azure subscription ID |
| `AZURE_PG_DSN_PERSISTENT` | Full DSN for persistent PostgreSQL (`postgres://gydadmin:<pass>@gyd-pg-main.postgres.database.azure.com:5432/gatheryourdeals?sslmode=require`) |
| `GYD_JWT_SECRET` | JWT signing secret for the app (32+ chars) |
| `GYD_ADMIN_USERNAME` | App admin account username |
| `GYD_ADMIN_PASSWORD` | App admin account password |
| `OTEL_EXPORTER_OTLP_HEADERS` | Optional — Honeycomb ingest key (`x-honeycomb-team=<key>`) |

## One-Time Manual Setup

These steps were performed on 2026-03-28 and do not need to be repeated unless rebuilding from scratch.

### 1. Terraform state backend

```bash
az group create --name gyd-tf-state --location westus
az storage account create --name gydtfstate --resource-group gyd-tf-state --location westus --sku Standard_LRS
az storage container create --name tfstate --account-name gydtfstate
```

### 2. Persistent resource group and VNet

```bash
az group create --name gyd-persistent --location westus
az network vnet create --name gyd-vnet --resource-group gyd-persistent --location westus --address-prefix 10.0.0.0/16
az network vnet subnet create --name subnet-pg --resource-group gyd-persistent --vnet-name gyd-vnet --address-prefix 10.0.1.0/24 --delegations Microsoft.DBforPostgreSQL/flexibleServers
az network vnet subnet create --name subnet-apps --resource-group gyd-persistent --vnet-name gyd-vnet --address-prefix 10.0.2.0/24
```

### 3. Delegate subnet-apps to Container Apps (required before first CI run)

```bash
az network vnet subnet update \
  --name subnet-apps \
  --resource-group gyd-persistent \
  --vnet-name gyd-vnet \
  --delegations Microsoft.App/environments
```

### 4. Persistent PostgreSQL

```bash
az postgres flexible-server create \
  --name gyd-pg-main \
  --resource-group gyd-persistent \
  --location westus \
  --sku-name Standard_B1ms \
  --tier Burstable \
  --version 16 \
  --admin-user gydadmin \
  --admin-password <your-password> \
  --vnet gyd-vnet \
  --subnet subnet-pg \
  --storage-size 32
```

Save admin password in Bitwarden. Add `AZURE_PG_DSN_PERSISTENT` to GitHub Actions secrets.

### 5. Service Principal

```bash
az ad app create --display-name "gyd-github-actions"
APP_ID=$(az ad app list --display-name "gyd-github-actions" --query "[0].appId" -o tsv)
az ad sp create --id "$APP_ID"
SP_OBJECT_ID=$(az ad sp show --id "$APP_ID" --query "id" -o tsv)
SUBSCRIPTION_ID=$(az account show --query id -o tsv)
az role assignment create --assignee "$SP_OBJECT_ID" --role Contributor --scope "/subscriptions/$SUBSCRIPTION_ID"

# Add federated credential for GitHub Actions
az ad app federated-credential create \
  --id "$APP_ID" \
  --parameters '{
    "name": "github-actions-load-test",
    "issuer": "https://token.actions.githubusercontent.com",
    "subject": "repo:yuewang199511/GatherYourDeals-data:environment:load-test",
    "audiences": ["api://AzureADTokenAudience"]
  }'

echo "AZURE_CLIENT_ID: $APP_ID"
echo "AZURE_TENANT_ID: $(az account show --query tenantId -o tsv)"
echo "AZURE_SUBSCRIPTION_ID: $SUBSCRIPTION_ID"
```

## Teardown

The teardown step runs with `if: always()` — executes even when tests fail or are cancelled.

**Only the ephemeral `gyd-lt-<run_id>` RG is destroyed.** The persistent `gyd-persistent` RG is never touched by CI.

Two-phase teardown to handle partial failures:
1. `terraform destroy` — removes Terraform-tracked resources cleanly
2. `az group delete --name gyd-lt-<run_id>` — backstop that cascades to Container Apps and any orphans

## Cost Estimate

### Persistent (always on)

| Resource | Monthly cost |
|---|---|
| PostgreSQL B_Standard_B1ms | ~$15/mo |
| Storage 32 GB | ~$4/mo |
| Storage Account (tf state) | ~$0/mo |
| **Total** | **~$19/mo** |

### Per CI run (ephemeral)

| Resource | Duration | Approx cost |
|---|---|---|
| Container Apps Environment | ~45 min | ~$0.00 (no idle cost) |
| Go service Container App (4 vCPU / 8 Gi, active ~30 min) | ~30 min | ~$0.05 |
| DB reset Container App Job (0.25 vCPU, ~30 s) | ~30 s | ~$0.00 |
| ACR Basic | ~45 min | ~$0.01 |
| **Total per run** | | **~$0.06** |

## Provider-Specific Docs

- Known issues → `docs/CICD/azure/KNOWN_ISSUES.md`
