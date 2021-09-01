# todo construct these values
resource "aws_s3_bucket" "argo_artifact_bucket" {
  bucket = "k1-argo-artifacts-086f9d27715bf69624e84cda9a2801"
  acl    = "private"

  tags = {
    Name = "k1-argo-artifacts-086f9d27715bf69624e84cda9a2801"
  }
}

resource "aws_s3_bucket" "gitlab_backup_bucket" {
  bucket = "k1-gitlab-backup-086f9d27715bf69624e84cda9a2801"
  acl    = "private"

  tags = {
    Name = "k1-gitlab-backup-086f9d27715bf69624e84cda9a2801"
  }
}

resource "aws_s3_bucket" "chartmuseum_artifact_bucket" {
  bucket = "k1-chartmuseum-086f9d27715bf69624e84cda9a2801"
  acl    = "private"

  tags = {
    Name = "k1-chartmuseum-086f9d27715bf69624e84cda9a2801"
  }
}
