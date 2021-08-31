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

variable "argocd_redirect_uris" {
  type = list(string)
  default = [
    "https://argocd.starter.kubefirst.com/auth/callback",
    "https://argocd.starter.kubefirst.com/applications",
  ]
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


variable "argo_redirect_uris" {
  type = list(string)
  default = [
    "https://argo.starter.kubefirst.com/argo/oauth2/callback"
  ]
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


variable "argo_redirect_uris" {
  type = list(string)
  default = [
    "https://argo.starter.kubefirst.com/argo/oauth2/callback"
  ]
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
