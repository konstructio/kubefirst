variable username { 
  type = string 
  description = "a distinct username that is unique to this user throughout the kubefirst ecosystem"
}
variable fullname {
  type = string 
  description = "example: Jane Doe"
}
variable email { 
  type = string
  description = "jane.doe@yourdomain.com"
}
variable is_admin { 
  default = false 
  description = "setting to true will add the user to the admins group, granting them admin access to our apps. setting to false will add the user to the developer group, granting them developer access to our apps"
}
variable enabled {
  default = true
  description = "setting to false allows you to destroy all resources so you can cleanly remove the user before removing them from terraform"
}
variable admins_group_id {
  type = string
  description = "for admin group assignment when is_admin"
}
variable developer_group_id {
  type = string
  description = "for developer group assignment when not is_admin"
}

resource "random_password" "user" {
  length           = 16
  special          = true
  override_special = "_#!"
}

resource "gitlab_user" "user" {
  count            = var.enabled ? 1 : 0
  name             = var.fullname
  username         = var.username
  password         = random_password.user.result
  email            = var.email
  is_admin         = var.is_admin
  projects_limit   = 100
  can_create_group = true
  is_external      = false
  reset_password   = false 
  # initial gitlab password are stored in vault. to allow gitlab to manage passwords,
  # you should remove `password` and change `reset_password` to true. however, you'll need to
  # enable gitlab email before setting to reset_password to true. see this link for config settings:
  # https://github.com/gitlabhq/omnibus-gitlab/blob/master/doc/settings/smtp.md
  # we didn't want this dependency on kubefirst's initial setup due to the variations in how companies
  # manage email. if you don't have company email available to you, the gmail integration works well.
}

resource "gitlab_group_membership" "user_admin_group" {
  count = var.enabled ? 1 : 0
  group_id     = var.is_admin ? var.admins_group_id : var.developer_group_id
  user_id      = gitlab_user.user[count.index].id
  access_level = "maintainer"
}

resource "vault_generic_secret" "user_password" {
  count = var.enabled ? 1 : 0 # keep secret in vault if user is enabled
  path = "users/${gitlab_user.user[count.index].username}"
  data_json = <<EOT
{
  "PASSWORD" : "${random_password.user.result}"
}
EOT
}
