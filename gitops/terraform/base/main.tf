terraform {
  backend "s3" {
    bucket  = "<TF_STATE_BUCKET>"
    key     = "terraform/base/tfstate.tf"
    region  = "<AWS_DEFAULT_REGION>"
    encrypt = true
  }
}


provider "aws" {

  region = var.aws_region
}
module "eks" {
  source = "./eks"

  aws_account_id = var.aws_account_id
  cluster_name   = "kubefirst"
  iam_user_arn   = var.iam_user_arn
}

module "kms" {
  aws_account_id = var.aws_account_id
  source = "./kms"
}

module "dynamodb" {
  source = "./dynamodb"
}

module "s3" {
  source = "./s3"
}

data "aws_route53_zone" "hosted_zone" {
  name = var.hosted_zone_name
}

module "ec2" {
  source = "./ec2"

  aws_region               = var.aws_region
  vpc_default_sg_id        = module.eks.kubefirst_vpc_default_sg
  vpc_public_subnet        = module.eks.kubefirst_vpc_public_subnets[0]
  gitlab_sg_id             = module.security_groups.gitlab_sg_id
  gitlab_url               = var.gitlab_url
  gitlab_bot_root_password = var.gitlab_bot_root_password
  hosted_zone_id           = data.aws_route53_zone.hosted_zone.zone_id
  email_domain             = var.email_domain
}

module "security_groups" {
  source = "./security-groups"

  kubefirst_vpc_id = module.eks.kubefirst_vpc_id
}

output "vault_unseal_kms_key" {
  value = module.kms.vault_unseal_kms_key
}
