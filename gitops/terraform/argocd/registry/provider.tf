terraform {
  required_providers {
    argocd = {
      source  = "oboukili/argocd"
      version = "1.2.2"
    }
  }
}

provider "argocd" {}
