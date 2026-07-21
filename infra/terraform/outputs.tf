# Filled in by later tasks as each resource is added.

output "ecr_repository_url" {
  value = aws_ecr_repository.cred.repository_url
}

output "ec2_instance_id" {
  value = aws_instance.cred.id
}

output "elastic_ip" {
  value = aws_eip.cred.public_ip
}
