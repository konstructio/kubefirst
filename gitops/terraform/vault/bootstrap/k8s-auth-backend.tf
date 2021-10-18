data "terraform_remote_state" "eks" {
  backend = "s3"
  config = {
    bucket = "<TF_STATE_BUCKET>"
    key    = "terraform/base/tfstate.tf"
    region = var.aws_region
  }
}

data "aws_eks_cluster" "cluster" {
  name = data.terraform_remote_state.eks.outputs.eks_module.cluster_name
}

data "aws_eks_cluster_auth" "cluster" {
  name = data.terraform_remote_state.eks.outputs.eks_module.cluster_name
}

provider "kubernetes" {
  host                   = data.aws_eks_cluster.cluster.endpoint
  cluster_ca_certificate = base64decode(data.aws_eks_cluster.cluster.certificate_authority.0.data)
  token                  = data.aws_eks_cluster_auth.cluster.token
}

resource "vault_auth_backend" "k8s" {
  type = "kubernetes"
  path = "kubernetes/kubefirst"
}

data "kubernetes_service_account" "external_secrets" {
  metadata {
    name = "external-secrets"
    namespace = "external-secrets"
  }
}

data "kubernetes_secret" "external_secrets_token_secret" {
  metadata {
    name = data.kubernetes_service_account.external_secrets.default_secret_name
    namespace = "external-secrets"
  }
}

resource "vault_kubernetes_auth_backend_config" "vault_k8s_auth" {
  backend            = vault_auth_backend.k8s.path
  kubernetes_host    = data.aws_eks_cluster.cluster.endpoint
  kubernetes_ca_cert = base64decode(data.aws_eks_cluster.cluster.certificate_authority.0.data)
  token_reviewer_jwt = data.kubernetes_secret.external_secrets_token_secret.data.token
}

resource "vault_kubernetes_auth_backend_role" "k8s_external_secrets" {
  backend                          = vault_auth_backend.k8s.path
  role_name                        = "external-secrets"
  bound_service_account_names      = ["external-secrets"]
  bound_service_account_namespaces = ["*"]
  token_ttl                        = 86400
  token_policies                   = ["admin", "default"]
}
