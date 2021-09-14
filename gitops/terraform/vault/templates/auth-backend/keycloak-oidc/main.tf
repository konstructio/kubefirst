# locals {
#   url_region     = split("-", var.aws_region)[1]
#   oidc_provider  = "keycloak"
#   keycloak_realm = "kubefirst"
#   root_domain    = "kubefirst.com"
# }

# resource "vault_jwt_auth_backend" "keycloak_oidc" {
#   description        = "jwt backend for ${var.app_name} oidc authentication"
#   path               = "oidc"
#   type               = "oidc"
#   oidc_discovery_url = "https://${local.oidc_provider}-${local.url_region}.${var.aws_account_name}.${local.root_domain}/auth/realms/${local.keycloak_realm}"
#   oidc_client_id     = var.app_name
#   oidc_client_secret = var.oidc_client_secret
#   default_role       = var.default_role
# }

# variable "app_name" {
#   type = string
# }

# variable "aws_account_name" {
#   type = string
# }

# variable "aws_region" {
#   type = string
# }

# variable "default_role" {
#   type    = string
#   default = "developer"
# }

# variable "oidc_client_secret" {
#   type = string
# }

# variable "vault_oidc_roles" {
#   type = list(object({
#     role_name             = string
#     allowed_redirect_uris = list(string)
#     token_policy_list     = list(string)
#   }))
# }

# # todo roles with count
# module "oidc_role" {
#   source = "../../keycloak-oidc-client"
  
#   count = length(var.vault_oidc_roles)

#   allowed_redirect_uris = var.vault_oidc_roles[count.index].allowed_redirect_uris
#   app_name              = var.app_name
#   role_name             = var.vault_oidc_roles[count.index].role_name
#   token_policy_list     = concat(var.default_token_policy, var.vault_oidc_roles[count.index].token_policy_list)
#   vault_auth_backend    = vault_jwt_auth_backend.keycloak_oidc.path
# }

# variable "allowed_redirect_uris" {
#   type    = list(string)
#   default = []
# }

# variable "default_token_policy" {
#   type    = list(string)
#   default = ["default"]
# }

# variable "token_policy_list" {
#   type    = list(string)
#   default = []
# }

