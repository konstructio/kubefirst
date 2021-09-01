terraform {
  backend "s3" {
    bucket  = "kubefirst-demo-dbb09532cff3c1057a58577e87bc35"
    key     = "terraform/keycloak/tfstate.tf"
    region  = "<AWS_DEFAULT_REGION>"
    encrypt = true
  }
}