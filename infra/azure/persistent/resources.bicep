param location string

@secure()
param pgAdminPassword string

// ── Networking ────────────────────────────────────────────────────────────────

resource vnet 'Microsoft.Network/virtualNetworks@2023-09-01' = {
  name: 'gyd-vnet'
  location: location
  properties: {
    addressSpace: {
      addressPrefixes: ['10.0.0.0/16']
    }
  }
}

resource subnetPg 'Microsoft.Network/virtualNetworks/subnets@2023-09-01' = {
  parent: vnet
  name: 'subnet-pg'
  properties: {
    addressPrefix: '10.0.1.0/24'
    delegations: [
      {
        name: 'pg-delegation'
        properties: {
          serviceName: 'Microsoft.DBforPostgreSQL/flexibleServers'
        }
      }
    ]
  }
}

resource subnetApps 'Microsoft.Network/virtualNetworks/subnets@2023-09-01' = {
  parent: vnet
  name: 'subnet-apps'
  properties: {
    addressPrefix: '10.0.2.0/24'
    delegations: [
      {
        name: 'apps-delegation'
        properties: {
          serviceName: 'Microsoft.App/environments'
        }
      }
    ]
  }
  dependsOn: [subnetPg]
}

// ── Private DNS ───────────────────────────────────────────────────────────────

resource privateDnsZone 'Microsoft.Network/privateDnsZones@2020-06-01' = {
  name: 'gyd-pg-main.private.postgres.database.azure.com'
  location: 'global'
}

resource dnsVnetLink 'Microsoft.Network/privateDnsZones/virtualNetworkLinks@2020-06-01' = {
  parent: privateDnsZone
  name: 'gyd-vnet-pg-dns-link'
  location: 'global'
  properties: {
    virtualNetwork: {
      id: vnet.id
    }
    registrationEnabled: false
  }
}

// ── PostgreSQL Flexible Server ─────────────────────────────────────────────────
// Always-on Burstable B1ms (~$15/mo). DSN stored in GitHub secret
// AZURE_PG_DSN_PERSISTENT. Admin password stored in Bitwarden.

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
      delegatedSubnetResourceId: subnetPg.id
      privateDnsZoneArmResourceId: privateDnsZone.id
      publicNetworkAccess: 'Disabled'
    }
    highAvailability: {
      mode: 'Disabled'
    }
  }
  dependsOn: [dnsVnetLink]
}

resource database 'Microsoft.DBforPostgreSQL/flexibleServers/databases@2024-08-01' = {
  parent: postgres
  name: 'gatheryourdeals'
}

// ── Container Registry ─────────────────────────────────────────────────────────
// Persistent ACR: images pushed on every CI run; never torn down between runs.
// Basic SKU; adminUserEnabled lets CI authenticate for image push/pull.
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
// Persistent, VNet-integrated. The Container App (gyd-lt) is created or updated
// by CI on each run — the environment itself is never torn down.
// Provision time: ~5-10 min (one-time only).
resource caEnv 'Microsoft.App/managedEnvironments@2023-05-01' = {
  name: 'gyd-lt-env'
  location: location
  properties: {
    vnetConfiguration: {
      infrastructureSubnetId: subnetApps.id
    }
  }
}

// ── Outputs ───────────────────────────────────────────────────────────────────
// Note: pgBouncer is no longer a persistent CA resource.
// It runs as a sidecar container inside gyd-lt (deployed by CI each run).
// Sidecar = localhost:6432; eliminates Azure CA internal TCP service discovery issues.

output subnetAppsId string = subnetApps.id
output subnetPgId string = subnetPg.id
output postgresqlFqdn string = postgres.properties.fullyQualifiedDomainName
output acrLoginServer string = acr.properties.loginServer
output acrName string = acr.name
output caEnvName string = caEnv.name
output caEnvDefaultDomain string = caEnv.properties.defaultDomain
