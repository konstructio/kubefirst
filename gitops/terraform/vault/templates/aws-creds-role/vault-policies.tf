resource "vault_policy" "vault-creds" {
  name = var.user_short_name

  policy = <<EOT
path "aws/creds/${var.user_short_name}" {
  capabilities = ["read"]
}
EOT
}
