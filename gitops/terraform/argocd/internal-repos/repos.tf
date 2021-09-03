resource "argocd_repository" "kubefirst-charts" {
  repo     = "https://chartmuseum.<AWS_HOSTED_ZONE_NAME>"
  type     = "helm"
  name     = "kubefirst-charts"
  # username = data.vault_generic_secret.chartmuseum_secrets.data["basic-auth-user"]
  # password = data.vault_generic_secret.chartmuseum_secrets.data["basic-auth-pass"]
}
