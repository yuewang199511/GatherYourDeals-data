terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 4.0"
    }
  }
}

# Authentication is handled by ARM_* environment variables injected by
# the azure/login@v2 action (OIDC federated credential).
provider "azurerm" {
  features {}
  use_oidc = true
}

locals {
  # All names share this suffix so every run has a distinct, isolated RG.
  # ACR names are alphanumeric only; other names allow hyphens.
  name_suffix  = "gyd-lt-${var.run_id}"
  acr_name     = "gydlt${var.run_id}"
  pg_admin     = "gydadmin"
}

# ── Resource Group ─────────────────────────────────────────────────────────────
# Single RG contains all ephemeral resources. Destroying the RG cascades to
# every resource inside it, including the Container App created separately by CI.
resource "azurerm_resource_group" "main" {
  name     = local.name_suffix
  location = var.location

  tags = {
    purpose    = "load-test"
    run_id     = var.run_id
    managed_by = "terraform"
  }
}

# ── Container Registry ─────────────────────────────────────────────────────────
# Basic SKU is cheapest; admin_enabled lets the CI runner authenticate with
# username/password when creating the Container App (admin creds, ephemeral).
resource "azurerm_container_registry" "main" {
  name                = local.acr_name
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
  sku                 = "Basic"
  admin_enabled       = true
}

# ── PostgreSQL Flexible Server ─────────────────────────────────────────────────
resource "azurerm_postgresql_flexible_server" "main" {
  name                   = local.name_suffix
  resource_group_name    = azurerm_resource_group.main.name
  location               = azurerm_resource_group.main.location
  version                = "16"
  administrator_login    = local.pg_admin
  administrator_password = var.pg_admin_password

  # B_Standard_B2ms: 2 vCPU / 4 GB — enough headroom for load test traffic.
  sku_name   = "B_Standard_B2ms"
  storage_mb = 32768

  # Minimum allowed by Azure; geo-redundancy off to cut cost.
  backup_retention_days        = 7
  geo_redundant_backup_enabled = false

  # Public access is required so the CI runner can run psql for DB reset/seed,
  # and so the ACI container can reach the DB over the public internet.
  # The firewall rule below restricts this to the test window only.
  public_network_access_enabled = true
}

resource "azurerm_postgresql_flexible_server_database" "main" {
  name      = "gatheryourdeals"
  server_id = azurerm_postgresql_flexible_server.main.id
}

# Allow all IPs. This RG is ephemeral (destroyed after each test run), so the
# risk window is bounded by the test duration (~45 min).
resource "azurerm_postgresql_flexible_server_firewall_rule" "allow_all" {
  name             = "allow-all"
  server_id        = azurerm_postgresql_flexible_server.main.id
  start_ip_address = "0.0.0.0"
  end_ip_address   = "255.255.255.255"
}

# ── Container Apps Environment ─────────────────────────────────────────────────
# Consumption plan: scale to zero when idle, billed per vCPU-second used.
# This matches Railway's business model — no idle cost between load test runs.
resource "azurerm_container_app_environment" "main" {
  name                = local.name_suffix
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
}
