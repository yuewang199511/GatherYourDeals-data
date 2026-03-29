terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 4.0"
    }
  }

  # Remote backend — state stored in Azure Blob Storage.
  # From any machine: az login && terraform init
  backend "azurerm" {
    resource_group_name  = "gyd-tf-state"
    storage_account_name = "gydtfstate"
    container_name       = "tfstate"
    key                  = "persistent.tfstate"
  }
}

provider "azurerm" {
  features {}
  resource_provider_registrations = "none"
}

# ─────────────────────────────────────────────────────────────────────────────
# NOTE: All resources below were provisioned manually on 2026-03-28 before this
# Terraform stack existed. To bring them under Terraform management, import
# each resource before running `terraform apply`. Replace <SUB_ID> with your
# Azure subscription ID (AZURE_SUBSCRIPTION_ID secret value).
#
#   terraform import azurerm_resource_group.main \
#     /subscriptions/<SUB_ID>/resourceGroups/gyd-persistent
#
#   terraform import azurerm_virtual_network.main \
#     /subscriptions/<SUB_ID>/resourceGroups/gyd-persistent/providers/Microsoft.Network/virtualNetworks/gyd-vnet
#
#   terraform import azurerm_subnet.pg \
#     /subscriptions/<SUB_ID>/resourceGroups/gyd-persistent/providers/Microsoft.Network/virtualNetworks/gyd-vnet/subnets/subnet-pg
#
#   terraform import azurerm_subnet.apps \
#     /subscriptions/<SUB_ID>/resourceGroups/gyd-persistent/providers/Microsoft.Network/virtualNetworks/gyd-vnet/subnets/subnet-apps
#
#   terraform import azurerm_postgresql_flexible_server.main \
#     /subscriptions/<SUB_ID>/resourceGroups/gyd-persistent/providers/Microsoft.DBforPostgreSQL/flexibleServers/gyd-pg-main
#
#   terraform import azurerm_postgresql_flexible_server_database.main \
#     /subscriptions/<SUB_ID>/resourceGroups/gyd-persistent/providers/Microsoft.DBforPostgreSQL/flexibleServers/gyd-pg-main/databases/gatheryourdeals
#
# To provision from scratch on a new subscription, remove the import notes and
# run `terraform apply` directly.
# ─────────────────────────────────────────────────────────────────────────────

resource "azurerm_resource_group" "main" {
  name     = "gyd-persistent"
  location = var.location
}

# ── Networking ────────────────────────────────────────────────────────────────

resource "azurerm_virtual_network" "main" {
  name                = "gyd-vnet"
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
  address_space       = ["10.0.0.0/16"]
}

# Dedicated to PostgreSQL Flexible Server.
resource "azurerm_subnet" "pg" {
  name                 = "subnet-pg"
  resource_group_name  = azurerm_resource_group.main.name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = ["10.0.1.0/24"]

  delegation {
    name = "pg-delegation"
    service_delegation {
      name    = "Microsoft.DBforPostgreSQL/flexibleServers"
      actions = ["Microsoft.Network/virtualNetworks/subnets/join/action"]
    }
  }
}

# Dedicated to Container Apps Environment (all ephemeral app replicas and jobs).
# Must be delegated before the ephemeral Terraform stack can reference it.
resource "azurerm_subnet" "apps" {
  name                 = "subnet-apps"
  resource_group_name  = azurerm_resource_group.main.name
  virtual_network_name = azurerm_virtual_network.main.name
  address_prefixes     = ["10.0.2.0/24"]

  delegation {
    name = "apps-delegation"
    service_delegation {
      name    = "Microsoft.App/environments"
      actions = ["Microsoft.Network/virtualNetworks/subnets/join/action"]
    }
  }
}

# ── PostgreSQL Flexible Server ─────────────────────────────────────────────────
# Always-on Burstable B1ms (~$15/mo). DSN stored in GitHub secret
# AZURE_PG_DSN_PERSISTENT. Admin password stored in Bitwarden.
resource "azurerm_postgresql_flexible_server" "main" {
  name                   = "gyd-pg-main"
  resource_group_name    = azurerm_resource_group.main.name
  location               = azurerm_resource_group.main.location
  version                = "16"
  administrator_login    = "gydadmin"
  administrator_password = var.pg_admin_password

  sku_name   = "B_Standard_B1ms"
  storage_mb = 32768

  backup_retention_days        = 7
  geo_redundant_backup_enabled = false

  # Private access only — connections go through the VNet.
  public_network_access_enabled = false

  delegated_subnet_id = azurerm_subnet.pg.id
  private_dns_zone_id = azurerm_private_dns_zone.pg.id

  lifecycle {
    # Prevent accidental destruction — this is long-lived infrastructure.
    prevent_destroy = true
  }

  depends_on = [azurerm_private_dns_zone_virtual_network_link.pg]
}

resource "azurerm_postgresql_flexible_server_database" "main" {
  name      = "gatheryourdeals"
  server_id = azurerm_postgresql_flexible_server.main.id

  lifecycle {
    prevent_destroy = true
  }
}

# Private DNS zone — auto-created by Azure when server was provisioned.
# Name format: <server-name>.private.postgres.database.azure.com
resource "azurerm_private_dns_zone" "pg" {
  name                = "gyd-pg-main.private.postgres.database.azure.com"
  resource_group_name = azurerm_resource_group.main.name
}

resource "azurerm_private_dns_zone_virtual_network_link" "pg" {
  name                  = "gyd-vnet-pg-dns-link"
  resource_group_name   = azurerm_resource_group.main.name
  private_dns_zone_name = azurerm_private_dns_zone.pg.name
  virtual_network_id    = azurerm_virtual_network.main.id
  registration_enabled  = false
}
