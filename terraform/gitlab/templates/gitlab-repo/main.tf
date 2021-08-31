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

resource "aws_ecr_repository" "ecr-repo" {
  count                = var.create_ecr != true ? 0 : 1
  name                 = var.repo_name
  image_tag_mutability = "IMMUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }
}

resource "aws_ecr_repository_policy" "ecr-policy" {
  count      = var.create_ecr != true ? 0 : 1
  repository = aws_ecr_repository.ecr-repo[count.index].name

  policy = <<EOF
{
    "Version": "2008-10-17",
    "Statement": [
        {
            "Sid": "ecr pull policy for downstream aws accounts",
            "Effect": "Allow",
            "Principal": {
              "AWS": [
                "arn:aws:iam::${var.aws_account_id}:root"
              ]
            },
            "Action": [
                "ecr:GetDownloadUrlForLayer",
                "ecr:BatchGetImage"
            ]
        }
    ]
}
EOF
}

resource "gitlab_deploy_key" "deploy-key" {
  depends_on = [gitlab_project.repo]
  count      = var.create_deploy_key != true ? 0 : 1
  project    = "kubefirst/${var.repo_name}"
  title      = "kubefirst ${var.group_name} deploy-key"
  can_push   = true
  key        = file("${path.root}/terraform-ssh-key.pub")
}


output "gitlab_deploy_keys" {
  value = gitlab_deploy_key.deploy-key
}
