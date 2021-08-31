terraform {
  backend "s3" {
    bucket  = "kubefirst-demo-dbb09532cff3c1057a58577e87bc35"
    key     = "terraform/gitlab/tfstate.tf"
    region  = "us-east-1"
    encrypt = true
  }
}

resource "gitlab_group" "kubefirst" {
  name                   = "kubefirst-starter"
  path                   = "kubefirst-starter"
  description            = "a private group for kubefirst repositories"
  request_access_enabled = false
  visibility_level       = "private"
}

module "chubbo" {
  depends_on = [
    gitlab_group.kubefirst
  ]
  source                                = "./templates/gitlab-repo"
  repo_name                             = "chubbo"
  create_ecr                            = true
  only_allow_merge_if_pipeline_succeeds = false
  remove_source_branch_after_merge      = true
  group_name                            = gitlab_group.kubefirst.id
  aws_account_id                        = var.aws_account_id
}

module "metaphor" {
  depends_on = [
    gitlab_group.kubefirst
  ]
  source                                = "./templates/gitlab-repo"
  repo_name                             = "metaphor"
  create_ecr                            = true
  initialize_with_readme                = true
  import_url                            = "https://github.com/kubefirst/metaphor"
  create_deploy_key                     = true
  only_allow_merge_if_pipeline_succeeds = false
  remove_source_branch_after_merge      = true
  group_name                            = gitlab_group.kubefirst.id
  aws_account_id                        = var.aws_account_id
}

module "nebulous" {
  depends_on = [
    gitlab_group.kubefirst
  ]
  source                                = "./templates/gitlab-repo"
  repo_name                             = "nebulous"
  create_ecr                            = true
  initialize_with_readme                = true
  create_deploy_key                     = true
  only_allow_merge_if_pipeline_succeeds = false
  remove_source_branch_after_merge      = true
  group_name                            = gitlab_group.kubefirst.id
  aws_account_id                        = var.aws_account_id
}
