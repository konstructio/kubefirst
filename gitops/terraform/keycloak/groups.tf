# #! todo blocker!! need to figure out how to restrict users from joining new groups in the ui 
# # todo need to add a developer group and add everyone to the developer group by default
resource "keycloak_group" "kubefirst_group" {
  realm_id = keycloak_realm.kubefirst.id
  name     = "kubefirst"
}

resource "keycloak_group" "admin_group" {
  realm_id  = keycloak_realm.kubefirst.id
  parent_id = keycloak_group.kubefirst_group.id
  name      = "admins"
}

resource "keycloak_group_roles" "admin_roles" {
  realm_id = keycloak_realm.kubefirst.id
  group_id = keycloak_group.admin_group.id

  role_ids = [
    keycloak_role.admin_role.id,
  ]
}

resource "keycloak_group" "developer_group" {
  realm_id  = keycloak_realm.kubefirst.id
  parent_id = keycloak_group.kubefirst_group.id
  name      = "developers"
}

resource "keycloak_group_roles" "developer_roles" {
  realm_id = keycloak_realm.kubefirst.id
  group_id = keycloak_group.developer_group.id

  role_ids = [
    keycloak_role.developer_role.id,
  ]
}
