terraform {
  backend "s3" {
    bucket      = "<TF_STATE_BUCKET>"
    key         = "terraform/argocd/tfstate.tf"
    region      = "<AWS_DEFAULT_REGION>"
    encrypt     = true
  }
}
