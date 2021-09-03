variable "gitlab_token" {
  type = string
}

#* private git repos
resource "argocd_repository_credentials" "private" {
  url      = "https://gitlab.<AWS_HOSTED_ZONE_NAME>"
  username = "argocd"
  password = var.gitlab_token
}

resource "argocd_repository" "kubefirst_argocd" {
  repo = "https://gitlab.<AWS_HOSTED_ZONE_NAME>/kubefirst/gitops.git"
  type = "git"
  depends_on = [argocd_repository_credentials.private]
}

#* public helm chart repos
resource "argocd_repository" "bitnami" {
  repo = "https://charts.bitnami.com/bitnami"
  type = "helm"
  name = "bitnami"
}

resource "argocd_repository" "chartmuseum" {
  repo = "https://chartmuseum.github.io/charts"
  type = "helm"
  name = "chartmuseum"
}

resource "argocd_repository" "datadog" {
  repo = "https://helm.datadoghq.com"
  type = "helm"
  name = "datadog"
}

resource "argocd_repository" "external_secrets" {
  repo = "https://external-secrets.github.io/kubernetes-external-secrets"
  type = "helm"
  name = "external-secrets"
}

resource "argocd_repository" "gitlab" {
  repo = "https://charts.gitlab.io"
  type = "helm"
  name = "gitlab"
}

resource "argocd_repository" "hashicorp" {
  repo = "https://helm.releases.hashicorp.com"
  type = "helm"
  name = "hashicorp"
}

resource "argocd_repository" "ingress_nginx" {
  repo = "https://kubernetes.github.io/ingress-nginx"
  type = "helm"
  name = "ingress-nginx"
}

resource "argocd_repository" "jet_stack" {
  repo = "https://charts.jetstack.io"
  type = "helm"
  name = "jet-stack"
}

resource "argocd_repository" "kube2iam" {
  repo = "https://jtblin.github.io/kube2iam"
  type = "helm"
  name = "kube2iam"
}

resource "argocd_repository" "runatlantis" {
  repo = "https://runatlantis.github.io/helm-charts"
  type = "helm"
  name = "atlantis"
}
