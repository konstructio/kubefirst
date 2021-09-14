# 
# todo - need to talk about what else we want to be a part of repo creation?
# https://www.terraform.io/docs/providers/gitlab/r/project.html#argument-reference 
resource "gitlab_project" "repo" {
  name                   = var.repo_name
  archived               = var.archived
  visibility_level       = "private"
  default_branch         = var.default_branch
  namespace_id           = var.group_name
  import_url             = var.import_url
  initialize_with_readme = var.initialize_with_readme
  shared_runners_enabled = true
  # https://docs.gitlab.com/ee/user/packages/container_registry/
  only_allow_merge_if_all_discussions_are_resolved = true
  only_allow_merge_if_pipeline_succeeds            = var.only_allow_merge_if_pipeline_succeeds
  remove_source_branch_after_merge                 = var.remove_source_branch_after_merge
}

resource "aws_ecr_repository" "ecr_repo" {
  count                = var.create_ecr != true ? 0 : 1
  name                 = var.repo_name
  image_tag_mutability = "IMMUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }
}
