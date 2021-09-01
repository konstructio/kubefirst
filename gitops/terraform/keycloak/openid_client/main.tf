terraform {
  required_providers {
    keycloak = {
      source  = "mrparkers/keycloak"
      version = ">= 2.0.0"
    }
  }
}
provider "keycloak" {
    client_id     = "admin-cli"
    username      = "gitlab-bot"
    password      = "ATU6VaGr6A"
    url           = "https://keycloak.<AWS_HOSTED_ZONE_NAME>"
}


resource "keycloak_openid_client" "openid_client" {
  realm_id  = var.keycloak_realm_id
  client_id = var.client_id

  name                  = var.client_id
  enabled               = var.enabled
  standard_flow_enabled = true

  access_type         = var.acccess_type
  valid_redirect_uris = var.redirect_uris

  login_theme = var.login_theme
}
resource "keycloak_openid_client_default_scopes" "client_default_scopes" {
  client_id = keycloak_openid_client.openid_client.id
  realm_id  = var.keycloak_realm_id

  default_scopes = [
    "profile",
    "email",
    "roles",
    "web-origins",
    var.scope,
  ]
}
resource "vault_generic_secret" "openid_client_secrets" {
  path = "secret/admin/oidc-clients/${keycloak_openid_client.openid_client.client_id}"

  data_json = <<EOT
{
  "${upper(keycloak_openid_client.openid_client.client_id)}_CLIENT_ID":   "${keycloak_openid_client.openid_client.client_id}",
  "${upper(keycloak_openid_client.openid_client.client_id)}_CLIENT_SECRET": "${keycloak_openid_client.openid_client.client_secret}"
}
EOT
}

#* 
#* variables
#* 
variable "scope" {
  type = string
}

variable "keycloak_realm_id" {
  type = string
}

variable "client_id" {
  type = string
}

variable "redirect_uris" {
  type = list(string)
}

variable "enabled" {
  type    = bool
  default = true
}

variable "acccess_type" {
  type    = string
  default = "CONFIDENTIAL"
}

variable "login_theme" {
  type    = string
  default = "keycloak"
}

#*
#*outputs
#*
output "keycloak_openid_client_id" {
  value = keycloak_openid_client.openid_client.client_id
}

output "keycloak_openid_client_secret" {
  value = keycloak_openid_client.openid_client.client_secret
}

