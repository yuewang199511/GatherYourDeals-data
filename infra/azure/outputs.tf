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

output "postgresql_fqdn" {
  description = "PostgreSQL Flexible Server FQDN"
  value       = azurerm_postgresql_flexible_server.main.fqdn
}

output "postgresql_admin_login" {
  description = "PostgreSQL admin username"
  value       = azurerm_postgresql_flexible_server.main.administrator_login
}

output "postgresql_database" {
  description = "PostgreSQL database name"
  value       = azurerm_postgresql_flexible_server_database.main.name
}
