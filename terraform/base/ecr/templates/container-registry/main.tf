resource "aws_ecr_repository" "ecr-repo" {
  name                 = var.repo_name
  image_tag_mutability = "IMMUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }
}

resource "aws_ecr_repository_policy" "ecr-policy" {
  repository = aws_ecr_repository.ecr-repo.name

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