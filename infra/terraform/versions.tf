terraform {
  required_version = ">= 1.9.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  # Partial on purpose: a backend block can't use variables, so
  # bucket/key/region/table come from -backend-config at init time instead —
  # keeps this file identical across every fork/account.
  backend "s3" {}
}

provider "aws" {
  region = var.aws_region
  # No profile: auth comes from whatever the shell has active.
}
