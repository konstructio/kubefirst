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

  aws_account_id = var.aws_account_id
  aws_region = var.aws_region
  aws_account_name = var.aws_account_name
}

# todo leftover terraform needs evaluation
# module "leftover" {
#   source = "./leftover"

#   aws_account_id = var.aws_account_id
#   aws_region = var.aws_region
#   aws_account_name = var.aws_account_name
# }
