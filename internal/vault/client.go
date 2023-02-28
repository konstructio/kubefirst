package vault

import (
	vaultapi "github.com/hashicorp/vault/api"
)

var Conf VaultConfiguration = VaultConfiguration{
	Config: NewVault(),
}

func NewVault() vaultapi.Config {

	config := vaultapi.DefaultConfig()

	return *config
}
