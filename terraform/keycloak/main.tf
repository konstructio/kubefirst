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
    url           = "https://keycloak.starter.kubefirst.com"
}
resource "keycloak_realm" "kubefirst" {
  realm   = "kubefirst"
  enabled = true
}

data "keycloak_openid_client" "realm_management" {
  realm_id  = keycloak_realm.kubefirst.id
  client_id = "realm-management"
}

data "keycloak_role" "create_client" {
  realm_id  = keycloak_realm.kubefirst.id
  client_id = data.keycloak_openid_client.realm_management.id
  name      = "create-client"
}

resource "keycloak_generic_client_role_mapper" "realm_management_to_admin" {
  realm_id  = keycloak_realm.kubefirst.id
  client_id = data.keycloak_openid_client.realm_management.id
  role_id   = keycloak_role.admin_role.id
}

resource "keycloak_generic_client_role_mapper" "realm_management_to_developer" {
  realm_id  = keycloak_realm.kubefirst.id
  client_id = data.keycloak_openid_client.realm_management.id
  role_id   = keycloak_role.developer_role.id
}
resource "keycloak_role" "admin_role" {
  depends_on  = [keycloak_realm.kubefirst]
  realm_id    = keycloak_realm.kubefirst.id
  name        = "admin"
  description = "admin role"

  composite_roles = data.keycloak_role.realm_management_roles[*].id
}

resource "keycloak_role" "developer_role" {
  depends_on  = [keycloak_realm.kubefirst]
  realm_id    = keycloak_realm.kubefirst.id
  name        = "developer"
  description = "developer role"

  composite_roles = data.keycloak_role.realm_developer_roles[*].id
}

# developer roles
variable "realm_developer_roles" {
  type = list(string)
  # todo look up these roles and see if they are needed
  # https://wjw465150.gitbooks.io/keycloak-documentation/content/server_admin/topics/admin-console-permissions/per-realm.html
  default = [
    "query-clients",
    "query-groups",
    "query-realms",
    "query-users",
    "view-authorization",
    "view-clients",
    "view-events",
    "view-identity-providers",
    "view-realm",
    "view-users"
  ]

}
# use the data source
data "keycloak_role" "realm_developer_roles" {
  count     = length(var.realm_developer_roles)
  name      = var.realm_developer_roles[count.index]
  realm_id  = keycloak_realm.kubefirst.id
  client_id = data.keycloak_openid_client.realm_management.id
}

# admin roles
variable "realm_management_roles" {
  type = list(string)
  default = [
    "create-client",
    "impersonation",
    "manage-authorization",
    "manage-clients",
    "manage-events",
    "manage-identity-providers",
    "manage-realm",
    "manage-users",
    "query-clients",
    "query-groups",
    "query-realms",
    "query-users",
    "realm-admin",
    "view-authorization",
    "view-clients",
    "view-events",
    "view-identity-providers",
    "view-realm",
    "view-users"
  ]

}

data "keycloak_role" "realm_management_roles" {
  count     = length(var.realm_management_roles)
  name      = var.realm_management_roles[count.index]
  realm_id  = keycloak_realm.kubefirst.id
  client_id = data.keycloak_openid_client.realm_management.id
}
