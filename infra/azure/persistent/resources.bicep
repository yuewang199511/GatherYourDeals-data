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

// ── Outputs ───────────────────────────────────────────────────────────────────

output subnetAppsId string = subnetApps.id
output subnetPgId string = subnetPg.id
output postgresqlFqdn string = postgres.properties.fullyQualifiedDomainName
