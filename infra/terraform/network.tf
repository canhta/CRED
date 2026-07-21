resource "aws_security_group" "ec2" {
  name        = "cred-ec2-sg"
  description = "Inbound HTTP/HTTPS for the cred test box; no SSH, shell access is via SSM."
  vpc_id      = data.aws_vpc.default.id

  ingress {
    description = "HTTP, needed for Caddy ACME HTTP-01 challenge"
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
