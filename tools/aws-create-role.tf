#
# Terraform plan to create the administrator role that will be assumed
#
# Please read the comment within the file (not just this one) carefully to prevent any security issues within your organization!
#
# Replace the AWS account ID `111111111111` with yours.
#
# Ensure that the default values fit your needs (i.e., AWS region, role permission...)
#
# To run this plan:
# terraform init
# terraform apply
#

terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
      version = "4.67.0"
    }
  }
}

provider "aws" {
  region = "us-east-1"
}

resource "aws_iam_role" "assumed_admin" {

  # The role name
  name = "KubernetesAdmin"

  # The default session time is 1 hour, this set it to 12 hours for convenience. It's less annoying, but less secure, feel free to remove or change!
  max_session_duration = 43200

  #
  # Below is a permissive role not intended for long-term use.
  #
  # It grants all IAM users of the AWS account the ability to assume the role `KubernetesAdmin`, which we created and give the `AdministratorAccess` policy.
  #
  # The value `:root` grants assume to the whole account but you can replace it with your individual IAM ARN, or your role if appropriate.
  #
  # As a reminder, the value `111111111111` below should be replaced with your AWS account ID.
  #
  # Anyone with IAM can assume the role while it's in place like this. You can scope it down to your specific user, or across accounts, or whatever you need.
  #
  assume_role_policy   = <<EOF
  {
      "Version": "2012-10-17",
      "Statement": [
          {
              "Sid": "AllowAssumeRoleToIamUsers",
              "Effect": "Allow",
              "Principal": {
                  "AWS": "arn:aws:iam::111111111111:root"
              },
              "Action": "sts:AssumeRole"
          }
      ]
  }
  EOF
}

resource "aws_iam_role_policy_attachment" "assumed_admin_admin_policy" {
  role       = aws_iam_role.assumed_admin.name
  policy_arn = "arn:aws:iam::aws:policy/AdministratorAccess"
}
