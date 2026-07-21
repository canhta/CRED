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
