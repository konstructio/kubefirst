resource "vault_aws_secret_backend_role" "admin-role" {
  backend         = var.secret_backend_path
  name            = var.user_short_name
  credential_type = "iam_user"

  policy_arns = data.aws_iam_policy.aws-managed-admin-policies[*].arn
}
