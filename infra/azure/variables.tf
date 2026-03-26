variable "run_id" {
  description = "GitHub Actions run ID — appended to all resource names to guarantee uniqueness per run"
  type        = string
}

variable "location" {
  description = "Azure region for all resources"
  type        = string
  default     = "eastus"
}

variable "pg_admin_password" {
  description = "Password for the PostgreSQL admin account (gydadmin)"
  type        = string
  sensitive   = true
}
