variable "location" {
  description = "Azure region for all persistent resources"
  type        = string
  default     = "westus"
}

variable "pg_admin_password" {
  description = "PostgreSQL admin password — stored in Bitwarden, passed via TF_VAR_pg_admin_password"
  type        = string
  sensitive   = true
}
