# todo construct these values
resource "aws_s3_bucket" "argo_artifact_bucket" {
  bucket = "kubefirst-starter-argo-artifacts"
  acl    = "private"

  tags = {
    Name = "kubefirst-starter-argo-artifacts"
  }
}

resource "aws_s3_bucket" "gitlab_backup_bucket" {
  bucket = "kubefirst-starter-gitlab-backup-bucket"
  acl    = "private"

  tags = {
    Name = "kubefirst-starter-gitlab-backup-bucket"
  }
}

resource "aws_s3_bucket" "chartmuseum_artifact_bucket" {
  bucket = "kubefirst-starter-chartmuseum-artifact-bucket"
  acl    = "private"

  tags = {
    Name = "kubefirst-starter-chartmuseum-artifact-bucket"
  }
}
