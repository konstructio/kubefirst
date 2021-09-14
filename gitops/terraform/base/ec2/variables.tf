variable "vpc_default_sg_id" {
  type = string
}

variable "vpc_public_subnet" {
  type = string
}

variable "gitlab_sg_id" {
  type = string
}

variable "aws_region" {
  type = string
}

variable "gitlab_bot_root_password" {
  type = string
}

variable "gitlab_url" {
  type = string
}

variable "email_domain" {
  type = string
}

variable "hosted_zone_id" {
  type = string
}
