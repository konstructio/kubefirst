terraform {
  backend "s3" {
    bucket  = "@S3_BUCKET_NAME@"
    key     = "terraform/tfstate.tf"
    region  = "@AWS_DEFAULT_REGION@"
    encrypt = true
  }
}

# terraform {
#   backend "s3" {
#     bucket  = "@S3_BUCKET_NAME@"
#     key     = "terraform/tfstate.tf"
#     region  = "@AWS_DEFAULT_REGION@"
#     encrypt = true
#   }
# }

provider "aws" {

  region = var.region
}
module "eks" {
  source = "./eks"

  aws_account_id = var.aws_account_id
  cluster_name   = "k8s-preprod"
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

  region                   = var.region
  vpc_default_sg_id        = module.eks.preprod_vpc_default_sg
  vpc_public_subnet        = module.eks.preprod_vpc_public_subnets[0]
  gitlab_sg_id             = module.security-groups.gitlab_sg_id
  gitlab_url               = var.gitlab_url
  gitlab_bot_root_password = var.gitlab_bot_root_password
  hosted_zone_id           = data.aws_route53_zone.hosted_zone.zone_id
  email_domain             = var.email_domain
}

module "security-groups" {
  source = "./security-groups"

  preprod_vpc_id = module.eks.preprod_vpc_id
}

module "route53" {
  source = "./route53"

  route53_acm_verification_records = module.acm.acm_validation_records
  hosted_zone_id                   = data.aws_route53_zone.hosted_zone.zone_id
}

module "acm" {
  source = "./acm"

  hosted_zone_name = var.hosted_zone_name
}

module "ecr" {
  source = "./ecr"

  aws_account_id = var.aws_account_id
}
