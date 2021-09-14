variable "gitlab_token" {
  type = string
}

module "argocd_repos" {
  source = "./repos"
  gitlab_token = var.gitlab_token
}

module "argocd_internal_repos" {
  source = "./internal-repos"
}

module "argocd_registry" {
  source = "./registry"
}
