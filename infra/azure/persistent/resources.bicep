param location string

@secure()
param pgAdminPassword string

// ── PostgreSQL Flexible Server ─────────────────────────────────────────────────
// Always-on Burstable B1ms (~$15/mo). Public access; protected by firewall rule
// that restricts to Azure-backbone traffic only (blocks public internet scanners).
// Admin DSN stored in GitHub secret AZURE_PG_DSN_PERSISTENT.
// No VNet — services connect via public FQDN; no private DNS needed.
resource postgres 'Microsoft.DBforPostgreSQL/flexibleServers@2024-08-01' = {
  name: 'gyd-pg-main'
  location: location
  sku: {
    name: 'Standard_B1ms'
    tier: 'Burstable'
  }
  properties: {
    administratorLogin: 'gydadmin'
    administratorLoginPassword: pgAdminPassword
    version: '16'
    storage: {
      storageSizeGB: 32
    }
    backup: {
      backupRetentionDays: 7
      geoRedundantBackup: 'Disabled'
    }
    network: {
      publicNetworkAccess: 'Enabled'
    }
    highAvailability: {
      mode: 'Disabled'
    }
  }
}

// Allow all Azure-backbone traffic; blocks public internet scanners/bots.
// 0.0.0.0→0.0.0.0 is the Azure special flag for "allow Azure services only" —
// it is NOT the same as 0.0.0.0→255.255.255.255 (fully open).
resource pgFirewallAllowAzure 'Microsoft.DBforPostgreSQL/flexibleServers/firewallRules@2024-08-01' = {
  parent: postgres
  name: 'allow-azure-services'
  properties: {
    startIpAddress: '0.0.0.0'
    endIpAddress: '0.0.0.0'
  }
}

resource database 'Microsoft.DBforPostgreSQL/flexibleServers/databases@2024-08-01' = {
  parent: postgres
  name: 'gatheryourdeals'
}

// ── Container Registry ─────────────────────────────────────────────────────────
// Persistent ACR: images pushed on every CI run; never torn down between runs.
resource acr 'Microsoft.ContainerRegistry/registries@2023-07-01' = {
  name: 'gydltpersistent'
  location: location
  sku: {
    name: 'Basic'
  }
  properties: {
    adminUserEnabled: true
  }
}

// ── Container Apps Environment ─────────────────────────────────────────────────
// No VNet integration — pgBouncer and app Container Apps use internal ingress DNS
// for service-to-service communication within this environment.
// pgBouncer and gyd-lt Container Apps are deployed by CI, not here.
resource caEnv 'Microsoft.App/managedEnvironments@2023-05-01' = {
  name: 'gyd-lt-env'
  location: location
  properties: {}
}

// ── Outputs ───────────────────────────────────────────────────────────────────
output postgresqlFqdn string = postgres.properties.fullyQualifiedDomainName
output acrLoginServer string = acr.properties.loginServer
output acrName string = acr.name
output caEnvName string = caEnv.name
output caEnvDefaultDomain string = caEnv.properties.defaultDomain
