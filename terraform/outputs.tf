output "acm_arn" {
  description = "the arn of the acm certificate generated"
  value       = module.acm.acm_certificate_arn
}

output "ecr_repo_arns" {
  description = "ecr repository registry information"
  value       = module.ecr
}

output "gitlab_public_ip" {
  value = module.ec2
}
output "cluster_name" {
  value = module.eks.cluster_name
}