terraform {
  required_providers {
    gitlab = {
      source = "gitlabhq/gitlab"
    }
    random = {
      source = "hashicorp/random"
    }
    vault = {
      source = "hashicorp/vault"
    }
  }
  required_version = ">= 0.13"
}
