
output "acm_validation_records" {
  description = "The domain to be validated"
  value       = aws_acm_certificate.cert.domain_validation_options
}

output "acm_certificate_arn" {
  description = "The arn of the certificate created"
  value       = aws_acm_certificate.cert.arn
}
