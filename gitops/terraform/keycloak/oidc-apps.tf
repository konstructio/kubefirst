# client scopes can all bind to this
resource "keycloak_openid_client_scope" "group_scope" {
  realm_id               = keycloak_realm.kubefirst.id
  name                   = "groups"
  description            = "When requested, this scope will map a user's group memberships to a claim"
  include_in_token_scope = true
  gui_order              = 1
}

resource "keycloak_openid_group_membership_protocol_mapper" "group_membership_mapper" {
  realm_id        = keycloak_realm.kubefirst.id
  client_scope_id = keycloak_openid_client_scope.group_scope.id
  name            = "groups"

  claim_name = "groups"
}


variable "argo_redirect_uris" {
  type = list(string)
}

module "argo" {
  source = "./openid_client"

  client_id         = "argo"
  keycloak_realm_id = keycloak_realm.kubefirst.id
  redirect_uris     = var.argo_redirect_uris
  scope             = keycloak_openid_client_scope.group_scope.name
}

output "argo_openid_client" {
  value = module.argo
}


variable "argocd_redirect_uris" {
  type = list(string)
}
module "argocd" {
  source = "./openid_client"

  client_id         = "argocd"
  keycloak_realm_id = keycloak_realm.kubefirst.id
  redirect_uris     = var.argocd_redirect_uris
  scope             = keycloak_openid_client_scope.group_scope.name
}

output "argocd_openid_client" {
  value = module.argocd
}

variable "gitlab_redirect_uris" {
  type = list(string)
}

module "gitlab" {
  source = "./openid_client"

  client_id         = "gitlab"
  keycloak_realm_id = keycloak_realm.kubefirst.id
  redirect_uris     = var.gitlab_redirect_uris
  scope             = keycloak_openid_client_scope.group_scope.name
}

output "gitlab_openid_client" {
  value = module.gitlab
}

module "vault" {
  source = "./openid_client"

  client_id         = "vault"
  keycloak_realm_id = keycloak_realm.kubefirst.id
  redirect_uris     = var.vault_redirect_uris
  scope             = keycloak_openid_client_scope.group_scope.name
}

output "vault_openid_client" {
  value = module.vault
}
variable "vault_redirect_uris" {
  type = list(string)
}
