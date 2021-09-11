resource "random_string" "random" {
  length  = 6
  special = false
}

resource "aws_key_pair" "gitlab_public_key" {
  key_name   = "terraform-ssh-key-${random_string.random.result}"
  public_key = file("${path.root}/terraform-ssh-key.pub")
  
}

data "aws_ami_ids" "ubuntu" {
  owners = ["099720109477"]
  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd/ubuntu-bionic-18.04-amd64-server-20200112"]
  }
  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }
}

data "template_file" "gitlab_install_script" {
  template = file("${path.module}/scripts/install_gitlab.sh")
  vars = {
    EMAIL_DOMAIN             = var.email_domain
    GITLAB_URL               = var.gitlab_url
    GITLAB_BOT_ROOT_PASSWORD = var.gitlab_bot_root_password
  }
}

resource "aws_instance" "gitlab" {
  ami                         = data.aws_ami_ids.ubuntu.ids[0]
  associate_public_ip_address = true
  availability_zone           = "${var.aws_region}a"
  disable_api_termination     = false
  ebs_optimized               = false
  instance_type               = "t3.large"
  ipv6_address_count          = 0
  ipv6_addresses              = []
  key_name                    = aws_key_pair.gitlab_public_key.key_name
  monitoring                  = false
  source_dest_check           = true
  subnet_id                   = var.vpc_public_subnet
  tenancy                     = "default" # warning: replace if compliance requirements dictate non-shared resources
  user_data                   = data.template_file.gitlab_install_script.rendered
  lifecycle {
    ignore_changes = [
      user_data
    ]
  }
  volume_tags = {
    "Name" = "gitlab"
  }
  vpc_security_group_ids = [
    var.vpc_default_sg_id,
    var.gitlab_sg_id,
  ]
  metadata_options {
    http_endpoint               = "enabled"
    http_put_response_hop_limit = 1
    http_tokens                 = "optional"
  }
  root_block_device {
    delete_on_termination = true
    encrypted             = false
    volume_size           = 30
    volume_type           = "gp2"
  }
  tags = {
    "Name" = "gitlab"
  }
}
resource "aws_route53_record" "gitlab" {
  zone_id = var.hosted_zone_id
  name    = var.gitlab_url
  type    = "A"
  ttl     = "300"
  records = [aws_instance.gitlab.public_ip]

  depends_on = [aws_instance.gitlab]
}
