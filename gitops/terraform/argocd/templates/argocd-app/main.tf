# data "vault_generic_secret" "argocd_secrets" {
#   path = "secret/k8s-mgmt/argocd/argocd-west-login" # TODO: change
# }

terraform {
  required_providers {
    argocd = {
      source  = "oboukili/argocd"
      version = "1.2.2"
    }
  }
}

provider "argocd" {}

variable "resource_path" {
  type = string
}

variable "destination_cluster" {
  type = string
  default = "https://kubernetes.default.svc"
}

variable "repo_url" {
  type    = string
  default = "https://gitlab.<AWS_HOSTED_ZONE_NAME>/kubefirst/gitops.git"
}

variable "destination_namespace" {
  type    = string
  default = "argocd"
}

variable "app_name" {
  type = string
}

variable "recurse_directory" {
  type    = bool
  default = false
}

resource "argocd_application" "app" {
  metadata {
    namespace = "argocd"
    name      = "app-${var.app_name}"
    labels = {
      test = "true"
    }
  }

  wait = false # todo true

  spec {
    source {
      repo_url        = var.repo_url
      path            = var.resource_path
      target_revision = "HEAD"

      directory {
        recurse = var.recurse_directory
      }
    }
    sync_policy {
      automated = {
        prune       = true
        self_heal   = true
        allow_empty = false
      }
      
      sync_options = ["Validate=false"]
      retry {
        limit = "5"
        backoff = {
          duration     = "30s"
          max_duration = "2m"
          factor       = "2"
        }
      }
    }

    destination {
      server    = var.destination_cluster
      namespace = var.destination_namespace
    }
  }
}
