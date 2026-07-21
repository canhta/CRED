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

# Trusts the account's existing OIDC provider (data.aws_iam_openid_connect_provider.github) —
# never a second one. Scoped to this repo's main branch only.
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
        }
        # StringLike, not StringEquals, on sub: a repo or owner with any
        # rename history gets a sub claim of "repo:owner@id/repo@id:ref:..."
        # instead of the plain "repo:owner/repo:ref:...", confirmed via
        # CloudTrail against this repo (previously named SHIFT). Both forms
        # are matched so this works whether or not that history exists.
        StringLike = {
          "token.actions.githubusercontent.com:sub" = [
            "repo:${var.github_repo}:ref:refs/heads/main",
            "repo:${split("/", var.github_repo)[0]}@*/${split("/", var.github_repo)[1]}@*:ref:refs/heads/main",
          ]
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
