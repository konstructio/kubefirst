package vault

import (
	vaultapi "github.com/hashicorp/vault/api"
	"github.com/rs/zerolog/log"
)

func (conf *VaultConfiguration) AutoUnseal() (*vaultapi.InitResponse, error) {

	vaultClient, err := vaultapi.NewClient(&conf.Config)
	if err != nil {
		return &vaultapi.InitResponse{}, err
	}
	log.Info().Msg("created vault client, initializing vault with auto unseal")

	initResponse, err := vaultClient.Sys().Init(&vaultapi.InitRequest{
		RecoveryShares:    RecoveryShares,
		RecoveryThreshold: RecoveryThreshold,
	})
	log.Info().Msg("vault initialization complete")

	return initResponse, err
}
