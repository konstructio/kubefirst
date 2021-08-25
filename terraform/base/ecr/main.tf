module "metaphor-repository" {
  source = "./templates/container-registry"

  repo_name      = "metaphor"
  aws_account_id = var.aws_account_id
}