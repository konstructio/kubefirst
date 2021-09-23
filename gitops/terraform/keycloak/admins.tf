# todo update
# 1. create a new module for an admin user
# 2. add the new username output to the list ouptut `all_admins`

variable "admin_users" {
  type = list(object({
    username = string
    first_name = string
    last_name = string
  }))
  default = [{
    username = "admin1"
    first_name = "Admin"
    last_name = "One"
  },
  {
    username = "admin2"
    first_name = "Admin"
    last_name = "Two"
  }]
}
resource "random_string" "admin_user_password" {
  count = length(var.admin_users)
  length           = 16
  special          = true
  override_special = "/@$#"
}

output "admin_usernames" {
  value = keycloak_user.admin_user[*].username
}

variable "team" {
  type = string
  default = "ops"
}

variable "email_domain" {
  type    = string
  default = "kubefirst.com"
}


resource "keycloak_user" "admin_user" {
  count = length(var.admin_users)
  realm_id = keycloak_realm.kubefirst.id
  username = var.admin_users[count.index].username
  enabled  = true

  email      = "${var.admin_users[count.index].username}@${var.email_domain}"
  first_name = var.developer_users[count.index].first_name
  last_name  = var.developer_users[count.index].last_name

  initial_password {
    value     = random_string.admin_user_password[count.index].result
    temporary = true
  }
}


# todo: review region addition with kubefirst team
resource "vault_generic_secret" "admin_user_password" {
  count = length(var.admin_users)
  path  = "users/${keycloak_user.admin_user[count.index].username}"

  data_json = <<EOT
{
  "initial-password": "${random_string.admin_user_password[count.index].result}",
  "username": "${keycloak_user.admin_user[count.index].username}",
  "email": "${keycloak_user.admin_user[count.index].email}"
}
EOT
}

resource "keycloak_group_memberships" "admin_members" {
  realm_id = keycloak_realm.kubefirst.id
  group_id = keycloak_group.admin_group.id

  members = keycloak_user.admin_user[*].username
}
