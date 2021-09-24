variable "developer_users" {
  type = list(object({
    username = string
    first_name = string
    last_name = string
  }))
  default = [{
    username = "dev1"
    first_name = "Developer"
    last_name = "One"
  },
  {
    username = "dev2"
    first_name = "Developer"
    last_name = "Two"
  }]
}

resource "random_string" "developer_user_password" {
  count = length(var.developer_users)
  length           = 16
  special          = true
  override_special = "/@$#"
}

output "developer_usernames" {
  value = keycloak_user.developer_user[*].username
}

resource "keycloak_user" "developer_user" {
  count = length(var.developer_users)
  realm_id = keycloak_realm.kubefirst.id
  username = var.developer_users[count.index].username
  enabled  = true

  email      = "${var.developer_users[count.index].username}@${var.email_domain}"
  first_name = var.developer_users[count.index].first_name
  last_name  = var.developer_users[count.index].last_name

  initial_password {
    value     = random_string.developer_user_password[count.index].result
    temporary = true
  }
}

resource "vault_generic_secret" "developer_user_password" {
  count = length(var.developer_users)
  path  = "users/${keycloak_user.developer_user[count.index].username}"

  data_json = <<EOT
{
  "initial-password": "${random_string.developer_user_password[count.index].result}",
  "username": "${keycloak_user.developer_user[count.index].username}",
  "email": "${keycloak_user.developer_user[count.index].email}"
}
EOT
}

#! the ids from users[count.index] not module
resource "keycloak_group_memberships" "developer_members" {
  realm_id = keycloak_realm.kubefirst.id
  group_id = keycloak_group.developer_group.id

  members = keycloak_user.developer_user[*].username
}
