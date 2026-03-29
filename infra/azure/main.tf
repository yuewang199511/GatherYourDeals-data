terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 4.0"
    }
  }
}

# Authentication via ARM_CLIENT_ID / ARM_CLIENT_SECRET / ARM_TENANT_ID /
# ARM_SUBSCRIPTION_ID set by the azure/login step in CI.
provider "azurerm" {
  features {}
  # The SP has Contributor, which cannot bulk-register all providers.
  # We register only the namespaces we need via `az provider register` in CI.
  resource_provider_registrations = "none"
}

locals {
  # All ephemeral names share this suffix so every run has a distinct, isolated RG.
  # ACR names are alphanumeric only; other names allow hyphens.
  name_suffix = "gyd-lt-${var.run_id}"
  acr_name    = "gydlt${var.run_id}"
}

# ── Resource Group ─────────────────────────────────────────────────────────────
# Single RG contains all ephemeral resources for this run. Deleting it cascades
# to everything inside, including the Container App created by CLI after Terraform.
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
# Basic SKU is cheapest; admin_enabled lets CI authenticate for image push/pull.
resource "azurerm_container_registry" "main" {
  name                = local.acr_name
  resource_group_name = azurerm_resource_group.main.name
  location            = azurerm_resource_group.main.location
  sku                 = "Basic"
  admin_enabled       = true
}

# ── Container Apps Environment ─────────────────────────────────────────────────
# Consumption plan: scale to zero, billed per vCPU-second — matches Railway's model.
# VNet integration via infrastructure_subnet_id lets Container Apps reach the
# persistent private PostgreSQL at gyd-pg-main.private.postgres.database.azure.com.
#
# Prerequisites (one-time manual setup):
#   az network vnet subnet update \
#     --name subnet-apps \
#     --resource-group gyd-persistent \
#     --vnet-name gyd-vnet \
#     --delegations Microsoft.App/environments
resource "azurerm_container_app_environment" "main" {
  name                     = local.name_suffix
  resource_group_name      = azurerm_resource_group.main.name
  location                 = azurerm_resource_group.main.location
  infrastructure_subnet_id = var.apps_subnet_id
}
