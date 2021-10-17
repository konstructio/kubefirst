resource "gitlab_group" "developer" {
  name        = "developer"
  path        = "developer"
  description = "developer group"
}

module "developer_one" {
  source   = "./templates/oidc-user"
  admins_group_id    = gitlab_group.admins.id
  developer_group_id = gitlab_group.developer.id
  username           = "developer1"
  fullname           = "Developer One"
  email              = "developer1@yourcompany.com"
}

module "developer_two" {
  source   = "./templates/oidc-user"
  admins_group_id    = gitlab_group.admins.id
  developer_group_id = gitlab_group.developer.id
  username           = "developer2"
  fullname           = "Developer Two"
  email              = "developer2@yourcompany.com"
}
