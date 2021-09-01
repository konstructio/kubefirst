output "gitlab_public_ip" {
  value = module.ec2
}
output "cluster_name" {
  value = module.eks.cluster_name
}

output "eks_module" {
  value = module.eks
}
