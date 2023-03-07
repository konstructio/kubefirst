package vault

import (
	vaultapi "github.com/hashicorp/vault/api"
)

var Conf VaultConfiguration = VaultConfiguration{
	Config: NewVault(),
}

func NewVault() vaultapi.Config {
	config := vaultapi.DefaultConfig()
	config.Address = "http://127.0.0.1:8200"

	return *config
}
