terraform {
  backend "s3" {
    bucket  = "kubefirst-demo-dbb09532cff3c1057a58577e87bc35"
    key     = "terraform/vault/tfstate.tf"
    region  = "us-east-1"
    encrypt = true
  }
}

module "bootstrap" {
  source = "./bootstrap"

  aws_account_id           = var.aws_account_id
  aws_region               = var.aws_region
  aws_account_name         = var.aws_account_name
  aws_secret_access_key    = var.aws_secret_access_key
  vault_token              = var.vault_token
  email_address            = var.email_address
  vault_addr               = var.vault_addr
  aws_access_key_id        = var.aws_access_key_id
  gitlab_bot_root_password = var.gitlab_bot_root_password
  hosted_zone_id = var.hosted_zone_id
  gitlab_runner_token = var.gitlab_runner_token
}

# todo leftover terraform needs evaluation
# module "leftover" {
#   source = "./leftover"

#   aws_account_id = var.aws_account_id
#   aws_region = var.aws_region
#   aws_account_name = var.aws_account_name
# }
