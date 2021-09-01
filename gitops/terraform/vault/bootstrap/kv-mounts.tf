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
#   vaultMountPoint: kubernetes/starter-us-east-1
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
  "ARGOCD_AUTH_PASSWORD": "${var.argocd_auth_password}",
  "ARGOCD_AUTH_USERNAME": "admin",
  "ARGOCD_INSECURE": "false",
  "ARGOCD_SERVER": "argocd.starter.kubefirst.com:443",
  "ARGO_SERVER_URL": "argo.starter.kubefirst.com:443",
  "ATLANTIS_GITLAB_HOSTNAME": "gitlab.starter.kubefirst.com",
  "ATLANTIS_GITLAB_TOKEN": "${var.atlantis_gitlab_token}",
  "ATLANTIS_GITLAB_USER": "atlantis",
  "ATLANTIS_GITLAB_WEBHOOK_SECRET": "${var.atlantis_gitlab_webhook_secret}",
  "AWS_ACCESS_KEY_ID": "${var.aws_access_key_id}",
  "AWS_DEFAULT_REGION": "us-east-1",
  "AWS_ROLE_TO_ASSUME": "arn:aws:iam::126827061464:role/KubernetesAdmin",
  "AWS_SECRET_ACCESS_KEY": "${var.aws_secret_access_key}",
  "AWS_SESSION_NAME": "GitHubAction",
  "GITLAB_BASE_URL": "https://gitlab.starter.kubefirst.com",
  "GITLAB_TOKEN": "${var.gitlab_token}",
  "KEYCLOAK_CLIENT_ID": "admin-cli",
  "KEYCLOAK_PASSWORD": "${var.keycloak_password}",
  "KEYCLOAK_REALM": "master",
  "KEYCLOAK_URL": "https://keycloak.starter.kubefirst.com",
  "KEYCLOAK_USER": "gitlab-bot",
  "KUBECONFIG": "/.kube/config",
  "TF_VAR_aws_account_id": "<AWS_ACCOUNT_ID>",
  "TF_VAR_aws_region": "us-east-1",
  "TF_VAR_keycloak_admin_password": "${var.keycloak_admin_password}",
  "TF_VAR_keycloak_vault_oidc_client_secret": "${var.keycloak_vault_oidc_client_secret}",
  "TF_VAR_vault_redirect_uris": "[\"https://vault.starter.kubefirst.com/ui/vault/auth/oidc/oidc/callback\",\"http://localhost:8200/ui/vault/auth/oidc/oidc/callback\",\"http://localhost:8250/oidc/callback\",\"https://vault.starter.kubefirst.com:8250/oidc/callback\"]",
  "TF_VAR_argo_redirect_uris": "[\"https://argo.starter.kubefirst.com/argo/oauth2/callback\"]",
  "TF_VAR_argocd_redirect_uris": "[\"https://argocd.starter.kubefirst.com/auth/callback\",\"https://argocd.starter.kubefirst.com/applications\"]",
  "TF_VAR_gitlab_redirect_uris": "[\"https://gitlab.starter.kubefirst.com\"]",
  "VAULT_ADDR": "https://vault.starter.kubefirst.com",
  "VAULT_TOKEN": "${var.vault_token}"
}
EOT
}

resource "vault_mount" "users" {
  path        = "users"
  type        = "kv-v2"
  description = "kv v2 backend"
}
