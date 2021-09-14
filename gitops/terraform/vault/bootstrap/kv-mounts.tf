resource "vault_mount" "secret" {
  path        = "secret"
  type        = "kv-v2"
  description = "the default vault kv v2 backend"
}

resource "vault_mount" "users" {
  path        = "users"
  type        = "kv-v2"
  description = "kv v2 backend"
}


