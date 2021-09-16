terraform {
  backend "s3" {
    bucket  = "<TF_STATE_BUCKET>"
    key     = "terraform/gitlab/tfstate.tf"
    region  = "<AWS_DEFAULT_REGION>"
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
  initialize_with_readme                = false
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
  initialize_with_readme                = false
  only_allow_merge_if_pipeline_succeeds = false
  remove_source_branch_after_merge      = true
}

resource "gitlab_project_hook" "atlantis" {
  depends_on = [
    module.gitops
  ]
  project               = "kubefirst/gitops"
  url                   = "https://atlantis.<AWS_HOSTED_ZONE_NAME>/events"
  merge_requests_events = true
  push_events           = true
  note_events           = true
}
