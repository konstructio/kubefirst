variable "region" {
  type = string
}

variable "aws_account_id" {
  type = string
}

variable "terraform_state_store_bucket_name" {
  type = string
}

variable "gitlab_url" {
  type = string
}
variable "hosted_zone_name" {
  type = string
}

variable "email_domain" {
  type = string
}

# todo clean up
variable "gitlab_hostname" {
  type = string
  # # if you change this value also change it in ec2/scripts/install_gitlab.sh
  # default = "gitlab-kubefrst.preprod.kubefirst.com" # todo gitlab-kubefirst
}
