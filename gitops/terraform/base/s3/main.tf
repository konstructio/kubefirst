# todo construct these values
resource "aws_s3_bucket" "argo_artifact_bucket" {
  bucket = "<ARGO_ARTIFACT_BUCKET>"
  acl    = "private"

  tags = {
    Name = "<ARGO_ARTIFACT_BUCKET>"
  }
}

resource "aws_s3_bucket" "gitlab_backup_bucket" {
  bucket = "<GITLAB_BACKUP_BUCKET>"
  acl    = "private"

  tags = {
    Name = "<GITLAB_BACKUP_BUCKET>"
  }
}

resource "aws_s3_bucket" "chartmuseum_artifact_bucket" {
  bucket = "<CHARTMUSEUM_BUCKET>"
  acl    = "private"

  tags = {
    Name = "<CHARTMUSEUM_BUCKET>"
  }
}
