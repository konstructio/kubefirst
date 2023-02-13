package civo

import (
	"fmt"
	"os"

	"github.com/kubefirst/kubefirst/internal/ssl"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func backupCivoSSL(cmd *cobra.Command, args []string) error {

	domainName := viper.GetString("domain-name")
	k1Dir := viper.GetString("kubefirst.k1-dir")
	kubeconfigPath := viper.GetString("kubefirst.kubeconfig-path")
	backupDir := fmt.Sprintf("%s/ssl/%s", k1Dir, domainName)

	if _, err := os.Stat(backupDir + "/certificates"); os.IsNotExist(err) {
		// path/to/whatever does not exist
		paths := []string{backupDir + "/certificates", backupDir + "/clusterissuers", backupDir + "/secrets"}

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

	err := ssl.Backup(backupDir, domainName, k1Dir, kubeconfigPath)
	if err != nil {
		log.Info().Msg("error backing up ssl resources")
		return err
	}
	return nil
}
