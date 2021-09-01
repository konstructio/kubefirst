terraform {
  backend "s3" {
    bucket  = "k1-state-store-086f9d27715bf69624e84cda9a2801"
    key     = "terraform/keycloak/tfstate.tf"
    region  = "us-east-1"
    encrypt = true
  }
}