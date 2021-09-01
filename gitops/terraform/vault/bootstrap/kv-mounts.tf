resource "vault_mount" "secret" {
  path        = "secret"
  type        = "kv-v2"
  description = "the default vault kv v2 backend"
}

resource "vault_generic_secret" "test_secret" {
  path = "${vault_mount.secret.path}/test"

  data_json = <<EOT
{
  "TEST": "PASSED"
}
EOT
}
#! the following is example yaml to use the above generic secret
# todo use argo to verify this

# apiVersion: "kubernetes-client.io/v1"
# kind: ExternalSecret
# metadata:
#   name: test-secrets
# spec:
#   backendType: vault
#   vaultMountPoint: kubernetes/starter-<AWS_DEFAULT_REGION>
#   vaultRole: external-secrets
#   kvVersion: 2
#   data:
#   - name: TEST
#     key: secret/data/test
#     property: TEST


resource "vault_generic_secret" "kubefirst_secrets" {
  path = "${vault_mount.secret.path}/kubefirst-init"

  data_json = <<EOT
{
  "AWS_ACCESS_KEY_ID" : "${var.aws_access_key_id}",
  "AWS_SECRET_ACCESS_KEY" : "${var.aws_secret_access_key}",
  "AWS_HOSTED_ZONE_ID" : "${var.hosted_zone_id}",
  "AWS_ACCOUNT_ID" : "${var.aws_account_id}",
  "AWS_DEFAULT_REGION" : "${var.aws_region}",
  "EMAIL_ADDRESS" : "${var.email_address}",
  "VAULT_ADDR": "${var.vault_addr}",
  "VAULT_TOKEN" : "${var.vault_token}",
  "GITLAB_BOT_ROOT_PASSWORD" : "${var.gitlab_bot_root_password}" 
}
EOT
}

resource "random_password" "chartmuseum_user_password" {
  length           = 16
  special          = true
  override_special = "!@"
}


resource "vault_generic_secret" "chartmuseum_secrets" {
  path = "${vault_mount.secret.path}/chartmuseum"

  data_json = <<EOT
{
  "AWS_ACCESS_KEY_ID" : "${var.aws_access_key_id}",
  "AWS_SECRET_ACCESS_KEY" : "${var.aws_secret_access_key}",
  "BASIC_AUTH_USER" : "admin",
  "BASIC_AUTH_PASS" : "${random_password.chartmuseum_user_password.result}"
}
EOT
}
resource "vault_generic_secret" "gitlab_runner_secrets" {
  path = "${vault_mount.secret.path}/gitlab-runner"

  data_json = <<EOT
{
  "RUNNER_REGISTRATION_TOKEN": "${var.gitlab_runner_token}",
  "RUNNER_TOKEN": ""
}
EOT
}

resource "random_password" "keycloak_admin_password" {
  length           = 16
  special          = true
  override_special = "!@"
}
resource "random_password" "keycloak_mgmt_password" {
  length           = 16
  special          = true
  override_special = "!@"
}
resource "random_password" "keycloak_database_password" {
  length           = 16
  special          = true
  override_special = "!@"
}

resource "vault_generic_secret" "keycloak_secrets" {
  path = "${vault_mount.secret.path}/keycloak"

  data_json = <<EOT
{
  "KEYCLOAK_ADMIN_PASSWORD": "${random_password.keycloak_admin_password.result}",
  "KEYCLOAK_MANAGEMENT_PASSWORD": "${random_password.keycloak_mgmt_password.result}",
  "KEYCLOAK_DATABASE_PASSWORD": "${random_password.keycloak_database_password.result}",
  "POSTGRESQL_PASSWORD": "${random_password.keycloak_database_password.result}"
}
EOT
}
resource "vault_generic_secret" "atlantis_secrets" {
  path = "${vault_mount.secret.path}/atlantis"

  data_json = <<EOT
{
  "ARGOCD_AUTH_PASSWORD": "SgBgjrLCVUNcqgMY",
  "ARGOCD_AUTH_USERNAME": "admin",
  "ARGOCD_INSECURE": "false",
  "ARGOCD_SERVER": "argocd.<AWS_HOSTED_ZONE_NAME>:443",
  "ARGO_SERVER_URL": "argo.<AWS_HOSTED_ZONE_NAME>:443",
  "ATLANTIS_GITLAB_HOSTNAME": "gitlab.<AWS_HOSTED_ZONE_NAME>",
  "ATLANTIS_GITLAB_TOKEN": "gQevK69TPXSos5cXYC7m",
  "ATLANTIS_GITLAB_USER": "atlantis",
  "ATLANTIS_GITLAB_WEBHOOK_SECRET": "c75e7d48b854a36e13fb1d76a6eb5aa750a5e83a3ec6a0093413ed71d3313622",
  "AWS_ACCESS_KEY_ID": "",
  "AWS_DEFAULT_REGION": "<AWS_DEFAULT_REGION>",
  "AWS_ROLE_TO_ASSUME": "arn:aws:iam::126827061464:role/KubernetesAdmin",
  "AWS_SECRET_ACCESS_KEY": "",
  "AWS_SESSION_NAME": "GitHubAction",
  "GITLAB_BASE_URL": "https://gitlab.<AWS_HOSTED_ZONE_NAME>",
  "GITLAB_TOKEN": "gQevK69TPXSos5cXYC7m",
  "KEYCLOAK_CLIENT_ID": "admin-cli",
  "KEYCLOAK_PASSWORD": "d74c272854380f77594afcba",
  "KEYCLOAK_REALM": "master",
  "KEYCLOAK_URL": "https://keycloak.<AWS_HOSTED_ZONE_NAME>",
  "KEYCLOAK_USER": "gitlab-bot",
  "KUBECONFIG": "/.kube/config",
  "TF_VAR_aws_account_id": "126827061464",
  "TF_VAR_aws_account_name": "starter",
  "TF_VAR_aws_region": "<AWS_DEFAULT_REGION>",
  "TF_VAR_keycloak_admin_password": "d74c272854380f77594afcba",
  "TF_VAR_keycloak_vault_oidc_client_secret": "c949ef91-ff45-45be-a843-e1687c86c9bc",
  "TF_VAR_vault_redirect_uris": "[\"https://vault.<AWS_HOSTED_ZONE_NAME>/ui/vault/auth/oidc/oidc/callback\",\"http://localhost:8200/ui/vault/auth/oidc/oidc/callback\",\"http://localhost:8250/oidc/callback\",\"https://vault.<AWS_HOSTED_ZONE_NAME>:8250/oidc/callback\"]",
  "TF_VAR_argo_redirect_uris": "[\"https://argo.<AWS_HOSTED_ZONE_NAME>/argo/oauth2/callback\"]",
  "TF_VAR_argocd_redirect_uris": "[\"https://argocd.<AWS_HOSTED_ZONE_NAME>/auth/callback\",\"https://argocd.<AWS_HOSTED_ZONE_NAME>/applications\"]",
  "TF_VAR_gitlab_redirect_uris": "[\"https://gitlab.<AWS_HOSTED_ZONE_NAME>\"]",
  "VAULT_ADDR": "https://vault.<AWS_HOSTED_ZONE_NAME>",
  "VAULT_TOKEN": "s.8sufV5TDY9qcSLXJCwHqKBhP"
}
EOT
}

resource "vault_mount" "users" {
  path        = "users"
  type        = "kv-v2"
  description = "kv v2 backend"
}
