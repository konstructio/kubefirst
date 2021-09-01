terraform {
  backend "s3" {
    bucket  = "k1-state-store-086f9d27715bf69624e84cda9a2801"
    key     = "terraform/gitlab/tfstate.tf"
    region  = "us-east-1"
    encrypt = true
  }
}

resource "gitlab_group" "kubefirst" {
  name                   = "kubefirst"
  path                   = "kubefirst"
  description            = "a private group for kubefirst repositories"
  request_access_enabled = false
  visibility_level       = "private"
}

module "metaphor" {
  depends_on = [
    gitlab_group.kubefirst
  ]
  source                                = "./templates/gitlab-repo"
  group_name                            = gitlab_group.kubefirst.id
  repo_name                             = "metaphor"
  create_ecr                            = true
  initialize_with_readme                = true
  import_url                            = "https://github.com/kubefirst/metaphor"
  only_allow_merge_if_pipeline_succeeds = false
  remove_source_branch_after_merge      = true
}

module "gitops" {
  depends_on = [
    gitlab_group.kubefirst
  ]
  source                                = "./templates/gitlab-repo"
  group_name                            = gitlab_group.kubefirst.id
  repo_name                             = "gitops"
  create_ecr                            = true
  initialize_with_readme                = true
  only_allow_merge_if_pipeline_succeeds = false
  remove_source_branch_after_merge      = true
}
