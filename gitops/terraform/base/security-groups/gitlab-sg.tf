resource "aws_security_group" "gitlab_security_group" {
  name                   = "gitlab"
  description            = "gitlab"
  vpc_id                 = var.kubefirst_vpc_id
  revoke_rules_on_delete = true
  tags = {
    Name = "gitlab"
  }

  ingress {
    description = "argo"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    description = "http"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    description      = "gitlab"
    from_port        = 80
    to_port          = 80
    protocol         = "tcp"
    ipv6_cidr_blocks = ["::/0"]
  }

  ingress {
    description = "atlantis"
    from_port   = 4141
    to_port     = 4141
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    description      = "atlantis"
    from_port        = 4141
    to_port          = 4141
    protocol         = "tcp"
    ipv6_cidr_blocks = ["::/0"]
  }

  ingress {
    description = "gitlab tls"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    description      = "gitlab tls"
    from_port        = 443
    to_port          = 443
    protocol         = "tcp"
    ipv6_cidr_blocks = ["::/0"]
  }

  egress {
    from_port   = 0
    to_port     = 65535
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
}
