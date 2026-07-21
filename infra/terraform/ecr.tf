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
