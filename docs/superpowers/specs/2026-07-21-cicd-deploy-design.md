# CI/CD build and deployment — design

- **Date:** 2026-07-21
- **Status:** Approved by operator, pending implementation plan.

## Problem

The repo has no deployment target, no `Dockerfile`, and no IaC. `ci.yml`
(`.github/workflows/ci.yml`) already lints, guards CGO, and runs unit,
integration, and race suites on every push to `main` — that stays as-is. What's
missing is a path from a passing commit to a running instance the operator can
open in a browser, provisioned via Terraform against their own AWS account
(profile `canhta`, `ap-southeast-1`), for hands-on testing rather than a
public release.

## Approach

**One environment, EC2 + RDS, both free-tier, manual deploy.** Rejected
alternatives, in the order they were considered:

- **ECS Fargate.** Fargate requires an Application Load Balancer, which bills
  hourly regardless of traffic (~USD 16-20/month) — the single largest fixed
  cost of any option evaluated, and not free-tier eligible. Rejected on cost
  for a low-traffic test environment.
- **App Runner + RDS.** No ALB, but still a second paid-looking primitive
  (App Runner's own compute billing) layered over an already-free-tier-eligible
  EC2 instance. Once free-tier EC2 was on the table, App Runner added
  complexity without reducing cost below zero.
- **Single EC2 instance running app + Postgres together** (docker-compose,
  matching local dev). Cheapest possible if free tier didn't apply, but with
  free tier available for *both* EC2 and RDS (750 hours/month each, counted
  independently, 12 months), coupling app and database onto one box trades
  away a cleaner separation for no cost benefit. Rejected once free tier
  changed the comparison.
- **Automatic deploy on push to `main`.** Fastest feedback loop, but the
  operator explicitly wants manual control over when the test box updates.
- **AWS Secrets Manager / SSM Parameter Store for app secrets.** Secrets
  Manager costs ~USD 0.40/month per secret; SSM Parameter Store (standard
  tier) is free but the operator specifically wants GitHub Actions secrets as
  the source of truth. Honored as stated.
- **Route 53 + ACM for TLS.** The domain (`cred.quickdemo.site`) is managed in
  Hostinger, not Route 53, and ACM's DNS validation doesn't reach a
  non-Route-53 zone without manual CNAME steps each renewal. Caddy's built-in
  ACME HTTP-01 challenge needs only ports 80/443 reachable and the A record
  pointing at the instance — no AWS certificate machinery at all.

## Architecture

```
GitHub Actions (OIDC — no long-lived AWS keys stored as secrets)
  ci.yml (existing, unchanged)      — lint / cgo-guard / tests, gates main
  deploy.yml (new, workflow_dispatch only)
    build-and-push  — docker build (multi-stage) → ECR, tagged with git SHA
    deploy          — assemble .env from GitHub secrets, aws ssm send-command
                       → instance pulls the new tag, restarts via compose

AWS ap-southeast-1 (profile: canhta)
  EC2 t4g.micro (free tier, Graviton/arm64)      RDS db.t4g.micro (free tier)
    container: cred   (from ECR)  ──── 5432 ────►  PostgreSQL 17 + pgvector
    container: caddy  (TLS for cred.quickdemo.site)  single-AZ, no public access
    IAM instance role: SSM Session Manager + scoped ECR pull (no port 22)
  Elastic IP  ◄── DNS A record (operator configures in Hostinger)
  Security group: 80/443 open; 5432 open to the EC2 security group only
```

Shell access to the instance is via SSM Session Manager, not SSH — the
instance role carries `AmazonSSMManagedInstanceCore` and the security group
never opens port 22. This is a real reduction in attack surface, not a
stylistic choice: with SSH there is a key to lose and a port to scan; with
Session Manager, access is gated entirely by IAM, which the operator already
controls.

## Terraform

New directory `infra/terraform/`, a single root module — one environment does
not earn a `modules/` split. Resources:

- **Bootstrap** (`infra/terraform/bootstrap/`, applied once, locally, not from
  CI): an S3 bucket + DynamoDB lock table for the main config's state. This
  has to precede the backend it creates, so it is a separate config with its
  own (tiny, local) state file, run by hand a single time.
- **EC2**: one `t4g.micro` instance (arm64/Graviton — free-tier eligible, and
  cheaper than `t3.micro` once the free-tier window ends), Amazon Linux 2023,
  `user_data` installing Docker + the Compose plugin and creating `/opt/cred`.
  An Elastic IP attached to it.
- **IAM**: an instance role with `AmazonSSMManagedInstanceCore` plus an
  inline policy scoped to `ecr:GetAuthorizationToken` /
  `BatchGetImage` / `GetDownloadUrlForLayer` against the one `cred` ECR repo —
  no broader ECR or account access.
- **ECR**: a private repository named `cred`, with a lifecycle policy
  expiring untagged images and keeping only the most recent tagged images
  (exact count decided in the implementation plan), so storage never grows
  unbounded.
- **RDS**: `db.t4g.micro`, engine `postgres` 17.x, 20 GB gp3, single-AZ (the
  free tier does not cover Multi-AZ), `publicly_accessible = false`, in a
  security group that allows inbound 5432 only from the EC2 instance's
  security group.
- **Security groups**: `ec2_sg` (ingress 80/443 from `0.0.0.0/0`, no 22),
  `rds_sg` (ingress 5432 from `ec2_sg` only).
- **GitHub OIDC**: **Verified** (`aws iam get-open-id-connect-provider`,
  account `931628308308`) an `aws_iam_openid_connect_provider` for
  `token.actions.githubusercontent.com` already exists in this account —
  created 2026-07-15, tagged to an unrelated project (`future-app`,
  `sandbox`). AWS allows only one provider per URL per account, so Terraform
  references it via `data "aws_iam_openid_connect_provider"`, not a resource —
  creating a second would fail. Only a new IAM role is created: trusted for
  `repo:canhta/CRED:ref:refs/heads/main` (this repo's actual slug, verified
  via `git remote -v`), scoped to push to the one ECR repo and
  `ssm:SendCommand` / `ssm:GetCommandInvocation` against the one EC2 instance.
  No long-lived AWS access keys are stored as GitHub secrets. The existing
  `gha-sandbox` role (trusted for a different repo/environment entirely) is
  untouched.
- **Outputs**: EC2 instance ID, Elastic IP, ECR repository URL, RDS endpoint,
  OIDC deploy role ARN — the values `deploy.yml` and the operator's Hostinger
  DNS step both need.

Default VPC and subnets are used via data sources — no custom VPC, no NAT
gateway (a NAT gateway alone runs ~USD 32/month and nothing here needs
outbound-only private subnets badly enough to justify it).

## Docker build

New `Dockerfile` at the repo root, mirroring `Taskfile.yml`'s `build` task
exactly:

```
FROM node:22 AS web
WORKDIR /web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /web/dist ./web/dist
RUN CGO_ENABLED=0 GOARCH=arm64 go build -tags embed -o /cred .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /cred /cred
EXPOSE 8080
ENTRYPOINT ["/cred", "web"]
```

`GOARCH=arm64` targets the `t4g` instance family. GitHub-hosted runners are
x86_64, so the build step uses `docker buildx build --platform linux/arm64`
(QEMU emulation) — slower than a native build, but this pipeline runs on
manual trigger only, so the extra minute or two per deploy is not a cost that
compounds.

**Verified**: `internal/store/migrations/00001_initial_schema.sql:7` already
runs `CREATE EXTENSION IF NOT EXISTS vector`, so no separate Terraform step is
needed to enable pgvector on RDS — the existing goose migration path (run via
the app's own migrate command) covers it, same as local dev.

## Deploy mechanism

A checked-in `infra/docker-compose.deploy.yml` (two services: `cred` from
ECR, and `caddy:2` for TLS) plus a `Caddyfile` routing
`cred.quickdemo.site` to `cred:8080` — both copied onto the instance by
`user_data` at launch, so a deploy never needs to re-ship them.

`.github/workflows/deploy.yml`, `workflow_dispatch` only:

1. **build-and-push**: OIDC-assume the deploy role, `docker buildx build`,
   tag with the git SHA (and `latest`), push both tags to ECR.
2. **deploy** (needs `build-and-push`): OIDC-assume the deploy role, assemble
   an `.env` blob from GitHub Actions secrets — the RDS master password (used
   to build `DATABASE_URL`) and `CRED_LLM_API_KEY` — plus the RDS endpoint (a
   Terraform output, stored as a repo variable, not a secret), then
   `aws ssm send-command`
   (`AWS-RunShellScript`) targeting the instance, running a script that writes
   `/opt/cred/.env` and does `docker compose -f docker-compose.deploy.yml pull
   && up -d`. The workflow polls `aws ssm get-command-invocation` until the
   command finishes and fails the job if it didn't succeed.

**Named trade-off**: SSM command parameters (including the assembled `.env`
contents) are visible in that AWS account's Systems Manager console and
CloudTrail history. This is acceptable here only because the account has a
single principal — the operator themself — with no other IAM users who could
read that history. GitHub Actions still redacts secret values from the
*workflow logs themselves*; the exposure is scoped to AWS-side history in an
account only the operator can access. If this environment ever gets a second
person with AWS console access, this stops being acceptable and the deploy
mechanism needs to move to SSM Parameter Store SecureString or Secrets
Manager instead.

## Cost

| Resource | During free tier (12 months) | After |
|---|---|---|
| EC2 t4g.micro | $0 (750 hrs/month) | ~USD 6-7/month |
| RDS db.t4g.micro, 20 GB | $0 (750 hrs/month + storage) | ~USD 12-13/month |
| ECR storage (<500 MB expected) | $0 | ~USD 0.05/month |
| Elastic IP (attached) | $0 | $0 (only unattached EIPs bill) |
| Data transfer | $0 (100 GB/month included) | usage-dependent, likely near $0 |
| **Total** | **~$0/month** | **~USD 20-22/month** |

**Checked, still unresolved**: `aws freetier get-free-tier-usage` (account
`931628308308`) returns real usage lines for Glue, KMS, Lambda, SNS, SQS, and
CloudWatch, but none for EC2 or RDS — because neither has ever been launched
in this account, not because of eligibility either way. That API only reports
usage for services already used, so it cannot answer the question for a
service not yet provisioned. Eligibility still has to be confirmed in
Billing → Free Tier before applying — this design assumes classic 12-month
free tier, not the newer credit-based model some accounts now have, and nothing
checked so far confirms which one applies here.

Also **verified**: no existing EC2 instances, RDS instances, or ECR
repositories in `ap-southeast-1` on this profile, and the default VPC
(`vpc-03f3f7d41672fe92a`) exists and is usable as-is — a clean slate, no
collision risk for the resources this design adds.

**Operator is authenticated as the account root user**
(`arn:aws:iam::931628308308:root`), not an IAM user or role. AWS recommends
against using root for routine work, including `terraform apply` — it can't
be scoped down by IAM policy the way a dedicated deploy user or role can. Not
a blocker for this design, but worth creating an IAM user/role for Terraform
operations before this becomes a recurring workflow.

## Testing / verification

- `ci.yml` is unchanged and continues to gate `main` — this design adds no
  new Go code, so no new unit/integration tests are needed there.
- `terraform plan` reviewed by the operator before every `terraform apply`
  (manual step, not automated in CI — Terraform applies against a personal
  AWS account are not something this design automates).
- First deploy is verified end-to-end manually: trigger `deploy.yml`, confirm
  the SSM command succeeds, confirm `cred.quickdemo.site` serves the console
  over HTTPS with a valid Let's Encrypt certificate, confirm recall/claims
  pages load real data from the RDS-backed store.
- No automated smoke test against the live instance in this design — out of
  scope, see below.

## Out of scope

- A second environment (staging/prod split). One test environment only; nothing
  here blocks adding another later by copying the Terraform config with a
  different `terraform.tfvars`.
- Automated post-deploy smoke tests / health-check gating the SSM command's
  success.
- Blue/green or zero-downtime deploy — `docker compose up -d` briefly drops
  the container, acceptable for a test environment.
- Migrating from Hostinger DNS to Route 53.
- Any change to `ci.yml`'s existing lint/test/build gates.
