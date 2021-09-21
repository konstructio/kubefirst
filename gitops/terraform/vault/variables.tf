variable "aws_account_id" {
  type = string
}

variable "aws_region" {
  type = string
}

variable "aws_secret_access_key" {
  type = string
}
variable "vault_token" {
  type = string
}
variable "email_address" {
  type = string
}
variable "vault_addr" {
  type = string
}
variable "aws_access_key_id" {
  type = string
}
variable "gitlab_bot_root_password" {
  type = string
}

variable "hosted_zone_id" {
  type = string
}
variable "gitlab_runner_token" {
  type = string
}

variable "argocd_auth_password" {
  type = string
}
variable "atlantis_gitlab_token" {
  type = string
}
variable "atlantis_gitlab_webhook_secret" {
  type = string
}
variable "gitlab_token" {
  type = string
}
variable "keycloak_password" {
  type = string
}
variable "keycloak_admin_password" {
  type = string
}

variable "iam_user_arn" {
  type = string
}
variable "email_domain" {
  type = string
}
variable "hosted_zone_name" {
  type = string
}

variable "vault_redirect_uris" {
  type = list(string)
}