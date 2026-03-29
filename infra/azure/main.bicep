param runId string
param location string = 'eastus'
param appsSubnetId string

var nameSuffix = 'gyd-lt-${runId}'
var acrName = 'gydlt${runId}'

// ── Container Registry ─────────────────────────────────────────────────────────
// Basic SKU is cheapest; adminUserEnabled lets CI authenticate for image push/pull.
resource acr 'Microsoft.ContainerRegistry/registries@2023-07-01' = {
  name: acrName
  location: location
  sku: {
    name: 'Basic'
  }
  properties: {
    adminUserEnabled: true
  }
}

// ── Container Apps Environment ─────────────────────────────────────────────────
// Consumption plan: scale to zero, billed per vCPU-second.
// VNet integration via infrastructureSubnetId lets Container Apps reach the
// persistent private PostgreSQL at gyd-pg-main.private.postgres.database.azure.com.
resource caEnv 'Microsoft.App/managedEnvironments@2023-05-01' = {
  name: nameSuffix
  location: location
  properties: {
    vnetConfiguration: {
      infrastructureSubnetId: appsSubnetId
    }
  }
}

// ── Outputs ───────────────────────────────────────────────────────────────────

output acrLoginServer string = acr.properties.loginServer
output acrName string = acr.name
output containerAppEnvironmentName string = caEnv.name
