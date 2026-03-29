output "resource_group_name" {
  description = "Name of the ephemeral resource group"
  value       = azurerm_resource_group.main.name
}

output "location" {
  description = "Azure region where resources are deployed"
  value       = azurerm_resource_group.main.location
}

output "acr_login_server" {
  description = "ACR login server FQDN (e.g. gydlt123.azurecr.io)"
  value       = azurerm_container_registry.main.login_server
}

output "acr_name" {
  description = "ACR registry name (short, without domain)"
  value       = azurerm_container_registry.main.name
}

output "container_app_environment_name" {
  description = "Container Apps Environment name — passed to az containerapp create"
  value       = azurerm_container_app_environment.main.name
}
