module "registry" {
  source = "../templates/argocd-app"

  app_name      = "registry"
  resource_path = "registry"
}
