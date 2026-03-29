# CI/CD with Azure — Load Test Environment

## Overview

Azure is used exclusively for **load test runs**. All resources are persistent and scale to zero when idle — no provisioning wait per run.

```
GitHub Actions (workflow_dispatch)
  └── az postgres flexible-server execute: goose migrations (as gydadmin)
  └── docker buildx build + push: build image → ACR
  └── az containerapp create/update: deploy pgBouncer Container App (internal, min 1 replica)
  └── az containerapp create/update: deploy gyd-lt Container App (external HTTPS)
  └── psql: TRUNCATE + seed test data
  └── run_all.sh / run_group.sh: load tests
```

## Architecture

### Persistent infrastructure (`gyd-persistent` RG — always on)

Provisioned once via `infra/azure/persistent/resources.bicep`. Never destroyed by CI.

| Resource | Name | Notes |
|---|---|---|
| Resource Group | `gyd-persistent` | Permanent |
| PostgreSQL Flexible Server | `gyd-pg-main` (Burstable B1ms) | PostgreSQL 16, public access, firewall: Azure services only |
| Container Registry | `gydltpersistent` | Basic SKU; images pushed each CI run |
| Container Apps Environment | `gyd-lt-env` | No VNet; internal ingress DNS for service-to-service comms |

### Container Apps (deployed by CI, persistent between runs)

| App | Ingress | Purpose |
|---|---|---|
| `gyd-pgbouncer` | Internal, min 1 replica | Connection pooler; app connects here, pgBouncer → Postgres |
| `gyd-lt` | External HTTPS, min 0 replicas | Go service under test |

### Connection flow

```
gyd-lt (app)
  └── postgres://gydadmin@gyd-pgbouncer.internal.<env-domain>:5432  (sslmode=disable, internal)
        └── gyd-pgbouncer (edoburu/pgbouncer, session mode)
              └── postgres://gydadmin@gyd-pg-main.postgres.database.azure.com:5432  (sslmode=require)
```

## Security model

| Layer | Mechanism |
|---|---|
| Network | Firewall: `0.0.0.0→0.0.0.0` (Azure services only — blocks public internet) |
| Transport | SSL required on Postgres; internal CA-env DNS for pgBouncer |
| Identity | gydadmin (admin, migrations + pgBouncer backend); gydapp (DML-only, future runtime split) |
| pgBouncer | Internal ingress only — not reachable from public internet |

## DB user setup (one-time after provisioning)

```bash
az postgres flexible-server execute \
  --name gyd-pg-main \
  --resource-group gyd-persistent \
  --admin-user gydadmin \
  --admin-password '<password>' \
  --database-name gatheryourdeals \
  --file-path infra/azure/setup-db-users.sql
```

See `infra/azure/setup-db-users.sql` for the `gydapp` restricted user definition.

## Required GitHub Secrets

| Secret | Description |
|---|---|
| `AZURE_CLIENT_ID` | Service principal client ID |
| `AZURE_CLIENT_SECRET` | Service principal client secret |
| `AZURE_TENANT_ID` | Azure AD tenant ID |
| `AZURE_SUBSCRIPTION_ID` | Azure subscription ID |
| `AZURE_PG_DSN_PERSISTENT` | Admin DSN: `postgres://gydadmin:<pass>@gyd-pg-main.postgres.database.azure.com:5432/gatheryourdeals?sslmode=require` |
| `GYD_JWT_SECRET` | JWT signing secret (32+ chars) |
| `GYD_ADMIN_USERNAME` | App admin account username |
| `GYD_ADMIN_PASSWORD` | App admin account password |
| `OTEL_EXPORTER_OTLP_HEADERS` | Optional — Honeycomb ingest key (`x-honeycomb-team=<key>`) |

## One-Time Manual Setup

### 1. Provision persistent infrastructure

```bash
az group create --name gyd-persistent --location westus3
az deployment group create \
  --resource-group gyd-persistent \
  --template-file infra/azure/persistent/resources.bicep \
  --parameters location=westus3 pgAdminPassword='<password>'
```

Save admin password in Bitwarden. Add `AZURE_PG_DSN_PERSISTENT` to GitHub Actions secrets.

### 2. Set up DB users

```bash
az postgres flexible-server execute \
  --name gyd-pg-main \
  --resource-group gyd-persistent \
  --admin-user gydadmin \
  --admin-password '<password>' \
  --database-name gatheryourdeals \
  --file-path infra/azure/setup-db-users.sql
```

### 3. Service Principal

```bash
az ad app create --display-name "gyd-github-actions"
APP_ID=$(az ad app list --display-name "gyd-github-actions" --query "[0].appId" -o tsv)
az ad sp create --id "$APP_ID"
SP_OBJECT_ID=$(az ad sp show --id "$APP_ID" --query "id" -o tsv)
SUBSCRIPTION_ID=$(az account show --query id -o tsv)
az role assignment create --assignee "$SP_OBJECT_ID" --role Contributor --scope "/subscriptions/$SUBSCRIPTION_ID"
```

## Cost Estimate

### Persistent (always on)

| Resource | Monthly cost |
|---|---|
| PostgreSQL Burstable B1ms | ~$15/mo |
| Storage 32 GB | ~$4/mo |
| **Total** | **~$19/mo** |

### Per CI run

| Resource | Approx cost |
|---|---|
| Container Apps (active ~30 min) | ~$0.05 |
| ACR image push | ~$0.01 |
| **Total per run** | **~$0.06** |

## Provider-Specific Docs

- Known issues → `docs/CICD/azure/KNOWN_ISSUES.md`
