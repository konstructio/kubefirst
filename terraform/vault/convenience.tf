# # todo probably need to move this into a manual module. we should be able to target this particular set of IAC since
# # * it probably doesn't have much place here long term
# data "kubernetes_secret" "argocd" {
#   metadata {
#     name = "argocd-initial-admin-secret"
#     namespace = "argocd"
#   }
# }

# resource "vault_generic_secret" "argocd_secrets" {
#   path = "secret/${var.aws_account_name}/aws/${var.aws_region}/argocd/argocd"

#   data_json = <<EOT
# {
#   "ARGOCD_PASSWORD":   "${data.kubernetes_secret.argocd.data["password"]}",
#   "ARGOCD_USER": "admin"
# }
# EOT
# }

# data "kubernetes_secret" "argo" {
#   metadata {
#     name = "argo-secrets"
#     namespace = "argo"
#   }
# }

# resource "vault_generic_secret" "argo_secrets" {
#   path = "secret/${var.aws_account_name}/aws/${var.aws_region}/argo/argo"

#   data_json = <<EOT
# {
#   "aws-access-key-id": "${data.kubernetes_secret.argo.data["aws-access-key-id"]}",
#   "aws-secret-access-key": "${data.kubernetes_secret.argo.data["aws-secret-access-key"]}",
#   "db-password": "${data.kubernetes_secret.argo.data["db-password"]}",
#   "db-username": "${data.kubernetes_secret.argo.data["db-username"]}",
#   "keycloak-client-id": "${data.kubernetes_secret.argo.data["keycloak-client-id"]}",
#   "keycloak-client-secret": "${data.kubernetes_secret.argo.data["keycloak-client-secret"]}"
# }
# EOT
# }

# data "kubernetes_secret" "chartmuseum" {
#   metadata {
#     name = "chartmuseum-secrets"
#     namespace = "chartmuseum"
#   }
# }

# resource "vault_generic_secret" "chartmuseum_secrets" {
#   path = "secret/${var.aws_account_name}/aws/${var.aws_region}/chartmuseum/chartmuseum"

#   data_json = <<EOT
# {
#   "aws-access-key-id": "${data.kubernetes_secret.chartmuseum.data["aws-access-key-id"]}",
#   "aws-secret-access-key": "${data.kubernetes_secret.chartmuseum.data["aws-secret-access-key"]}",
#   "basic-auth-pass": "${data.kubernetes_secret.chartmuseum.data["basic-auth-pass"]}",
#   "basic-auth-user": "${data.kubernetes_secret.chartmuseum.data["basic-auth-user"]}"
# }
# EOT
# }


# data "kubernetes_secret" "ci" {
#   metadata {
#     name = "ci-secrets"
#     namespace = "argo"
#   }
# }

# resource "vault_generic_secret" "ci_secrets" {
#   path = "secret/${var.aws_account_name}/aws/${var.aws_region}/argo/ci-secrets"

#   data_json = <<EOT
# {
#   "PERSONAL_ACCESS_TOKEN": "${data.kubernetes_secret.ci.data["PERSONAL_ACCESS_TOKEN"]}",
#   "aws-access-key-id": "${data.kubernetes_secret.ci.data["aws-access-key-id"]}",
#   "aws-secret-access-key": "${data.kubernetes_secret.ci.data["aws-secret-access-key"]}",
#   "personal-access-token": "${data.kubernetes_secret.ci.data["personal-access-token"]}",
#   "username": "${data.kubernetes_secret.ci.data["username"]}"
# }
# EOT
# }

# # data "kubernetes_secret" "metaphor" {
# #   metadata {
# #     name = "metaphor-development"
# #     namespace = "development"
# #   }
# # }

# # resource "vault_generic_secret" "metaphor_secrets" {
# #   path = "secret/${var.aws_account_name}/aws/${var.aws_region}/development/metaphor-development"

# #   data_json = <<EOT
# # {
# #   "SECRET_ONE": "${data.kubernetes_secret.metaphor.data["SECRET_ONE"]}",
# #   "SECRET_TWO": "${data.kubernetes_secret.metaphor.data["SECRET_TWO"]}"
# # }
# # EOT
# # }

# # data "kubernetes_secret" "gitlab_runner" {
# #   metadata {
# #     name = "gitlab-runner-gitlab-runner"
# #     namespace = "gitlab-runner"
# #   }
# # }
# # hacked this one, it wasnt able to locate the secret :thinking-face:
# resource "vault_generic_secret" "gitlab_runner_secrets" {
#   path = "secret/${var.aws_account_name}/aws/${var.aws_region}/gitlab-runner/gitlab-runner"

#   data_json = <<EOT
# {
#   "runner-registration-token": "${base64decode("Mnhvc2NXQlZoUlp6X1MyZFphaGc=")}",
#   "runner-token": ""
# }
# EOT
# }
