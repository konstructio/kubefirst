module "registry" {
  source = "../templates/argocd-app"

  app_name          = "registry"
  resource_path     = "registry"
  recurse_directory = true
}
