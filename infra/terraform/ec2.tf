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
