resource "vault_jwt_auth_backend_role" "role" {
  backend         = var.vault_auth_backend
  role_name       = var.role_name
  token_policies  = var.token_policy_list
  user_claim      = "sub"
  role_type       = "oidc"
  bound_audiences = [var.app_name]
  allowed_redirect_uris = var.allowed_redirect_uris
}

variable "allowed_redirect_uris" {
  type = list(string)
  default = []
}

variable "app_name" {
  type = string 
}

variable "role_name" {
  type = string 
  default = "developer"
}

variable "token_policy_list" {
  type = list(string)
  default = []
}
variable "vault_auth_backend" {
  type = string
}