data "vault_generic_secret" "vault_oidc_secrets" {
  path = "secret/admin/oidc/vault"
}

variable "vault_redirect_uris" {
  type = list(string)
}

resource "vault_jwt_auth_backend" "gitlab_oidc" {
  description        = "jwt backend for vault oidc authentication"
  path               = "oidc"
  type               = "oidc"
  oidc_discovery_url = "https://gitlab.<AWS_HOSTED_ZONE_NAME>"
  oidc_client_id     = data.vault_generic_secret.vault_oidc_secrets.data["application_id"]
  oidc_client_secret = data.vault_generic_secret.vault_oidc_secrets.data["secret"]
  default_role       = "developer"
}

resource "vault_jwt_auth_backend_role" "admin" {
  backend         = vault_jwt_auth_backend.gitlab_oidc.path
  role_name       = "admin"
  token_policies  = ["admin"]
  user_claim      = "sub"
  role_type       = "oidc"
  #bound_audiences = [data.vault_generic_secret.vault_oidc_secrets.data["application_id"]]
  oidc_scopes = ["openid"]
  bound_claims = {
    groups = "admins"
  }

  allowed_redirect_uris = var.vault_redirect_uris
}

resource "vault_jwt_auth_backend_role" "developer" {
  backend         = vault_jwt_auth_backend.gitlab_oidc.path
  role_name       = "developer"
  token_policies  = ["developer"]
  user_claim      = "sub"
  role_type       = "oidc"
  #bound_audiences = [data.vault_generic_secret.vault_oidc_secrets.data["application_id"]]
  oidc_scopes = ["openid"]
  bound_claims = {
    groups = "developer"
  }
  allowed_redirect_uris = var.vault_redirect_uris
}
