output "vnet_id" {
  description = "Resource ID of gyd-vnet"
  value       = azurerm_virtual_network.main.id
}

output "subnet_apps_id" {
  description = "Resource ID of subnet-apps — used by the ephemeral Terraform stack (TF_VAR_apps_subnet_id)"
  value       = azurerm_subnet.apps.id
}

output "subnet_pg_id" {
  description = "Resource ID of subnet-pg"
  value       = azurerm_subnet.pg.id
}

output "postgresql_fqdn" {
  description = "Public FQDN of the PostgreSQL server (not reachable — VNet-only)"
  value       = azurerm_postgresql_flexible_server.main.fqdn
}

output "postgresql_private_fqdn" {
  description = "Private FQDN for in-VNet connections"
  value       = "gyd-pg-main.private.postgres.database.azure.com"
}

output "pg_portal_url" {
  description = "Azure portal link for the PostgreSQL server"
  value       = "https://portal.azure.com/#resource${azurerm_postgresql_flexible_server.main.id}/overview"
}
