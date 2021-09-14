locals {
  aws_managed_admin_policies = ["arn:aws:iam::aws:policy/AdministratorAccess", "arn:aws:iam::aws:policy/IAMUserChangePassword", "arn:aws:iam::aws:policy/job-function/Billing"]
}

data "aws_iam_policy" "aws-managed-admin-policies" {
  count = length(local.aws_managed_admin_policies)
  arn   = local.aws_managed_admin_policies[count.index]
}
