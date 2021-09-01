# module "aws-role-gitlab-runner-gitlab-runner" {
#   source = "./templates/aws-creds-role"

#   user_short_name     = "gitlab-runner-gitlab-runner"
#   role                = "admin"
#   secret_backend_path = vault_aws_secret_backend.aws.path
# }
