targetScope = 'subscription'

param location string = 'westus3'

@secure()
param pgAdminPassword string

resource rg 'Microsoft.Resources/resourceGroups@2023-07-01' = {
  name: 'gyd-persistent'
  location: location
}

module resources 'resources.bicep' = {
  name: 'gyd-resources'
  scope: rg
  params: {
    location: location
    pgAdminPassword: pgAdminPassword
  }
}

output subnetAppsId string = resources.outputs.subnetAppsId
output subnetPgId string = resources.outputs.subnetPgId
output postgresqlFqdn string = resources.outputs.postgresqlFqdn
output acrLoginServer string = resources.outputs.acrLoginServer
output acrName string = resources.outputs.acrName
output caEnvName string = resources.outputs.caEnvName
