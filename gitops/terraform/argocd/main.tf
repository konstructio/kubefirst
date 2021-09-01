variable "gitlab_token" {
  type = string
}

module "argocd_repos" {
  source = "./repos"
  gitlab_token = var.gitlab_token
}

module "argocd_manifests" {
  source = "./manifests"
  depends_on module.argocd_repos
}
