terraform {
  required_version = ">= 1.9.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  # Partial backend config on purpose: bucket/key/region/table are
  # account-specific and a backend block cannot use variables, so they're
  # supplied at `terraform init` time via -backend-config, keeping this file
  # identical across every fork/account.
  #
  # backend.hcl's dynamodb_table locks via a DynamoDB table (Task 1's
  # bootstrap already provisions one). Terraform >= 1.10 offers S3-native
  # locking (`use_lockfile = true`) as the non-deprecated replacement; not
  # switched yet since the DynamoDB table is already live and working.
  backend "s3" {}
}

provider "aws" {
  region = var.aws_region
  # No hardcoded profile — see Global Constraints in the implementation plan
  # for this operator's own auth workaround, which lives in shell commands,
  # never in this file.
}
