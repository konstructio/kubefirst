# todo remove hardcode
terraform {
  backend "s3" {
    bucket  = "k1-state-store-086f9d27715bf69624e84cda9a2801"
    key     = "terraform/tfstate.tf"
    region  = "us-east-1"
    encrypt = true
  }
}

# terraform {
#   backend "s3" {
#     bucket  = "kubefirst-demo-aa26801359a3d171219f6752a867ac"
#     key     = "terraform/tfstate.tf"
#     region  = "us-east-1"
#     encrypt = true
#   }
# }

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
