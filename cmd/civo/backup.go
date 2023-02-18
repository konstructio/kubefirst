package civo

import (
	"os"

	"github.com/kubefirst/kubefirst/internal/civo"
	"github.com/kubefirst/kubefirst/internal/ssl"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func backupCivoSSL(cmd *cobra.Command, args []string) error {

	clusterName := viper.GetString("flags.cluster-name")
	domainName := viper.GetString("flags.domain-name")
	githubOwner := viper.GetString("flags.github-owner")

	config := civo.GetConfig(clusterName, domainName, githubOwner)

	if _, err := os.Stat(config.SSLBackupDir + "/certificates"); os.IsNotExist(err) {
		// path/to/whatever does not exist
		paths := []string{config.SSLBackupDir + "/certificates", config.SSLBackupDir + "/clusterissuers", config.SSLBackupDir + "/secrets"}

		for _, path := range paths {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				log.Info().Msgf("checking path: %s", path)
				err := os.MkdirAll(path, os.ModePerm)
				if err != nil {
					log.Info().Msg("directory already exists, continuing")
				}
			}
		}
	}

	err := ssl.Backup(config.SSLBackupDir, domainNameFlag, config.K1Dir, config.Kubeconfig)
	if err != nil {
		log.Info().Msg("error backing up ssl resources")
		return err
	}
	return nil
}
