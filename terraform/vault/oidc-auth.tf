variable "keycloak_vault_oidc_client_secret" {
  type = string
}

module "keycloak_vault" {
  source = "./templates/auth-backend/keycloak-oidc"

  app_name           = "vault"
  aws_account_name   = var.aws_account_name
  aws_region         = var.aws_region
  oidc_client_secret = var.keycloak_vault_oidc_client_secret

  # todo these redirect uri's are set as a TF_VAR_ and should be sourced from vault 
  # todo need to separate the roles and pass in the list of role objects?
  vault_oidc_roles = [{
    role_name = "default"
    #! todo var.vault_redirect_uris
    # https://gitlab.mgmt.kubefirst.com/kubefirst/gitops/-/blob/bd374ab317c964d0fdefe3c5836a8bd715c2e709/terraform/mgmt/aws/us-west-2/vault/oidc-auth.tf#L17-42
    allowed_redirect_uris = [
      "http://localhost:8250/oidc/callback",
      "https://vault-west.mgmt.kubefirst.com:8250/oidc/callback",
      "https://vault-west.mgmt.kubefirst.com/ui/vault/auth/oidc/oidc/callback",
      "http://localhost:8200/ui/vault/auth/oidc/oidc/callback"
    ]
    token_policy_list = []
    },
    {
      role_name = "developer"
      allowed_redirect_uris = [
        "http://localhost:8250/oidc/callback",
        "https://vault-west.mgmt.kubefirst.com:8250/oidc/callback",
        "https://vault-west.mgmt.kubefirst.com/ui/vault/auth/oidc/oidc/callback",
        "http://localhost:8200/ui/vault/auth/oidc/oidc/callback"
      ]
      token_policy_list = ["developer"]
    },
    {
      role_name = "admin"
      allowed_redirect_uris = [
        "http://localhost:8250/oidc/callback",
        "https://vault-west.mgmt.kubefirst.com:8250/oidc/callback",
        "https://vault-west.mgmt.kubefirst.com/ui/vault/auth/oidc/oidc/callback",
        "http://localhost:8200/ui/vault/auth/oidc/oidc/callback"
      ]
      token_policy_list = ["admin"]
  }]
}
