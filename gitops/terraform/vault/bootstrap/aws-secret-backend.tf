resource "vault_aws_secret_backend" "aws" {
  path                      = "${var.aws_account_name}-${var.aws_region}"
  default_lease_ttl_seconds = 86400  # 1 day by default
  max_lease_ttl_seconds     = 432000 # 5 days is the max lease
}
