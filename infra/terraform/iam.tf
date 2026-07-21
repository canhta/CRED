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
