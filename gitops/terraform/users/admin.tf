resource "gitlab_group" "admins" {
  name        = "admins"
  path        = "admins"
  description = "admins group"
}

module "admin_one" {
  source   = "./templates/oidc-user"
  admins_group_id    = gitlab_group.admins.id
  developer_group_id = gitlab_group.developer.id
  username           = "admin1"
  fullname           = "Admin One"
  email              = "admin1@yourcompany.com"
  is_admin           = true
}

module "admin_two" {
  source   = "./templates/oidc-user"
  admins_group_id    = gitlab_group.admins.id
  developer_group_id = gitlab_group.developer.id
  username           = "admin2"
  fullname           = "Admin Two"
  email              = "admin2@yourcompany.com"
  is_admin           = true
}
