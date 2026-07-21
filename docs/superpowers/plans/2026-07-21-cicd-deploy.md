# CI/CD build and deployment — implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stand up a Terraform-provisioned AWS test environment (EC2 + RDS,
`ap-southeast-1`, profile `canhta`) and a manual GitHub Actions pipeline that
builds the `cred` binary into a container and deploys it to that environment.

**Architecture:** One EC2 `t4g.micro` instance runs two containers (`cred`,
`caddy`) via Docker Compose; Caddy terminates TLS for `cred.quickdemo.site`
using its own ACME client, no ACM/Route 53 involved. A separate RDS
`db.t4g.micro` PostgreSQL 17 instance holds the app's data (pgvector already
enabled by an existing migration). GitHub Actions authenticates to AWS via
OIDC (no long-lived keys), builds and pushes the image to ECR, then updates
the running instance via `aws ssm send-command` — chosen so port 22 never
has to open.

**Tech Stack:** Terraform (AWS provider `~> 5.0`), Docker (multi-stage build,
`distroless/static-debian12:nonroot`), Docker Compose on the instance, Caddy 2
for TLS, GitHub Actions with OIDC (`aws-actions/configure-aws-credentials`),
AWS SSM Run Command for the deploy step.

## Global Constraints

- **CRED is OSS — nothing in `infra/` may hardcode this operator's account ID,
  AWS profile name, or personal domain.** Every account-specific value is a
  Terraform variable with **no default** (`github_repo`, `domain_name`,
  `db_master_password`) or derived at apply time from the authenticated
  account (`data.aws_caller_identity.current.account_id`), never a literal.
  A fork deploying under a different AWS account supplies its own
  `terraform.tfvars` and `backend.hcl` and changes nothing else.
- This operator's own values, for reference while executing this plan (never
  baked into a committed file as a default): AWS account `931628308308`,
  region `ap-southeast-1`, GitHub repo `canhta/CRED` (verified via
  `git remote -v`), domain `cred.quickdemo.site`.
- **This operator's `canhta` CLI profile authenticates via a `login_session`
  key that is not part of AWS CLI's standard config schema** — Terraform's AWS
  provider cannot resolve it directly, confirmed by a live `terraform apply`
  failing with "No valid credential sources found" even with the profile
  named in the provider block. The workaround is local to this operator, not
  part of the reusable design: before any `terraform` command, run
  `eval "$(aws configure export-credentials --profile canhta --format env)"`
  to resolve the session into plain `AWS_ACCESS_KEY_ID` /
  `AWS_SECRET_ACCESS_KEY` / `AWS_SESSION_TOKEN` env vars, which Terraform's
  default credential chain does understand. A fork using a normal profile or
  SSO config does not need this step.
- An OIDC provider for `token.actions.githubusercontent.com` **already exists**
  in this operator's account (created 2026-07-15, tagged to an unrelated
  project). Never create a second one — reference it via a Terraform data
  source. Never modify or delete the existing `gha-sandbox` IAM role. (This is
  specific to this account; a fresh AWS account a fork deploys into may need
  the data source to become a resource instead — call this out if
  `terraform plan` errors with "no matching OpenIDConnectProvider found".)
- No SSH / port 22 anywhere. Shell access to the instance is via SSM Session
  Manager only.
- `CGO_ENABLED=0` always, matching every other build path in this repo
  (`.claude/rules/go.md` §3).
- The app's real environment variables (verified in `internal/config/config.go`):
  `DATABASE_URL`, `CRED_LLM_API_KEY`, `CRED_WEB_ADDR` (default `:8080`). There
  is **no session-signing secret** — sessions are random tokens hashed and
  stored in the database (`internal/api/auth.go:40-100`), not derived from a
  shared secret.
- The domain (`var.domain_name`, no default) is expected to be managed
  wherever the forker's registrar is — Caddy's ACME HTTP-01 challenge needs
  only ports 80/443 reachable and an A record pointing at the instance, no
  Route 53/ACM dependency for any fork.
- Free-tier eligibility for EC2/RDS is account-specific and **unconfirmed for
  this operator's account**
  (`docs/superpowers/specs/2026-07-21-cicd-deploy-design.md`, Cost section).
  Task 6 and Task 7 below create billable resources and each starts with a
  STOP step to confirm eligibility in Billing → Free Tier first.

---

## File Structure

```
infra/
  terraform/
    bootstrap/
      main.tf              # S3 state bucket + DynamoDB lock table (applied once, locally)
    versions.tf             # terraform block, partial backend "s3" {}, provider "aws"
    variables.tf            # aws_region, github_repo, domain_name, db_master_password — no
                             # account-specific defaults
    backend.hcl.example     # template for -backend-config; each fork copies to backend.hcl (gitignored)
    terraform.tfvars.example # template for github_repo/domain_name; each fork copies to terraform.tfvars (gitignored)
    data.tf                 # default VPC/subnets, existing OIDC provider, caller identity, AMI lookup
    network.tf               # aws_security_group.ec2, aws_security_group.rds
    ecr.tf                   # aws_ecr_repository.cred, lifecycle policy
    iam.tf                   # EC2 instance role/profile, GitHub OIDC deploy role
    ec2.tf                   # aws_instance.cred, aws_eip.cred
    rds.tf                   # aws_db_subnet_group, aws_db_instance.cred
    outputs.tf               # instance id, EIP, ECR URL, RDS address, deploy role ARN
    templates/
      user_data.sh.tftpl     # cloud-init script, embeds docker-compose.deploy.yml + rendered Caddyfile
      Caddyfile.tftpl        # ${domain_name} placeholder, filled from var.domain_name
    .gitignore
  docker-compose.deploy.yml  # checked in; no account-specific values; embedded via user_data
Dockerfile                   # repo root; multi-stage, mirrors `task build`
.dockerignore
.github/workflows/deploy.yml # workflow_dispatch only
```

---

### Task 1: Terraform state backend (bootstrap) — DONE

**Files:**
- Create: `infra/terraform/bootstrap/main.tf`
- Create: `infra/terraform/bootstrap/.gitignore`

**Interfaces:**
- Produces: an S3 bucket named `cred-tfstate-<account-id>` (derived from
  whichever account is authenticated, never hardcoded) and a DynamoDB table
  `cred-tfstate-lock`. Neither is Terraform-cross-referenced from the main
  config — a backend block cannot read another config's state — the operator
  fills in the bucket name via `backend.hcl` in Task 2.

- [x] **Step 1: Write the bootstrap config**

```hcl
# infra/terraform/bootstrap/main.tf
terraform {
  required_version = ">= 1.9.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

variable "aws_region" {
  description = "AWS region for the state backend. No default — every fork picks its own."
  type        = string
}

provider "aws" {
  region = var.aws_region
  # No hardcoded profile: authenticate with whatever the shell already has
  # active (a named profile, SSO, or plain env-var credentials). A fork
  # deploying under a different account/auth method needs no change here.
}

data "aws_caller_identity" "current" {}

resource "aws_s3_bucket" "tfstate" {
  # Derived from the authenticated account, not hardcoded, so a fork gets its
  # own uniquely-named bucket automatically.
  bucket = "cred-tfstate-${data.aws_caller_identity.current.account_id}"

  lifecycle {
    prevent_destroy = true
  }
}

resource "aws_s3_bucket_versioning" "tfstate" {
  bucket = aws_s3_bucket.tfstate.id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "tfstate" {
  bucket = aws_s3_bucket.tfstate.id
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_public_access_block" "tfstate" {
  bucket                  = aws_s3_bucket.tfstate.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_dynamodb_table" "tflock" {
  name         = "cred-tfstate-lock"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "LockID"

  attribute {
    name = "LockID"
    type = "S"
  }

  lifecycle {
    prevent_destroy = true
  }
}

output "state_bucket" {
  value = aws_s3_bucket.tfstate.bucket
}

output "lock_table" {
  value = aws_dynamodb_table.tflock.name
}
```

- [x] **Step 2: Apply it (this is a one-time, local-only apply — never run from CI)**

```bash
cd infra/terraform/bootstrap
eval "$(aws configure export-credentials --profile canhta --format env)"
terraform init
terraform apply -auto-approve -var="aws_region=ap-southeast-1"
```

`-var="aws_region=..."` is required now that the variable has no default —
every fork passes its own region explicitly, this operator's happens to be
`ap-southeast-1`.

The `eval "$(aws configure export-credentials ...)"` line is this operator's
own workaround for the `canhta` profile's non-standard `login_session` auth
(see Global Constraints) — resolve credentials this way whenever a plain
`AWS_PROFILE=...` fails with "No valid credential sources found". A fork using
a normal profile just runs `terraform apply` directly.

**Actual result:** `Apply complete! Resources: 5 added, 0 changed, 0
destroyed.` — `state_bucket = "cred-tfstate-931628308308"`,
`lock_table = "cred-tfstate-lock"`.

- [x] **Step 3: Verify the resources exist**

```bash
aws s3api head-bucket --bucket cred-tfstate-931628308308 --profile canhta && echo "bucket OK"
aws dynamodb describe-table --table-name cred-tfstate-lock --profile canhta \
  --query "Table.TableStatus" --output text
```

**Actual result:** `bucket OK`, then `ACTIVE`.

- [ ] **Step 4: Commit**

```bash
git add infra/terraform/bootstrap/main.tf
git commit -m "infra: bootstrap the Terraform state backend (S3 + DynamoDB lock)"
```

---

### Task 2: Terraform root skeleton — providers, data sources, variables

**Files:**
- Create: `infra/terraform/versions.tf`
- Create: `infra/terraform/variables.tf`
- Create: `infra/terraform/backend.hcl.example`
- Create: `infra/terraform/terraform.tfvars.example`
- Create: `infra/terraform/data.tf`
- Create: `infra/terraform/outputs.tf` (empty stub, filled in by later tasks)
- Create: `infra/terraform/.gitignore`

**Interfaces:**
- Consumes: nothing Terraform-managed from Task 1 — the state bucket/table
  names are supplied at `terraform init` time via `-backend-config`, not
  cross-referenced (a backend block can't read another config's state or use
  variables/data sources at all — a hard Terraform limitation, not a design
  choice here).
- Produces: `data.aws_vpc.default`, `data.aws_subnets.default`,
  `data.aws_iam_openid_connect_provider.github`, `data.aws_ami.al2023_arm64`,
  `data.aws_caller_identity.current`, and variables `var.aws_region`,
  `var.github_repo`, `var.domain_name`, `var.db_master_password` — all
  consumed by Tasks 3-8.

- [ ] **Step 1: Write `versions.tf` with a partial backend block**

```hcl
# infra/terraform/versions.tf
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
  backend "s3" {}
}

provider "aws" {
  region = var.aws_region
  # No hardcoded profile — see Global Constraints for this operator's own
  # auth workaround, which lives in shell commands, never in this file.
}
```

- [ ] **Step 2: Write `variables.tf` — no account-specific defaults**

```hcl
# infra/terraform/variables.tf
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
```

- [ ] **Step 3: Write `backend.hcl.example` and `terraform.tfvars.example` —
  the templates a fork copies and fills in**

```hcl
# infra/terraform/backend.hcl.example
# Copy to backend.hcl (gitignored) and fill in your own state bucket/table —
# see infra/terraform/bootstrap/main.tf, whose outputs are exactly these
# two values.
bucket         = "cred-tfstate-<your-account-id>"
key            = "cred/terraform.tfstate"
region         = "<your-region>"
dynamodb_table = "cred-tfstate-lock"
encrypt        = true
```

```hcl
# infra/terraform/terraform.tfvars.example
# Copy to terraform.tfvars (gitignored) and fill in your own values.
# db_master_password does NOT go here — set it via
# TF_VAR_db_master_password instead, so it never touches a file at all.
aws_region  = "your-region"
github_repo = "your-github-username/your-fork"
domain_name = "your-domain.example.com"
```

This operator's own (not committed as defaults) values:
`aws_region = "ap-southeast-1"`,
`github_repo = "canhta/CRED"`, `domain_name = "cred.quickdemo.site"`.

- [ ] **Step 4: Write `data.tf`**

```hcl
# infra/terraform/data.tf
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

# Verified via `aws iam get-open-id-connect-provider`: this provider already
# exists in the account (created for an unrelated project). AWS allows only
# one provider per URL per account, so it is looked up here, never created.
data "aws_iam_openid_connect_provider" "github" {
  url = "https://token.actions.githubusercontent.com"
}

data "aws_ami" "al2023_arm64" {
  most_recent = true
  owners      = ["amazon"]

  filter {
    name   = "name"
    values = ["al2023-ami-*-arm64"]
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
```

- [ ] **Step 5: Write an empty `outputs.tf` stub**

```hcl
# infra/terraform/outputs.tf
# Filled in by later tasks as each resource is added.
```

- [ ] **Step 6: Write `infra/terraform/.gitignore`**

```
.terraform/
*.tfstate
*.tfstate.*
*.auto.tfvars
terraform.tfvars
backend.hcl
crash.log
```

`terraform.tfvars` and `backend.hcl` are gitignored deliberately — only the
`.example` templates are tracked, so each fork's actual values (repo slug,
domain, state bucket name) never touch git history.

- [ ] **Step 7: Create this operator's own (gitignored) `backend.hcl` and
  `terraform.tfvars` from the examples**

```bash
cd infra/terraform
cp backend.hcl.example backend.hcl
cp terraform.tfvars.example terraform.tfvars
```

Then edit `backend.hcl` to set `bucket = "cred-tfstate-931628308308"` (from
Task 1's `state_bucket` output) and edit `terraform.tfvars` to set
`github_repo = "canhta/CRED"` and `domain_name = "cred.quickdemo.site"`.

- [ ] **Step 8: Init and validate (no resources yet — this only proves the
  backend and data sources resolve)**

```bash
cd infra/terraform
eval "$(aws configure export-credentials --profile canhta --format env)"
terraform init -backend-config=backend.hcl
TF_VAR_db_master_password=placeholder terraform validate
TF_VAR_db_master_password=placeholder terraform plan
```

Expected: `terraform init` reports
`Successfully configured the backend "s3"!`; `terraform validate` reports
`Success! The configuration is valid.`; `terraform plan` reports
`No changes. Your infrastructure matches the configuration.` (data sources
only, no resources declared yet).

- [ ] **Step 9: Commit (the tracked files only — `backend.hcl` and
  `terraform.tfvars` stay local, per `.gitignore`)**

```bash
git add infra/terraform/versions.tf infra/terraform/variables.tf \
  infra/terraform/backend.hcl.example infra/terraform/terraform.tfvars.example \
  infra/terraform/data.tf infra/terraform/outputs.tf infra/terraform/.gitignore
git commit -m "infra: Terraform root skeleton — provider, partial backend, data sources"
```

---

### Task 3: Security groups

**Files:**
- Create: `infra/terraform/network.tf`

**Interfaces:**
- Consumes: `data.aws_vpc.default` (Task 2).
- Produces: `aws_security_group.ec2`, `aws_security_group.rds` — consumed by
  Task 6 (EC2), Task 7 (RDS).

- [ ] **Step 1: Write `network.tf`**

```hcl
# infra/terraform/network.tf
resource "aws_security_group" "ec2" {
  name        = "cred-ec2-sg"
  description = "Inbound HTTP/HTTPS for the cred test box; no SSH — shell access is via SSM."
  vpc_id      = data.aws_vpc.default.id

  ingress {
    description = "HTTP, needed for Caddy's ACME HTTP-01 challenge"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    description = "HTTPS"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "cred-ec2-sg"
  }
}

resource "aws_security_group" "rds" {
  name        = "cred-rds-sg"
  description = "Postgres access from the cred EC2 instance only."
  vpc_id      = data.aws_vpc.default.id

  ingress {
    description     = "Postgres from the cred app instance"
    from_port       = 5432
    to_port         = 5432
    protocol        = "tcp"
    security_groups = [aws_security_group.ec2.id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "cred-rds-sg"
  }
}
```

- [ ] **Step 2: Plan and apply**

```bash
cd infra/terraform
eval "$(aws configure export-credentials --profile canhta --format env)"
TF_VAR_db_master_password=placeholder terraform plan
TF_VAR_db_master_password=placeholder terraform apply
```

Expected plan: `Plan: 2 to add, 0 to change, 0 to destroy.` Apply completes
with `Apply complete! Resources: 2 added, 0 changed, 0 destroyed.`

- [ ] **Step 3: Verify**

```bash
aws ec2 describe-security-groups --profile canhta --region ap-southeast-1 \
  --filters Name=group-name,Values=cred-ec2-sg,cred-rds-sg \
  --query 'SecurityGroups[].[GroupName,GroupId]' --output text
```

Expected: two rows, one per group name, each with a `sg-...` id.

- [ ] **Step 4: Commit**

```bash
git add infra/terraform/network.tf
git commit -m "infra: security groups for the cred EC2 instance and RDS"
```

---

### Task 4: ECR repository

**Files:**
- Create: `infra/terraform/ecr.tf`
- Modify: `infra/terraform/outputs.tf`

**Interfaces:**
- Produces: `aws_ecr_repository.cred` — consumed by Task 5 (EC2 pull policy)
  and Task 8 (GitHub deploy role push policy). Produces
  `ecr_repository_url` output — consumed by Task 7 Step 5 (migration
  verification) and set as a GitHub repository variable in Task 11.

- [ ] **Step 1: Write `ecr.tf`**

```hcl
# infra/terraform/ecr.tf
resource "aws_ecr_repository" "cred" {
  name                 = "cred"
  image_tag_mutability = "MUTABLE" # `latest` is re-pushed on every deploy

  image_scanning_configuration {
    scan_on_push = true
  }

  tags = {
    Name = "cred"
  }
}

resource "aws_ecr_lifecycle_policy" "cred" {
  repository = aws_ecr_repository.cred.name

  policy = jsonencode({
    rules = [
      {
        rulePriority = 1
        description  = "Expire untagged images after 1 day"
        selection = {
          tagStatus   = "untagged"
          countType   = "sinceImagePushed"
          countUnit   = "days"
          countNumber = 1
        }
        action = { type = "expire" }
      },
      {
        rulePriority = 2
        description  = "Keep only the 10 most recently pushed images overall"
        selection = {
          tagStatus   = "any"
          countType   = "imageCountMoreThan"
          countNumber = 10
        }
        action = { type = "expire" }
      }
    ]
  })
}
```

`tagStatus = "any"` matters here: an earlier draft of this rule used
`tagStatus = "tagged"` with `tagPrefixList = ["latest"]`, which only ever
counts the single `latest` tag and never expires old SHA-tagged images —
storage would grow unbounded despite the rule appearing to cap it. `"any"`
counts every image regardless of tag.

- [ ] **Step 2: Add the output (in this task, not deferred to Task 8 — Task
  7's migration-verification step needs it, and there's no reason to make it
  wait)**

```hcl
# append to infra/terraform/outputs.tf
output "ecr_repository_url" {
  value = aws_ecr_repository.cred.repository_url
}
```

- [ ] **Step 3: Plan and apply**

```bash
cd infra/terraform
eval "$(aws configure export-credentials --profile canhta --format env)"
TF_VAR_db_master_password=placeholder terraform apply
```

Expected: `Plan: 2 to add, 0 to change, 0 to destroy.` (outputs aren't
resources, so the count doesn't change) then apply succeeds.

- [ ] **Step 4: Verify**

```bash
cd infra/terraform
terraform output -raw ecr_repository_url
```

Expected output (account- and region-specific to whoever applied this —
this operator sees `931628308308.dkr.ecr.ap-southeast-1.amazonaws.com/cred`):
the repo's URI. Then:

```bash
eval "$(aws configure export-credentials --profile canhta --format env)"
aws ecr get-lifecycle-policy --region ap-southeast-1 \
  --repository-name cred --query 'lifecyclePolicyText' --output text
```

Expected: the JSON policy with both rules.

- [ ] **Step 5: Commit**

```bash
git add infra/terraform/ecr.tf
git commit -m "infra: ECR repository for the cred image, with a lifecycle policy"
```

---

### Task 5: EC2 instance role

**Files:**
- Create: `infra/terraform/iam.tf` (EC2-facing portion only — the GitHub OIDC
  deploy role is added to this same file in Task 8, once the EC2 instance ARN
  it references exists)

**Interfaces:**
- Consumes: `aws_ecr_repository.cred.arn` (Task 4).
- Produces: `aws_iam_instance_profile.cred_ec2` — consumed by Task 6 (EC2
  instance).

- [ ] **Step 1: Write the EC2 role/profile portion of `iam.tf`**

```hcl
# infra/terraform/iam.tf

# EC2 instance role: SSM Session Manager (so the instance never needs SSH)
# plus read-only ECR pull scoped to this one repository.
resource "aws_iam_role" "cred_ec2" {
  name = "cred-ec2-instance-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "ec2.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
}

resource "aws_iam_role_policy_attachment" "cred_ec2_ssm" {
  role       = aws_iam_role.cred_ec2.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
}

resource "aws_iam_role_policy" "cred_ec2_ecr_pull" {
  name = "cred-ec2-ecr-pull"
  role = aws_iam_role.cred_ec2.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid      = "ECRAuth"
        Effect   = "Allow"
        Action   = "ecr:GetAuthorizationToken"
        Resource = "*"
      },
      {
        Sid    = "ECRPull"
        Effect = "Allow"
        Action = [
          "ecr:BatchCheckLayerAvailability",
          "ecr:GetDownloadUrlForLayer",
          "ecr:BatchGetImage",
        ]
        Resource = aws_ecr_repository.cred.arn
      }
    ]
  })
}

resource "aws_iam_instance_profile" "cred_ec2" {
  name = "cred-ec2-instance-profile"
  role = aws_iam_role.cred_ec2.name
}
```

- [ ] **Step 2: Plan and apply**

```bash
cd infra/terraform
eval "$(aws configure export-credentials --profile canhta --format env)"
TF_VAR_db_master_password=placeholder terraform apply
```

Expected: `Plan: 4 to add, 0 to change, 0 to destroy.` then apply succeeds.

- [ ] **Step 3: Verify**

```bash
aws iam get-role --profile canhta --role-name cred-ec2-instance-role \
  --query 'Role.Arn' --output text
aws iam list-attached-role-policies --profile canhta \
  --role-name cred-ec2-instance-role --query 'AttachedPolicies[].PolicyName' --output text
```

Expected: an `arn:aws:iam::931628308308:role/cred-ec2-instance-role` ARN,
then `AmazonSSMManagedInstanceCore`.

- [ ] **Step 4: Commit**

```bash
git add infra/terraform/iam.tf
git commit -m "infra: EC2 instance role — SSM Session Manager + scoped ECR pull"
```

---

### Task 6: EC2 instance + Elastic IP

> **STOP before this step.** This creates a billable EC2 instance. Confirm
> free-tier eligibility for `t4g.micro` in the AWS Console under
> **Billing and Cost Management → Free Tier** for account `931628308308`
> before applying. If the account is on the newer credit-based free tier
> instead of the classic 12-month per-service one, this instance still works
> but bills against that credit balance instead of being free — check which
> applies before proceeding.

**Files:**
- Create: `infra/docker-compose.deploy.yml`
- Create: `infra/terraform/templates/Caddyfile.tftpl`
- Create: `infra/terraform/templates/user_data.sh.tftpl`
- Create: `infra/terraform/ec2.tf`
- Modify: `infra/terraform/outputs.tf`

**Interfaces:**
- Consumes: `aws_security_group.ec2` (Task 3), `aws_iam_instance_profile.cred_ec2`
  (Task 5), `data.aws_ami.al2023_arm64` (Task 2), `data.aws_subnets.default`
  (Task 2), `var.domain_name` (Task 2, rendered into the Caddyfile template).
- Produces: `aws_instance.cred`, `aws_eip.cred` — `aws_instance.cred.arn`
  is consumed by Task 8 (GitHub deploy role's `ssm:SendCommand` policy);
  `aws_eip.cred.public_ip` is the address the operator points Hostinger's DNS
  A record at.

- [ ] **Step 1: Write `infra/docker-compose.deploy.yml`**

```yaml
# infra/docker-compose.deploy.yml
# No hardcoded ECR URL — it's account-specific. ECR_IMAGE and IMAGE_TAG are
# both supplied at deploy time (see Task 11's .env assembly), so this file is
# identical across every fork/account.
services:
  cred:
    image: ${ECR_IMAGE}:${IMAGE_TAG:-latest}
    restart: unless-stopped
    env_file: .env
    expose:
      - "8080"

  caddy:
    image: caddy:2
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
      - caddy-data:/data
    depends_on:
      - cred

volumes:
  caddy-data:
```

- [ ] **Step 2: Write `infra/terraform/templates/Caddyfile.tftpl`** — a
  template, not a static file with a literal domain baked in, so this file is
  identical across every fork; `var.domain_name` fills in the one placeholder
  at apply time.

```
${domain_name} {
	reverse_proxy cred:8080
}
```

- [ ] **Step 3: Write `infra/terraform/templates/user_data.sh.tftpl`**

```bash
#!/bin/bash
set -euo pipefail

dnf install -y docker
systemctl enable --now docker
usermod -aG docker ec2-user

mkdir -p /usr/libexec/docker/cli-plugins
curl -sSL "https://github.com/docker/compose/releases/download/v2.29.7/docker-compose-linux-$(uname -m)" \
  -o /usr/libexec/docker/cli-plugins/docker-compose
chmod +x /usr/libexec/docker/cli-plugins/docker-compose

mkdir -p /opt/cred

cat > /opt/cred/docker-compose.deploy.yml <<'COMPOSE_EOF'
${docker_compose_content}
COMPOSE_EOF

cat > /opt/cred/Caddyfile <<'CADDY_EOF'
${caddyfile_content}
CADDY_EOF

# Written empty at boot; the first deploy.yml run populates real secrets.
touch /opt/cred/.env

cat > /etc/systemd/system/cred.service <<'UNIT_EOF'
[Unit]
Description=cred app stack
Requires=docker.service
After=docker.service network-online.target
Wants=network-online.target

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=/opt/cred
ExecStart=/usr/libexec/docker/cli-plugins/docker-compose -f docker-compose.deploy.yml up -d
ExecStop=/usr/libexec/docker/cli-plugins/docker-compose -f docker-compose.deploy.yml down

[Install]
WantedBy=multi-user.target
UNIT_EOF

systemctl daemon-reload
systemctl enable cred.service
```

- [ ] **Step 4: Write `infra/terraform/ec2.tf`**

```hcl
# infra/terraform/ec2.tf
resource "aws_instance" "cred" {
  ami                    = data.aws_ami.al2023_arm64.id
  instance_type          = "t4g.micro"
  subnet_id              = data.aws_subnets.default.ids[0]
  vpc_security_group_ids = [aws_security_group.ec2.id]
  iam_instance_profile   = aws_iam_instance_profile.cred_ec2.name

  user_data = templatefile("${path.module}/templates/user_data.sh.tftpl", {
    docker_compose_content = file("${path.module}/../docker-compose.deploy.yml")
    caddyfile_content = templatefile("${path.module}/templates/Caddyfile.tftpl", {
      domain_name = var.domain_name
    })
  })

  tags = {
    Name = "cred-test"
  }
}

resource "aws_eip" "cred" {
  instance = aws_instance.cred.id
  domain   = "vpc"

  tags = {
    Name = "cred-test-eip"
  }
}
```

- [ ] **Step 5: Add outputs**

```hcl
# append to infra/terraform/outputs.tf
output "ec2_instance_id" {
  value = aws_instance.cred.id
}

output "elastic_ip" {
  value = aws_eip.cred.public_ip
}
```

- [ ] **Step 6: Plan and apply**

```bash
cd infra/terraform
eval "$(aws configure export-credentials --profile canhta --format env)"
TF_VAR_db_master_password=placeholder terraform apply
```

Expected: `Plan: 2 to add, 0 to change, 0 to destroy.` then apply succeeds,
printing `ec2_instance_id` and `elastic_ip` in the outputs.

- [ ] **Step 7: Verify the instance booted and set itself up, via SSM (no SSH)**

```bash
INSTANCE_ID=$(cd infra/terraform && terraform output -raw ec2_instance_id)
aws ssm describe-instance-information --profile canhta --region ap-southeast-1 \
  --filters "Key=InstanceIds,Values=$INSTANCE_ID" \
  --query 'InstanceInformationList[0].PingStatus' --output text
```

Expected: `Online` (may take 2-3 minutes after `apply` for the SSM agent to
register — retry if it prints nothing yet).

```bash
aws ssm send-command --profile canhta --region ap-southeast-1 \
  --instance-ids "$INSTANCE_ID" --document-name "AWS-RunShellScript" \
  --parameters 'commands=["docker --version","test -f /opt/cred/docker-compose.deploy.yml && echo compose-file-present","systemctl is-enabled cred.service"]' \
  --query 'Command.CommandId' --output text
```

Take the printed command ID and check its result:

```bash
aws ssm get-command-invocation --profile canhta --region ap-southeast-1 \
  --instance-id "$INSTANCE_ID" --command-id <command-id-from-above> \
  --query 'StandardOutputContent' --output text
```

Expected output contains a Docker version line, `compose-file-present`, and
`enabled`.

- [ ] **Step 8: Commit**

```bash
git add infra/docker-compose.deploy.yml infra/terraform/templates/Caddyfile.tftpl \
  infra/terraform/templates/user_data.sh.tftpl infra/terraform/ec2.tf \
  infra/terraform/outputs.tf
git commit -m "infra: EC2 instance + Elastic IP, Docker/Compose via user_data"
```

---

### Task 7: RDS instance

> **STOP before this step**, same reason as Task 6 — `db.t4g.micro` is a
> billable resource. Re-confirm free-tier status before applying if you
> haven't already for this session.

**Files:**
- Create: `infra/terraform/rds.tf`
- Modify: `infra/terraform/outputs.tf`

**Interfaces:**
- Consumes: `aws_security_group.rds` (Task 3), `data.aws_subnets.default`
  (Task 2), `var.db_master_password` (Task 2).
- Produces: `aws_db_instance.cred.address` — consumed by Task 11
  (`.github/workflows/deploy.yml`'s `DATABASE_URL` assembly, via a GitHub
  Actions repository variable set from this output).

- [ ] **Step 1: Write `rds.tf`**

```hcl
# infra/terraform/rds.tf
resource "aws_db_subnet_group" "cred" {
  name       = "cred-test-db-subnets"
  subnet_ids = data.aws_subnets.default.ids

  tags = {
    Name = "cred-test-db-subnets"
  }
}

resource "aws_db_instance" "cred" {
  identifier = "cred-test-db"
  engine     = "postgres"
  # A specific 17.x is required — never left to default. This repo's
  # docker-compose.yml pins pg17 for the same reason: PG18 changed the PGDATA
  # path and silently loses data on the wrong volume mount. "17.10" was
  # verified available via `aws rds describe-db-engine-versions --engine
  # postgres --engine-version 17` in this deployment's region
  # (ap-southeast-1) — re-run that check before deploying to a different
  # region, since available minor versions vary by region.
  engine_version = "17.10"
  instance_class = "db.t4g.micro"

  allocated_storage = 20
  storage_type      = "gp3"

  db_name  = "cred"
  username = "cred"
  password = var.db_master_password

  db_subnet_group_name   = aws_db_subnet_group.cred.name
  vpc_security_group_ids = [aws_security_group.rds.id]
  publicly_accessible    = false
  multi_az                = false
  skip_final_snapshot     = true
  deletion_protection     = false

  tags = {
    Name = "cred-test-db"
  }
}
```

- [ ] **Step 2: Add the output**

```hcl
# append to infra/terraform/outputs.tf
output "rds_address" {
  value = aws_db_instance.cred.address
}
```

- [ ] **Step 3: Apply with the real master password (not the `placeholder`
  used in earlier tasks — pick a strong one and keep it, it's needed again in
  Task 11 as a GitHub secret)**

```bash
cd infra/terraform
eval "$(aws configure export-credentials --profile canhta --format env)"
TF_VAR_db_master_password='<your real password>' terraform apply
```

Expected: `Plan: 2 to add, 0 to change, 0 to destroy.` Apply can take
5-10 minutes for RDS to reach `available`.

- [ ] **Step 4: Verify**

```bash
aws rds describe-db-instances --profile canhta --region ap-southeast-1 \
  --db-instance-identifier cred-test-db \
  --query 'DBInstances[0].[DBInstanceStatus,EngineVersion,Endpoint.Address]' --output text
```

Expected: `available`, `17.10`, and the RDS endpoint hostname.

- [ ] **Step 5: Confirm the app's migration runs against RDS and pgvector is
  enabled.** Run this from the EC2 instance via SSM (it already has network
  access to RDS; your laptop does not, since the RDS security group only
  allows the EC2 instance's security group):

```bash
cd infra/terraform
INSTANCE_ID=$(terraform output -raw ec2_instance_id)
RDS_ADDRESS=$(terraform output -raw rds_address)
ECR_URL=$(terraform output -raw ecr_repository_url)  # available since Task 4
ECR_REGISTRY=${ECR_URL%%/*}                           # host part, before the first "/"
cd - >/dev/null

aws ssm send-command --profile canhta --region ap-southeast-1 \
  --instance-ids "$INSTANCE_ID" --document-name "AWS-RunShellScript" \
  --parameters "commands=[\"aws ecr get-login-password --region ap-southeast-1 | docker login --username AWS --password-stdin ${ECR_REGISTRY}\",\"docker run --rm -e DATABASE_URL='postgres://cred:<your real password>@${RDS_ADDRESS}:5432/cred?sslmode=require' ${ECR_URL}:latest /cred migrate\"]" \
  --query 'Command.CommandId' --output text
```

`ECR_URL` is no longer a hardcoded fallback — it's always available by this
point because Task 4 (not Task 8) is where `ecr_repository_url` is defined,
and Task 4 runs before this task.

This step depends on an image already existing at `:latest` in ECR — if
Task 9 (Dockerfile) and Task 10 (build-and-push workflow) haven't run yet,
skip this verification for now and come back to it after Task 10, before
moving on to Task 11.

Once it runs, check the output:

```bash
aws ssm get-command-invocation --profile canhta --region ap-southeast-1 \
  --instance-id "$INSTANCE_ID" --command-id <command-id-from-above> \
  --query 'StandardOutputContent' --output text
```

Expected: a line like `migrate    version 0 -> 5, river tables ensured`.

- [ ] **Step 6: Commit**

```bash
git add infra/terraform/rds.tf infra/terraform/outputs.tf
git commit -m "infra: RDS PostgreSQL 17 instance for the cred test environment"
```

---

### Task 8: GitHub OIDC deploy role

**Files:**
- Modify: `infra/terraform/iam.tf` (append the deploy role)
- Modify: `infra/terraform/outputs.tf`

**Interfaces:**
- Consumes: `data.aws_iam_openid_connect_provider.github` (Task 2),
  `aws_ecr_repository.cred.arn` (Task 4), `aws_instance.cred.arn` (Task 6),
  `var.github_repo` (Task 2).
- Produces: `aws_iam_role.gha_deploy.arn` — this is the value set as the
  `CRED_DEPLOY_ROLE_ARN` GitHub Actions repository variable, consumed by
  Task 10 and Task 11's workflow steps.

- [ ] **Step 1: Append the deploy role to `iam.tf`**

```hcl
# append to infra/terraform/iam.tf

# GitHub Actions deploy role. Trusts the account's EXISTING OIDC provider —
# do not create a second aws_iam_openid_connect_provider; AWS allows only one
# per URL per account, and this account already has one for a different
# project. Scoped to this repo's main branch only.
resource "aws_iam_role" "gha_deploy" {
  name = "cred-gha-deploy-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Federated = data.aws_iam_openid_connect_provider.github.arn }
      Action    = "sts:AssumeRoleWithWebIdentity"
      Condition = {
        StringEquals = {
          "token.actions.githubusercontent.com:aud" = "sts.amazonaws.com"
          "token.actions.githubusercontent.com:sub" = "repo:${var.github_repo}:ref:refs/heads/main"
        }
      }
    }]
  })
}

resource "aws_iam_role_policy" "gha_deploy" {
  name = "cred-gha-deploy-policy"
  role = aws_iam_role.gha_deploy.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid      = "ECRAuth"
        Effect   = "Allow"
        Action   = "ecr:GetAuthorizationToken"
        Resource = "*"
      },
      {
        Sid    = "ECRPush"
        Effect = "Allow"
        Action = [
          "ecr:BatchCheckLayerAvailability",
          "ecr:GetDownloadUrlForLayer",
          "ecr:BatchGetImage",
          "ecr:PutImage",
          "ecr:InitiateLayerUpload",
          "ecr:UploadLayerPart",
          "ecr:CompleteLayerUpload",
        ]
        Resource = aws_ecr_repository.cred.arn
      },
      {
        Sid    = "SendCommand"
        Effect = "Allow"
        Action = "ssm:SendCommand"
        Resource = [
          aws_instance.cred.arn,
          "arn:aws:ssm:${var.aws_region}::document/AWS-RunShellScript",
        ]
      },
      {
        Sid      = "ReadCommandStatus"
        Effect   = "Allow"
        Action   = "ssm:GetCommandInvocation"
        Resource = "*"
      }
    ]
  })
}
```

- [ ] **Step 2: Add the output** (`ecr_repository_url` already exists from
  Task 4 — only this one is new here)

```hcl
# append to infra/terraform/outputs.tf
output "gha_deploy_role_arn" {
  value = aws_iam_role.gha_deploy.arn
}
```

- [ ] **Step 3: Apply**

```bash
cd infra/terraform
eval "$(aws configure export-credentials --profile canhta --format env)"
TF_VAR_db_master_password='<your real password>' terraform apply
```

Expected: `Plan: 2 to add, 0 to change, 0 to destroy.`

- [ ] **Step 4: Verify the trust condition is scoped correctly and the
  pre-existing `gha-sandbox` role is untouched**

```bash
aws iam get-role --profile canhta --role-name cred-gha-deploy-role \
  --query 'Role.AssumeRolePolicyDocument.Statement[0].Condition' --output json
aws iam get-role --profile canhta --role-name gha-sandbox \
  --query 'Role.AssumeRolePolicyDocument.Statement[0].Condition.StringEquals."token.actions.githubusercontent.com:sub"' \
  --output text
```

Expected: the first prints a condition with `...:sub` =
`repo:canhta/CRED:ref:refs/heads/main`; the second still prints
`repo:Seta-International/agent-platform:environment:sandbox`, unchanged.

- [ ] **Step 5: Commit**

```bash
git add infra/terraform/iam.tf infra/terraform/outputs.tf
git commit -m "infra: GitHub OIDC deploy role, scoped to canhta/CRED main"
```

---

### Task 9: Dockerfile

**Files:**
- Create: `Dockerfile`
- Create: `.dockerignore`

**Interfaces:**
- Consumes: nothing from earlier tasks — pure repo build artifact.
- Produces: a container image runnable as `docker run -p 8080:8080 cred:test`
  — consumed by Task 10 (the CI build step builds this same Dockerfile).

- [ ] **Step 1: Write `.dockerignore`**

```
.git
.github
docs
.superpowers
infra
node_modules
web/node_modules
web/dist
cred
*.md
```

- [ ] **Step 2: Write `Dockerfile`**

```dockerfile
# syntax=docker/dockerfile:1

FROM node:22-bookworm AS web
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /web/dist ./web/dist
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build -tags embed -o /cred .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /cred /cred
EXPOSE 8080
ENTRYPOINT ["/cred", "web"]
```

`ARG TARGETARCH` is set automatically by BuildKit/buildx from the
`--platform` flag — no default needed for CI (Task 10 always passes
`--platform linux/arm64` explicitly).

- [ ] **Step 3: Build it locally to verify (native platform — substitute
  `linux/amd64` for `linux/arm64` if your machine is Apple Silicon)**

```bash
docker buildx build --platform linux/amd64 -t cred:test --load .
```

Expected: build completes through all three stages with no errors, ending
`naming to docker.io/library/cred:test`.

- [ ] **Step 4: Verify the built image actually starts and serves**

```bash
docker run -d --name cred-smoke -p 8080:8080 \
  -e DATABASE_URL="postgres://cred:cred@host.docker.internal:5433/cred?sslmode=disable" \
  cred:test
sleep 2
curl -sf http://localhost:8080/ >/dev/null && echo "reachable"
docker rm -f cred-smoke
```

This requires the local `docker-compose.yml` Postgres running
(`docker compose up -d db`) so the container has something to connect to.
Expected: `reachable`. (If it fails to connect, that's a local-dev
networking detail unrelated to this plan — the point of this check is that
the binary starts and serves HTTP, not that it reaches a real database yet.)

- [ ] **Step 5: Commit**

```bash
git add Dockerfile .dockerignore
git commit -m "build: add the multi-stage Dockerfile for the cred image"
```

---

### Task 10: GitHub Actions — build and push

**Files:**
- Create: `.github/workflows/deploy.yml` (this task writes the
  `build-and-push` job only; Task 11 appends the `deploy` job)

**Interfaces:**
- Consumes: `Dockerfile` (Task 9), the `CRED_DEPLOY_ROLE_ARN` repository
  variable (set from Task 8's `gha_deploy_role_arn` output).
- Produces: an image at
  `931628308308.dkr.ecr.ap-southeast-1.amazonaws.com/cred:<sha>` and `:latest`
  — consumed by Task 7's Step 5 (migration verification) and Task 11 (deploy
  job).

- [ ] **Step 1: Set the repository variables from Task 8's output and this
  operator's own region — nothing here is a workflow-file default, both live
  as repo variables so a fork sets its own without touching `deploy.yml`**

```bash
ROLE_ARN=$(cd infra/terraform && terraform output -raw gha_deploy_role_arn)
gh variable set CRED_DEPLOY_ROLE_ARN --body "$ROLE_ARN"
gh variable set CRED_AWS_REGION --body "ap-southeast-1"
```

Expected: `gh` reports both variables were set. Verify:

```bash
gh variable list | grep CRED_DEPLOY_ROLE_ARN
```

- [ ] **Step 2: Write `.github/workflows/deploy.yml`**

```yaml
name: deploy

on:
  workflow_dispatch:
    inputs:
      image_tag:
        description: "Image tag to build and deploy (defaults to the commit SHA)"
        required: false
        type: string

permissions:
  id-token: write
  contents: read

concurrency:
  group: cred-deploy
  cancel-in-progress: false

env:
  # No hardcoded region: every fork sets its own CRED_AWS_REGION repository
  # variable (Step 1 below). ECR_REPOSITORY is the project's own name, not an
  # account-specific value — every fork's Terraform also creates it as "cred"
  # (infra/terraform/ecr.tf), so it stays a literal here.
  AWS_REGION: ${{ vars.CRED_AWS_REGION }}
  ECR_REPOSITORY: cred

jobs:
  build-and-push:
    runs-on: ubuntu-latest
    outputs:
      image_tag: ${{ steps.tag.outputs.image_tag }}
    steps:
      - uses: actions/checkout@v5
        with: { persist-credentials: false }

      - id: tag
        run: echo "image_tag=${{ inputs.image_tag || github.sha }}" >> "$GITHUB_OUTPUT"

      - uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: ${{ vars.CRED_DEPLOY_ROLE_ARN }}
          aws-region: ${{ env.AWS_REGION }}

      - id: ecr-login
        uses: aws-actions/amazon-ecr-login@v2

      - uses: docker/setup-qemu-action@v3
      - uses: docker/setup-buildx-action@v3

      - name: Build and push
        uses: docker/build-push-action@v6
        with:
          context: .
          platforms: linux/arm64
          push: true
          tags: |
            ${{ steps.ecr-login.outputs.registry }}/${{ env.ECR_REPOSITORY }}:${{ steps.tag.outputs.image_tag }}
            ${{ steps.ecr-login.outputs.registry }}/${{ env.ECR_REPOSITORY }}:latest
```

- [ ] **Step 3: Commit and push (a workflow file needs to exist on `main` to
  be dispatchable)**

```bash
git add .github/workflows/deploy.yml
git commit -m "ci: add the deploy workflow's build-and-push job"
git push
```

- [ ] **Step 4: Trigger it manually and verify**

```bash
gh workflow run deploy.yml
gh run watch
```

Expected: the run succeeds (green). Then confirm the image landed:

```bash
aws ecr describe-images --profile canhta --region ap-southeast-1 \
  --repository-name cred --query 'imageDetails[].imageTags' --output text
```

Expected: a list including the commit SHA and `latest`.

- [ ] **Step 5 (deferred from Task 7): now run Task 7's Step 5** — the
  migration verification against RDS — since an image now exists at
  `:latest`.

---

### Task 11: GitHub Actions — deploy job

**Files:**
- Modify: `.github/workflows/deploy.yml` (append the `deploy` job)

**Interfaces:**
- Consumes: `needs.build-and-push.outputs.image_tag` (Task 10),
  `CRED_INSTANCE_ID` / `CRED_RDS_ADDRESS` repository variables (set below
  from Task 6/7 outputs), `CRED_DB_PASSWORD` / `CRED_LLM_API_KEY` GitHub
  secrets.
- Produces: a running, updated `cred` container on the EC2 instance.

- [ ] **Step 1: Set the remaining repository variables and secrets**

```bash
INSTANCE_ID=$(cd infra/terraform && terraform output -raw ec2_instance_id)
RDS_ADDRESS=$(cd infra/terraform && terraform output -raw rds_address)

gh variable set CRED_INSTANCE_ID --body "$INSTANCE_ID"
gh variable set CRED_RDS_ADDRESS --body "$RDS_ADDRESS"

gh secret set CRED_DB_PASSWORD    # paste the same password used in Task 7, Step 3
gh secret set CRED_LLM_API_KEY    # paste your Anthropic/OpenAI/DeepSeek key, or an empty value to disable auto-capture
```

Expected: `gh variable list` shows `CRED_INSTANCE_ID` and `CRED_RDS_ADDRESS`;
`gh secret list` shows `CRED_DB_PASSWORD` and `CRED_LLM_API_KEY` (values
never shown, that's expected for secrets).

- [ ] **Step 2: Append the `deploy` job to `.github/workflows/deploy.yml`**

```yaml
  deploy:
    needs: build-and-push
    runs-on: ubuntu-latest
    steps:
      - uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: ${{ vars.CRED_DEPLOY_ROLE_ARN }}
          aws-region: ${{ env.AWS_REGION }}

      - name: Send deploy command
        id: send
        run: |
          set -euo pipefail
          ENV_FILE=$(mktemp)
          {
            echo "DATABASE_URL=postgres://cred:${{ secrets.CRED_DB_PASSWORD }}@${{ vars.CRED_RDS_ADDRESS }}:5432/cred?sslmode=require"
            echo "CRED_LLM_API_KEY=${{ secrets.CRED_LLM_API_KEY }}"
            echo "CRED_WEB_ADDR=:8080"
          } > "$ENV_FILE"

          B64_ENV=$(base64 -w0 "$ENV_FILE")
          IMAGE_TAG="${{ needs.build-and-push.outputs.image_tag }}"

          COMMAND_ID=$(aws ssm send-command \
            --instance-ids "${{ vars.CRED_INSTANCE_ID }}" \
            --document-name "AWS-RunShellScript" \
            --comment "cred deploy ${IMAGE_TAG}" \
            --parameters "{\"commands\":[\"echo $B64_ENV | base64 -d > /opt/cred/.env\", \"cd /opt/cred && IMAGE_TAG=$IMAGE_TAG /usr/libexec/docker/cli-plugins/docker-compose -f docker-compose.deploy.yml pull\", \"cd /opt/cred && IMAGE_TAG=$IMAGE_TAG /usr/libexec/docker/cli-plugins/docker-compose -f docker-compose.deploy.yml up -d\"]}" \
            --query "Command.CommandId" --output text)

          echo "command_id=$COMMAND_ID" >> "$GITHUB_OUTPUT"

      - name: Wait for the command to finish
        run: |
          set -euo pipefail
          for i in $(seq 1 30); do
            STATUS=$(aws ssm get-command-invocation \
              --command-id "${{ steps.send.outputs.command_id }}" \
              --instance-id "${{ vars.CRED_INSTANCE_ID }}" \
              --query "Status" --output text 2>/dev/null || echo "Pending")
            echo "status: $STATUS"
            if [ "$STATUS" = "Success" ]; then
              exit 0
            fi
            if [ "$STATUS" = "Failed" ] || [ "$STATUS" = "Cancelled" ] || [ "$STATUS" = "TimedOut" ]; then
              aws ssm get-command-invocation \
                --command-id "${{ steps.send.outputs.command_id }}" \
                --instance-id "${{ vars.CRED_INSTANCE_ID }}"
              exit 1
            fi
            sleep 10
          done
          echo "timed out waiting for command"
          exit 1
```

- [ ] **Step 3: Commit and push**

```bash
git add .github/workflows/deploy.yml
git commit -m "ci: add the deploy job — ships secrets and updates the instance via SSM"
git push
```

- [ ] **Step 4: Trigger the full pipeline and verify**

```bash
gh workflow run deploy.yml
gh run watch
```

Expected: both jobs succeed. Then confirm the container actually updated:

```bash
INSTANCE_ID=$(cd infra/terraform && terraform output -raw ec2_instance_id)
aws ssm send-command --profile canhta --region ap-southeast-1 \
  --instance-ids "$INSTANCE_ID" --document-name "AWS-RunShellScript" \
  --parameters 'commands=["docker ps --format \"{{.Names}}: {{.Status}}\""]' \
  --query 'Command.CommandId' --output text
```

Then, with that command ID:

```bash
aws ssm get-command-invocation --profile canhta --region ap-southeast-1 \
  --instance-id "$INSTANCE_ID" --command-id <command-id-from-above> \
  --query 'StandardOutputContent' --output text
```

Expected: two lines, `cred-cred-1: Up ...` and `cred-caddy-1: Up ...` (exact
container names depend on Compose's project-naming, but both services show
`Up`).

---

### Task 12: DNS and end-to-end verification

**Files:** none — this task is entirely manual operator steps plus a
verification checklist. No code changes.

- [ ] **Step 1: Point DNS at the instance**

In Hostinger's DNS management for `quickdemo.site`, add an A record:
`cred` → the `elastic_ip` value from `terraform output -raw elastic_ip`.

- [ ] **Step 2: Wait for DNS propagation, then verify the certificate**

```bash
dig +short cred.quickdemo.site
```

Expected: prints the Elastic IP once propagated (can take a few minutes to
a few hours depending on the previous record's TTL, if any existed).

```bash
curl -vI https://cred.quickdemo.site/ 2>&1 | grep -E "subject:|issuer:|HTTP/"
```

Expected: `issuer: ... Let's Encrypt ...` (or `ZeroSSL`, Caddy's fallback CA)
and an `HTTP/2 200` or `HTTP/1.1 200` line — confirms Caddy obtained a real
certificate and is proxying to the app.

- [ ] **Step 3: Manual smoke test in a browser**

Open `https://cred.quickdemo.site/` and confirm:
- The console loads (login page, since the console is gated behind a
  session per the existing `beforeLoad` redirect logic).
- Registering/logging in works end-to-end against the real RDS-backed store.
- The Recall and Claims pages load without errors.

- [ ] **Step 4: Record the outcome**

If everything above passes, the environment is live and this plan is
complete. If free-tier eligibility from Task 6/7's STOP steps turned out
negative, note the actual monthly cost incurred so it's not a surprise on
the next AWS bill.

---

## Self-Review Notes

- **Spec coverage:** every section of
  `docs/superpowers/specs/2026-07-21-cicd-deploy-design.md` maps to a task —
  Terraform bootstrap/backend (Tasks 1-2), security groups (Task 3), ECR
  (Task 4), EC2 + IAM (Tasks 5-6), RDS (Task 7), OIDC deploy role (Task 8),
  Dockerfile (Task 9), the two-job `deploy.yml` (Tasks 10-11), DNS/TLS
  (Task 12). The spec's "Out of scope" items (second environment, automated
  smoke tests, blue/green, Route 53 migration) have no corresponding task,
  which is correct.
- **Type/name consistency checked:** `aws_ecr_repository.cred`,
  `aws_instance.cred`, `aws_security_group.ec2`/`.rds`,
  `aws_iam_instance_profile.cred_ec2`, `aws_iam_role.gha_deploy` are spelled
  identically everywhere they're referenced across tasks. Repository
  variables/secrets (`CRED_DEPLOY_ROLE_ARN`, `CRED_INSTANCE_ID`,
  `CRED_RDS_ADDRESS`, `CRED_DB_PASSWORD`, `CRED_LLM_API_KEY`) are named
  consistently between the `gh variable`/`gh secret` steps and the workflow
  YAML that reads them.
- **No placeholders left in code that ships** — the only literal
  `<placeholder>`-style text is in commands the operator runs by hand
  (passwords, command IDs copy-pasted from a prior step's output), which is
  inherent to interactive CLI verification, not a gap in the plan.
