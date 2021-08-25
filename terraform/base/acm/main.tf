resource "aws_acm_certificate" "cert" {
  domain_name       = "*.${var.hosted_zone_name}"
  validation_method = "DNS"

  tags = {
    Environment = "preprod"
  }

  lifecycle {
    create_before_destroy = true
  }
}