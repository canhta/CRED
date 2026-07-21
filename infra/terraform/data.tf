data "aws_caller_identity" "current" {}

data "aws_vpc" "default" {
  default = true
}

data "aws_subnets" "default" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.default.id]
  }
}

# Looked up, not created: this account already has one, and AWS allows only
# one per URL per account.
data "aws_iam_openid_connect_provider" "github" {
  url = "https://token.actions.githubusercontent.com"
}

data "aws_ami" "al2023_arm64" {
  most_recent = true
  owners      = ["amazon"]

  filter {
    # "al2023-ami-2*", not "al2023-ami-*": the wildcard also matched the
    # "minimal" edition (al2023-ami-minimal-...), which strips out the
    # pre-installed SSM agent — instances built from it never register with
    # Session Manager, discovered when a real instance sat at PingStatus
    # "None" indefinitely despite correct networking and IAM.
    name   = "name"
    values = ["al2023-ami-2*-arm64"]
  }

  filter {
    name   = "architecture"
    values = ["arm64"]
  }

  filter {
    name   = "root-device-type"
    values = ["ebs"]
  }
}
