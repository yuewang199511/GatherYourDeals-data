variable "run_id" {
  description = "GitHub Actions run ID — appended to all resource names to guarantee uniqueness per run"
  type        = string
}

variable "location" {
  description = "Azure region for all ephemeral resources — must match gyd-persistent region"
  type        = string
  default     = "westus"
}

variable "apps_subnet_id" {
  description = "Resource ID of subnet-apps in gyd-vnet (delegated to Microsoft.App/environments). Set via TF_VAR_apps_subnet_id in CI."
  type        = string
  # Format: /subscriptions/<SUB>/resourceGroups/gyd-persistent/providers/Microsoft.Network/virtualNetworks/gyd-vnet/subnets/subnet-apps
}
