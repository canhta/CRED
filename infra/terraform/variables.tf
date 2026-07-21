variable "aws_region" {
  description = "AWS region for the cred test environment. No default — every fork picks its own."
  type        = string
}

variable "github_repo" {
  description = "GitHub repo allowed to assume the deploy role, owner/name form. No default — every fork must supply its own."
  type        = string
}

variable "domain_name" {
  description = "Public domain Caddy issues a TLS certificate for. No default — every fork must supply its own."
  type        = string
}

variable "db_master_password" {
  description = <<-EOT
    RDS master password. Never committed. Supply via
    TF_VAR_db_master_password, never via a *.tfvars file.
  EOT
  type      = string
  sensitive = true
}
